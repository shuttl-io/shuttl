package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/shuttl-ai/cli/log"
)

// MessageType represents the type of IPC message
type MessageType string

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeEvent    MessageType = "event"
	MessageTypeError    MessageType = "error"
)

type ErrorObject struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *ErrorObject) Error() string {
	return fmt.Sprintf("%s, message: %s", e.Code, e.Message)
}

func (e *ErrorObject) String() string {
	return fmt.Sprintf("%s, message: %s", e.Code, e.Message)
}

// Message represents a message sent to/from the Shuttl application
type Message struct {
	Type      MessageType     `json:"type"`
	ID        string          `json:"id,omitempty"`
	Success   bool            `json:"success,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Result    json.RawMessage `json:"result"`
	ErrorObj  *ErrorObject    `json:"errorObj,omitempty"`
}

type Request struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Body   any    `json:"body,omitempty"`
}

// OutputLine represents a line of output from stdout or stderr
type OutputLine struct {
	Source    string    // "stdout" or "stderr"
	Content   string    // The raw line content
	Message   *Message  // Parsed message if JSON, nil otherwise
	Timestamp time.Time // When the line was received
	Err       error     // Any error parsing the line
}

// ClientState represents the current state of the IPC client
type ClientState int

const (
	StateIdle ClientState = iota
	StateRunning
	StateStopping
	StateStopped
)

// Client manages IPC communication with a Shuttl application
type SpecialChannel struct {
	outputChan chan OutputLine
	errChan    chan error
}

type Client struct {
	command []string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser

	// Named pipe paths and handles (using containerd/fifo for cross-platform support)
	pipeDir          string
	requestPipePath  string
	responsePipePath string
	requestPipe      io.ReadWriteCloser
	responsePipe     io.ReadWriteCloser

	// Channels for output
	outputChan chan OutputLine
	errChan    chan error

	// State management
	state   ClientState
	stateMu sync.RWMutex

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelCauseFunc

	// Wait group for goroutines
	wg sync.WaitGroup

	// Mutex for sending messages
	sendMu            sync.Mutex
	specialChannels   map[string]*SpecialChannel
	specialChannelsMu sync.RWMutex

	// Stderr buffer for capturing subprocess stderr output
	stderrBuffer   []string
	stderrBufferMu sync.Mutex
}

// NewClient creates a new IPC client for the given command and arguments
func NewClient(command []string) *Client {
	ctx, cancel := context.WithCancelCause(context.Background())

	return &Client{
		command:         command,
		outputChan:      make(chan OutputLine, 100),
		errChan:         make(chan error, 10),
		state:           StateIdle,
		ctx:             ctx,
		cancel:          cancel,
		specialChannels: make(map[string]*SpecialChannel),
		stderrBuffer:    make([]string, 0),
	}
}

// Start starts the Shuttl application subprocess
func (c *Client) Start() error {
	c.stateMu.Lock()
	if c.state != StateIdle && c.state != StateStopped {
		c.stateMu.Unlock()
		return fmt.Errorf("client is already running or stopping")
	}
	c.state = StateRunning
	c.stateMu.Unlock()

	if len(c.command) == 0 {
		c.setState(StateStopped)
		return fmt.Errorf("no command specified")
	}

	// Create named pipes for IPC
	if err := c.createNamedPipes(); err != nil {
		c.setState(StateStopped)
		return fmt.Errorf("failed to create named pipes: %w", err)
	}

	// Create the command with arguments
	c.cmd = exec.CommandContext(c.ctx, c.command[0], c.command[1:]...)
	c.cmd.Env = append(os.Environ(),
		"_SHUTTL_CONTROL=true",
		fmt.Sprintf("_SHUTTL_REQUEST_PIPE=%s", c.requestPipePath),
		fmt.Sprintf("_SHUTTL_RESPONSE_PIPE=%s", c.responsePipePath),
	)

	// Get stderr pipe for error output
	var err error
	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		c.cleanupNamedPipes()
		c.setState(StateStopped)
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the process
	if err := c.cmd.Start(); err != nil {
		c.cleanupNamedPipes()
		c.setState(StateStopped)
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Open named pipes after process starts (platform-specific)
	if err := c.openPipes(c.ctx); err != nil {
		c.cleanupNamedPipes()
		c.setState(StateStopped)
		return fmt.Errorf("failed to open pipes: %w", err)
	}

	// Start reading goroutines
	c.wg.Add(2)
	go c.readOutput(c.getResponsePipeReader(), "response_pipe")
	go c.readOutput(c.stderr, "stderr")

	// Start process monitor goroutine
	c.wg.Add(1)
	go c.monitorProcess()

	return nil
}

// readOutput reads lines from a pipe and sends them to the output channel
func (c *Client) readOutput(pipe io.ReadCloser, source string) {
	defer c.wg.Done()

	scanner := bufio.NewScanner(pipe)
	// Increase buffer size for large messages
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		select {
		case <-c.ctx.Done():
			return
		default:
			line := scanner.Text()
			log.DebugWithPrefix("IPC", "Received line from %s: %s", source, line)

			// Collect stderr lines into the buffer for later display if process exits early
			if source == "stderr" {
				c.stderrBufferMu.Lock()
				c.stderrBuffer = append(c.stderrBuffer, line)
				c.stderrBufferMu.Unlock()
			}

			output := OutputLine{
				Source:    source,
				Content:   line,
				Timestamp: time.Now(),
			}

			// Try to parse as JSON message
			var msg Message
			err := json.Unmarshal([]byte(line), &msg)
			if err == nil {
				output.Message = &msg
			}

			c.specialChannelsMu.RLock()
			channel, ok := c.specialChannels[msg.ID]
			c.specialChannelsMu.RUnlock()

			outputChan := c.outputChan
			errChan := c.errChan

			if ok {
				outputChan = channel.outputChan
				errChan = channel.errChan
			}
			if err != nil {
				errChan <- err
			}

			// Send to output channel for stdout or response_pipe, error channel for stderr
			if source == "stdout" || source == "response_pipe" {
				select {
				case outputChan <- output:
				default:
					// Channel full, drop oldest
					select {
					case <-outputChan:
					default:
					}
					outputChan <- output

					if err := scanner.Err(); err != nil && c.ctx.Err() == nil {
						errChan <- fmt.Errorf("%s scanner error: %w", source, err)
					}
				}
			} else {
				// stderr - send to error channel
				select {
				case errChan <- fmt.Errorf("%s: %s", source, line):
				default:
				}
			}
		}
	}

}

// monitorProcess monitors the subprocess and handles cleanup
func (c *Client) monitorProcess() {
	defer c.wg.Done()
	defer func() {
		c.wg.Wait()
		log.Info("Closing output and error channels")
		close(c.outputChan)
		close(c.errChan)
		for key := range c.specialChannels {
			c.CloseSpecialChannel(key)
		}
		// Clean up named pipes when process exits
		c.cleanupNamedPipes()
	}()

	err := c.cmd.Wait()
	c.setState(StateStopped)
	log.Warn("Subprocess exited: %v", err)

	if err != nil && c.ctx.Err() == nil {
		// Output the error and stderr to the user
		log.Error("Subprocess exited unexpectedly: %v", err)

		// Print the stderr output if we collected any
		c.stderrBufferMu.Lock()
		if len(c.stderrBuffer) > 0 {
			log.Error("Subprocess stderr output:\n")
			log.Error("----------------------------------------\n")
			for _, line := range c.stderrBuffer {
				log.Error("%s\n", line)
			}
			log.Error("----------------------------------------\n")
		}
		c.stderrBufferMu.Unlock()
		c.cancel(fmt.Errorf("subprocess exited unexpectedly: %w", err))
		return
	}

	// Close the output channel after process exits

}

// Send sends a message to the Shuttl application (blocking)
func (c *Client) Send(req Request) error {
	c.stateMu.RLock()
	if c.state != StateRunning {
		c.stateMu.RUnlock()
		return fmt.Errorf("client is not running")
	}
	c.stateMu.RUnlock()

	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	// Set timestamp if not set
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Write message with newline to the request pipe
	if c.requestPipe == nil {
		return fmt.Errorf("request pipe is not open")
	}
	_, err = c.requestPipe.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write to request pipe: %w", err)
	}

	return nil
}

// SendPayload sends a payload with the specified type (convenience method)
func (c *Client) SendPayload(msgType MessageType, id string, payload interface{}) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return c.Send(Request{
		ID:     id,
		Method: string(msgType),
		Body:   payloadData,
	})
}

// SendAsync sends a message asynchronously and returns a channel for the result
func (c *Client) SendAsync(req Request) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- c.Send(req)
		close(errCh)
	}()

	return errCh
}

func (c *Client) SendAsyncWithResult(ctx context.Context, req Request) (chan error, chan OutputLine) {
	errCh := make(chan error, 1)
	outputChan := make(chan OutputLine, 10)

	specialChannels := &SpecialChannel{
		outputChan: outputChan,
		errChan:    errCh,
	}
	c.specialChannelsMu.Lock()
	c.specialChannels[req.ID] = specialChannels
	c.specialChannelsMu.Unlock()

	c.Send(req)

	return errCh, outputChan
}

func (c *Client) CloseSpecialChannel(id string) {
	c.specialChannelsMu.Lock()
	channel, ok := c.specialChannels[id]
	if ok {
		close(channel.outputChan)
		close(channel.errChan)
		delete(c.specialChannels, id)
	}
	c.specialChannelsMu.Unlock()
}

func (c *Client) SendAndWaitForResponse(ctx context.Context, req Request) (OutputLine, error) {
	errCh, outputChan := c.SendAsyncWithResult(ctx, req)
	defer c.CloseSpecialChannel(req.ID)

	log.Debug("Waiting for response: %+v", req)
	for {
		select {
		case <-ctx.Done():
			return OutputLine{}, ctx.Err()
		case err, ok := <-errCh:
			if !ok {
				return OutputLine{}, nil
			}
			if err != nil {
				return OutputLine{}, err
			}
		case output, ok := <-outputChan:
			if !ok {
				return OutputLine{}, nil
			}
			return output, nil
		}
	}
}

func getID(messageType MessageType) string {
	return fmt.Sprintf("%s:%s", messageType, time.Now().Format("20060102150405"))
}

// Receive receives the next output line (blocking)
func (c *Client) Receive() (OutputLine, error) {
	select {
	case output, ok := <-c.outputChan:
		if !ok {
			return OutputLine{}, fmt.Errorf("output channel closed")
		}
		return output, nil
	case <-c.ctx.Done():
		return OutputLine{}, c.ctx.Err()
	}
}

func (c *Client) RecieveWithID(id string, ctx context.Context) (OutputLine, error) {
	for {
		select {
		case <-ctx.Done():
			return OutputLine{}, ctx.Err()
		default:
			msg, err := c.Receive()
			if err != nil {
				return OutputLine{}, err
			}
			if msg.Message.ID == id {
				return msg, nil
			}
			go func() {
				c.outputChan <- msg
			}()
		}
	}
}

// ReceiveTimeout receives the next output line with a timeout (blocking with timeout)
func (c *Client) ReceiveTimeout(timeout time.Duration) (OutputLine, error) {
	select {
	case output, ok := <-c.outputChan:
		if !ok {
			return OutputLine{}, fmt.Errorf("output channel closed")
		}
		return output, nil
	case <-time.After(timeout):
		return OutputLine{}, fmt.Errorf("receive timeout after %v", timeout)
	case <-c.ctx.Done():
		return OutputLine{}, c.ctx.Err()
	}
}

// TryReceive attempts to receive without blocking (non-blocking)
func (c *Client) TryReceive() (OutputLine, bool) {
	select {
	case output, ok := <-c.outputChan:
		if !ok {
			return OutputLine{}, false
		}
		return output, true
	default:
		return OutputLine{}, false
	}
}

// Output returns the output channel for non-blocking reads
func (c *Client) Output() <-chan OutputLine {
	return c.outputChan
}

// Errors returns the error channel for monitoring subprocess errors
func (c *Client) Errors() <-chan error {
	return c.errChan
}

// Stop gracefully stops the subprocess by closing the request pipe first
func (c *Client) Stop() error {
	c.stateMu.Lock()
	if c.state != StateRunning {
		c.stateMu.Unlock()
		return nil
	}
	c.state = StateStopping
	c.stateMu.Unlock()

	// Close request pipe to signal the subprocess to exit gracefully
	if c.requestPipe != nil {
		c.requestPipe.Close()
		c.requestPipe = nil
	}

	// Wait for subprocess to exit with timeout
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Subprocess exited gracefully
		c.cleanupNamedPipes()
	case <-time.After(5 * time.Second):
		// Force kill if not exited
		return c.Kill()
	}

	return nil
}

// Kill forcefully terminates the subprocess
func (c *Client) Kill() error {
	c.stateMu.Lock()
	if c.state == StateStopped || c.state == StateIdle {
		c.stateMu.Unlock()
		return nil
	}
	c.state = StateStopping
	c.stateMu.Unlock()

	// Cancel context to stop all goroutines
	c.cancel(nil)

	// Kill the process
	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Wait for goroutines to finish
	c.wg.Wait()
	c.setState(StateStopped)

	// Clean up named pipes
	c.cleanupNamedPipes()

	return nil
}

// Close stops the subprocess and cleans up resources
func (c *Client) Close() error {
	if err := c.Stop(); err != nil {
		// Try force kill if graceful stop fails
		return c.Kill()
	}
	return nil
}

// State returns the current state of the client
func (c *Client) State() ClientState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

// IsRunning returns true if the client is currently running
func (c *Client) IsRunning() bool {
	return c.State() == StateRunning
}

// setState sets the state with proper locking
func (c *Client) setState(state ClientState) {
	c.stateMu.Lock()
	c.state = state
	c.stateMu.Unlock()
}

// Wait blocks until the subprocess exits
func (c *Client) Wait() error {
	c.wg.Wait()
	return nil
}

// WaitTimeout waits for the subprocess to exit with a timeout
func (c *Client) WaitTimeout(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("wait timeout after %v", timeout)
	}
}

// Context returns the client's context for external cancellation handling
func (c *Client) Context() context.Context {
	return c.ctx
}

// ProcessID returns the PID of the subprocess, or 0 if not running
func (c *Client) ProcessID() int {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Pid
	}
	return 0
}

// Command returns the command array
func (c *Client) Command() []string {
	return c.command
}

// String returns the client state as a string
func (s ClientState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// ParseCommand parses a command string into an array of arguments
// It handles quoted strings and escapes properly
func ParseCommand(cmdStr string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)
	escaped := false

	for _, r := range cmdStr {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if inQuote {
			if r == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}

		if r == '"' || r == '\'' {
			inQuote = true
			quoteChar = r
			continue
		}

		if r == ' ' || r == '\t' {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

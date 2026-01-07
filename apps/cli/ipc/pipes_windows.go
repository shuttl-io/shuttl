//go:build windows

package ipc

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/google/uuid"
	"github.com/shuttl-ai/cli/log"
)

// pipeConn wraps a net.Conn to implement io.ReadWriteCloser
type pipeConn struct {
	net.Conn
}

// createNamedPipes creates Windows named pipe paths for IPC
func (c *Client) createNamedPipes() error {
	// Generate unique pipe names using UUID
	pipeID := uuid.New().String()
	c.requestPipePath = fmt.Sprintf(`\\.\pipe\shuttl-request-%s`, pipeID)
	c.responsePipePath = fmt.Sprintf(`\\.\pipe\shuttl-response-%s`, pipeID)

	log.DebugWithPrefix("IPC", "Created named pipe paths: request=%s, response=%s", c.requestPipePath, c.responsePipePath)
	return nil
}

// openPipes opens Windows named pipes for communication
// On Windows, we connect as a client - the subprocess will create the pipe servers
func (c *Client) openPipes(ctx context.Context) error {
	var err error

	// Wait for pipes to be available and connect
	// The subprocess will create these pipes, so we need to retry connection
	maxRetries := 100
	retryDelay := 50 * time.Millisecond

	// Connect to response pipe for reading
	var responseConn net.Conn
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		responseConn, err = winio.DialPipe(c.responsePipePath, nil)
		if err == nil {
			break
		}
		time.Sleep(retryDelay)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to response pipe after %d retries: %w", maxRetries, err)
	}
	c.responsePipe = &pipeConn{responseConn}

	// Connect to request pipe for writing
	var requestConn net.Conn
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			c.responsePipe.Close()
			return ctx.Err()
		default:
		}
		
		requestConn, err = winio.DialPipe(c.requestPipePath, nil)
		if err == nil {
			break
		}
		time.Sleep(retryDelay)
	}
	if err != nil {
		c.responsePipe.Close()
		return fmt.Errorf("failed to connect to request pipe after %d retries: %w", maxRetries, err)
	}
	c.requestPipe = &pipeConn{requestConn}

	return nil
}

// getResponsePipeReader returns a reader for the response pipe
func (c *Client) getResponsePipeReader() io.ReadCloser {
	return c.responsePipe
}

// cleanupNamedPipes cleans up Windows named pipes
func (c *Client) cleanupNamedPipes() {
	if c.requestPipe != nil {
		c.requestPipe.Close()
		c.requestPipe = nil
	}
	if c.responsePipe != nil {
		c.responsePipe.Close()
		c.responsePipe = nil
	}
	// Windows named pipes are cleaned up automatically when all handles are closed
	log.DebugWithPrefix("IPC", "Cleaned up named pipes")
}


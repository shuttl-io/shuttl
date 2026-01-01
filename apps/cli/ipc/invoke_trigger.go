package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shuttl-ai/cli/log"
)

// TriggerRequest represents the request payload for invoking a trigger
type TriggerRequest struct {
	AgentName   string                 `json:"agentName"`
	TriggerName string                 `json:"triggerName"`
	TriggerType string                 `json:"triggerType"`
	ThreadID    string                 `json:"threadId,omitempty"`
	HTTPRequest *SerializedHTTPRequest `json:"httpRequest"`
}

// SerializedHTTPRequest represents the serialized HTTP request
type SerializedHTTPRequest struct {
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Headers     map[string][]string `json:"headers"`
	Query       map[string][]string `json:"query"`
	Body        json.RawMessage     `json:"body,omitempty"`
	ContentType string              `json:"contentType"`
	RemoteAddr  string              `json:"remoteAddr"`
	Host        string              `json:"host"`
	Proto       string              `json:"proto"`
	Timestamp   time.Time           `json:"timestamp"`
}

// TriggerResponse represents the response from invoking a trigger
type TriggerResponse struct {
	Success   bool              `json:"success"`
	ThreadID  string            `json:"threadId,omitempty"`
	Events    []json.RawMessage `json:"events,omitempty"`
	Result    json.RawMessage   `json:"result,omitempty"`
	Error     string            `json:"error,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// TriggerStreamEvent represents a streaming event from a trigger invocation
type TriggerStreamEvent struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	ThreadID  string          `json:"threadId,omitempty"`
	Completed bool            `json:"completed"`
	Error     string          `json:"error,omitempty"`
}

// InvokeTrigger invokes a trigger via IPC and waits for completion
// It collects all events and returns them in the response
func (c *Client) InvokeTrigger(ctx context.Context, req TriggerRequest) (*TriggerResponse, error) {
	c.wg.Add(1)
	defer c.wg.Done()

	id := fmt.Sprintf("invoke_trigger:%s:%s:%d", req.AgentName, req.TriggerName, time.Now().UnixNano())

	ipcReq := Request{
		ID:     id,
		Method: "invokeTrigger",
		Body:   req,
	}

	log.Debug("Invoking trigger: %s/%s", req.AgentName, req.TriggerName)

	errCh, resultCh := c.SendAsyncWithResult(ctx, ipcReq)
	defer c.CloseSpecialChannel(id)

	var events []json.RawMessage
	var threadID string
	var finalResult json.RawMessage

	for {
		select {
		case <-ctx.Done():
			return &TriggerResponse{
				Success:   false,
				Error:     ctx.Err().Error(),
				Events:    events,
				ThreadID:  threadID,
				Timestamp: time.Now(),
			}, ctx.Err()

		case err := <-errCh:
			if err != nil {
				return &TriggerResponse{
					Success:   false,
					Error:     err.Error(),
					Events:    events,
					ThreadID:  threadID,
					Timestamp: time.Now(),
				}, nil
			}

		case result, ok := <-resultCh:
			if !ok {
				// Channel closed unexpectedly
				return &TriggerResponse{
					Success:   false,
					Error:     "result channel closed unexpectedly",
					Events:    events,
					ThreadID:  threadID,
					Timestamp: time.Now(),
				}, nil
			}

			// Check for error response
			if !result.Message.Success {
				errMsg := "trigger invocation failed"
				if result.Message.ErrorObj != nil {
					errMsg = result.Message.ErrorObj.Error()
				}
				return &TriggerResponse{
					Success:   false,
					Error:     errMsg,
					Events:    events,
					ThreadID:  threadID,
					Timestamp: time.Now(),
				}, nil
			}

			// Collect the event
			if result.Message.Result != nil {
				if result.Message.Type == "output_text" {
					events = append(events, result.Message.Result)
				}
			}

			// Check if this is a completion message
			msgType := string(result.Message.Type)
			if msgType == "status" || msgType == "" {
				var status struct {
					ThreadID string `json:"threadId"`
					Status   string `json:"status"`
				}
				if err := json.Unmarshal(result.Message.Result, &status); err == nil {
					if status.ThreadID != "" {
						threadID = status.ThreadID
					}
					if status.Status == "completed" || status.Status == "invoked" {
						finalResult = result.Message.Result
						return &TriggerResponse{
							Success:   true,
							ThreadID:  threadID,
							Events:    events,
							Result:    finalResult,
							Timestamp: time.Now(),
						}, nil
					}
				}
			}

			// Also check for threadId in other message types
			var threadCheck struct {
				ThreadID string `json:"threadId"`
			}
			if err := json.Unmarshal(result.Message.Result, &threadCheck); err == nil && threadCheck.ThreadID != "" {
				threadID = threadCheck.ThreadID
			}
		}
	}
}

// InvokeTriggerAsync invokes a trigger asynchronously and returns channels for results
func (c *Client) InvokeTriggerAsync(ctx context.Context, req TriggerRequest) (chan *TriggerResponse, chan error) {
	id := fmt.Sprintf("invoke_trigger:%s:%s:%d", req.AgentName, req.TriggerName, time.Now().UnixNano())

	ipcReq := Request{
		ID:     id,
		Method: "invokeTrigger",
		Body:   req,
	}

	responseCh := make(chan *TriggerResponse, 1)
	errCh, resultCh := c.SendAsyncWithResult(ctx, ipcReq)

	go func() {
		defer close(responseCh)
		defer c.CloseSpecialChannel(id)

		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		case result, ok := <-resultCh:
			if !ok {
				errCh <- fmt.Errorf("result channel closed")
				return
			}

			if !result.Message.Success {
				errMsg := "trigger invocation failed"
				if result.Message.ErrorObj != nil {
					errMsg = result.Message.ErrorObj.Error()
				}
				responseCh <- &TriggerResponse{
					Success:   false,
					Error:     errMsg,
					Timestamp: time.Now(),
				}
				return
			}

			responseCh <- &TriggerResponse{
				Success:   true,
				Result:    result.Message.Result,
				Timestamp: time.Now(),
			}
		}
	}()

	return responseCh, errCh
}

// InvokeTriggerStreaming invokes a trigger and streams events back as they arrive
// This is similar to StartChat but for triggers
func (c *Client) InvokeTriggerStreaming(ctx context.Context, req TriggerRequest) (chan *TriggerStreamEvent, chan error) {
	id := fmt.Sprintf("invoke_trigger:%s:%s:%d", req.AgentName, req.TriggerName, time.Now().UnixNano())

	ipcReq := Request{
		ID:     id,
		Method: "invokeTrigger",
		Body:   req,
	}

	log.Debug("Invoking trigger (streaming): %s/%s", req.AgentName, req.TriggerName)

	eventCh := make(chan *TriggerStreamEvent, 10)
	errCh, resultCh := c.SendAsyncWithResult(ctx, ipcReq)

	go func() {
		defer close(eventCh)
		defer c.CloseSpecialChannel(id)

		for {
			select {
			case <-ctx.Done():
				eventCh <- &TriggerStreamEvent{
					Type:      "error",
					Error:     ctx.Err().Error(),
					Completed: true,
				}
				return

			case result, ok := <-resultCh:
				if !ok {
					eventCh <- &TriggerStreamEvent{
						Type:      "error",
						Error:     "result channel closed unexpectedly",
						Completed: true,
					}
					return
				}

				// Check for error response
				if !result.Message.Success {
					errMsg := "trigger invocation failed"
					if result.Message.ErrorObj != nil {
						errMsg = result.Message.ErrorObj.Error()
					}
					eventCh <- &TriggerStreamEvent{
						Type:      "error",
						Error:     errMsg,
						Completed: true,
					}
					return
				}

				// Parse the message type and create appropriate event
				event := &TriggerStreamEvent{
					Type: string(result.Message.Type),
					Data: result.Message.Result,
				}

				// Check if this is a completion message
				if result.Message.Type == "status" || result.Message.Type == "" {
					// Try to parse as status to check for completion
					var status struct {
						ThreadID string `json:"threadId"`
						Status   string `json:"status"`
					}
					if err := json.Unmarshal(result.Message.Result, &status); err == nil {
						event.ThreadID = status.ThreadID
						if status.Status == "completed" || status.Status == "invoked" {
							event.Completed = true
							eventCh <- event
							return
						}
					}
				}

				eventCh <- event
			}
		}
	}()

	return eventCh, errCh
}

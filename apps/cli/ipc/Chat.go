package ipc

import (
	"context"
	"fmt"

	"encoding/json"

	"github.com/shuttl-ai/cli/log"
)

// BaseResponse represents the common fields in every IPC message
type BaseResponse struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"` // Delay parsing this
}

// --- Specific Result Payload Types ---

// ToolCallResult (type: "tool_call")
type ToolCallResult struct {
	TypeName string `json:"typeName"`
	ToolCall struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"` // Map or raw string
		CallID    string          `json:"callId"`
	} `json:"toolCall"`
}

// ToolCallCompletedResult (type: "tool_calls_completed")
type ToolCallCompletedResult []struct {
	Output string `json:"output"`
	Type   string `json:"type"`
	CallID string `json:"call_id"`
}

// TextDeltaResult (type: "output_text_delta")
type TextDeltaResult struct {
	TypeName        string `json:"typeName"`
	OutputTextDelta struct {
		Delta          string `json:"delta"`
		SequenceNumber int    `json:"sequenceNumber"`
	} `json:"outputTextDelta"`
}

// FinalOutputResult (type: "output_text")
type FinalOutputResult struct {
	OutputText struct {
		Text string `json:"text"`
	} `json:"outputText"`
}

// StatusResult (no type field, or simple status)
type StatusResult struct {
	ThreadID string `json:"threadId"`
	Status   string `json:"status"`
}

// ParsedResult holds the parsed result from a chat response
type ChatParsedResult struct {
	Type               string
	ToolCall           *ToolCallResult
	ToolCallsCompleted *ToolCallCompletedResult
	TextDelta          *TextDeltaResult
	FinalOutput        *FinalOutputResult
	Status             *StatusResult
	ResponseRequested  *json.RawMessage
}

func (c ChatParsedResult) String() string {
	if c.Type == "output_text_delta" {
		str, err := json.Marshal(c.TextDelta)
		if err != nil {
			return fmt.Sprintf("Error marshalling TextDelta: %v", err)
		}
		return string(str)
	}
	if c.Type == "output_text" {
		str, err := json.Marshal(c.FinalOutput)
		if err != nil {
			return fmt.Sprintf("Error marshalling FinalOutput: %v", err)
		}
		return string(str)
	}
	if c.Type == "tool_call" {
		str, err := json.Marshal(c.ToolCall)
		if err != nil {
			return fmt.Sprintf("Error marshalling ToolCall: %v", err)
		}
		return string(str)
	}
	if c.Type == "tool_calls_completed" {
		str, err := json.Marshal(c.ToolCallsCompleted)
		if err != nil {
			return fmt.Sprintf("Error marshalling ToolCallsCompleted: %v", err)
		}
		return string(str)
	}
	if c.Type == "response.requested" {
		str, err := json.Marshal(c.ResponseRequested)
		if err != nil {
			return fmt.Sprintf("Error marshalling ResponseRequested: %v", err)
		}
		return string(str)
	}
	if c.Type == "status" {
		str, err := json.Marshal(c.Status)
		if err != nil {
			return fmt.Sprintf("Error marshalling Status: %v", err)
		}
		return string(str)
	}
	return fmt.Sprintf("Unknown type: %s", c.Type)
}

// parseResult parses the raw JSON result into the appropriate struct based on the message type
func parseResult(msgType MessageType, raw json.RawMessage) (*ChatParsedResult, error) {
	result := &ChatParsedResult{Type: string(msgType)}

	switch msgType {
	case "tool_call":
		var toolCall ToolCallResult
		if err := json.Unmarshal(raw, &toolCall); err != nil {
			return nil, err
		}
		result.ToolCall = &toolCall

	case "tool_calls_completed":
		var completed ToolCallCompletedResult
		if err := json.Unmarshal(raw, &completed); err != nil {
			return nil, err
		}
		result.ToolCallsCompleted = &completed

	case "output_text_delta":
		var delta TextDeltaResult
		if err := json.Unmarshal(raw, &delta); err != nil {
			return nil, err
		}
		result.TextDelta = &delta

	case "output_text":
		var output FinalOutputResult
		if err := json.Unmarshal(raw, &output); err != nil {
			return nil, err
		}
		result.FinalOutput = &output
	case "response.requested":
		result.ResponseRequested = &raw
	default:
		// Try to parse as status
		var status StatusResult
		if err := json.Unmarshal(raw, &status); err == nil {
			result.Status = &status
		}
	}
	return result, nil
}

func (c *Client) StartChat(ctx context.Context, agentID string, message string) (chan *ChatParsedResult, chan error) {
	id := getID(RequestChat)
	req := Request{
		ID:     id,
		Method: "invokeAgent",
		Body:   ChatRequest{Agent: agentID, Prompt: message},
	}
	parsedResultCh := make(chan *ChatParsedResult, 10)
	errCh, resultCh := c.SendAsyncWithResult(ctx, req)
	go func() {
		defer close(parsedResultCh)
		defer c.CloseSpecialChannel(id)
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case result, ok := <-resultCh:
				if !ok {
					errCh <- fmt.Errorf("result channel closed")
					return
				}

				parsed, err := parseResult(result.Message.Type, result.Message.Result)
				if err != nil {
					log.Error("Failed to parse result: %v", err)
					errCh <- err
					return
				}

				parsedResultCh <- parsed
				if parsed.Status != nil && parsed.Status.Status == "completed" {
					return
				}
			}
		}
	}()
	return parsedResultCh, nil
}

func (c *Client) SendMessage(ctx context.Context, agentID string, threadID string, message string) (string, error) {
	id := getID(RequestChat)
	req := Request{
		ID:     id,
		Method: "chat",
		Body:   ChatRequest{ThreadID: &threadID, Prompt: message},
	}
	err := c.Send(req)
	if err != nil {
		return "", err
	}
	return "", nil
}

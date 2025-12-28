package ipc

import (
	"context"
	"encoding/json"

	"github.com/shuttl-ai/cli/log"
)

// SingleToolInfo represents an individual tool from the IPC response
type SingleToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Args        map[string]any `json:"args,omitempty"`
	ToolkitName string         `json:"toolkitName"`
}

func (c *Client) GetTools(ctx context.Context) ([]SingleToolInfo, error) {
	id := getID("request_tools")
	req := Request{
		ID:     id,
		Method: "listTools",
		Body:   nil,
	}
	output, err := c.SendAndWaitForResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	if !output.Message.Success {
		return nil, output.Message.ErrorObj
	}
	var toolListResponse []SingleToolInfo
	if err := json.Unmarshal(output.Message.Result, &toolListResponse); err != nil {
		return nil, err
	}
	log.Info("Received %d tools: %v", len(toolListResponse), toolListResponse)
	return toolListResponse, nil
}


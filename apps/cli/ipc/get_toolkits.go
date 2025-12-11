package ipc

import (
	"context"
	"encoding/json"

	"github.com/shuttl-ai/cli/log"
)

type ToolkitInfo struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Tools       []ToolInfo `json:"tools"`
}

type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Args        map[string]any `json:"args"`
}

func (c *Client) GetToolkits(ctx context.Context) ([]ToolkitInfo, error) {
	id := getID("request_toolkits")
	req := Request{
		ID:     id,
		Method: "listToolkits",
		Body:   nil,
	}
	err := c.Send(req)
	if err != nil {
		return nil, err
	}
	log.Debug("Waiting for toolkits response...")
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			response, err := c.RecieveWithID(id, ctx)

			if err != nil {
				return nil, err
			}
			if !response.Message.Success {
				return nil, response.Message.ErrorObj
			}
			var toolkitListResponse []ToolkitInfo
			if err := json.Unmarshal(response.Message.Result, &toolkitListResponse); err != nil {
				return nil, err
			}
			log.Debug("Received %d toolkits: %v", len(toolkitListResponse), toolkitListResponse)
			return toolkitListResponse, nil
		}
	}
}

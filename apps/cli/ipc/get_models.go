package ipc

import (
	"context"
	"encoding/json"

	"github.com/shuttl-ai/cli/log"
)

// ModelInfo represents model information from the IPC response
type ModelInfo struct {
	Identifier string `json:"identifier"`
	Key        Secret `json:"key"`
}

func (c *Client) GetModels(ctx context.Context) ([]ModelInfo, error) {
	id := getID("request_models")
	req := Request{
		ID:     id,
		Method: "listModels",
		Body:   nil,
	}
	output, err := c.SendAndWaitForResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	if !output.Message.Success {
		return nil, output.Message.ErrorObj
	}
	var modelListResponse []ModelInfo
	if err := json.Unmarshal(output.Message.Result, &modelListResponse); err != nil {
		return nil, err
	}
	log.Info("Received %d models: %v", len(modelListResponse), modelListResponse)
	return modelListResponse, nil
}


package ipc

import (
	"context"
	"encoding/json"

	"github.com/shuttl-ai/cli/log"
)

// PromptInfo represents prompt information from the IPC response
type PromptInfo struct {
	AgentName    string `json:"agentName"`
	SystemPrompt string `json:"systemPrompt"`
}

func (c *Client) GetPrompts(ctx context.Context) ([]PromptInfo, error) {
	c.wg.Add(1)
	defer c.wg.Done()
	id := getID("request_prompts")
	req := Request{
		ID:     id,
		Method: "listPrompts",
		Body:   nil,
	}
	output, err := c.SendAndWaitForResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	if !output.Message.Success {
		return nil, output.Message.ErrorObj
	}
	var promptListResponse []PromptInfo
	if err := json.Unmarshal(output.Message.Result, &promptListResponse); err != nil {
		return nil, err
	}
	log.Info("Received %d prompts: %v", len(promptListResponse), promptListResponse)
	return promptListResponse, nil
}

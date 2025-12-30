package ipc

import (
	"context"
	"encoding/json"

	"github.com/shuttl-ai/cli/log"
)

// TriggerInfo represents trigger information from the IPC response
type TriggerInfo struct {
	Name        string         `json:"name"`
	TriggerType string         `json:"triggerType"`
	Description string         `json:"description"`
	Args        map[string]any `json:"args,omitempty"`
	AgentName   string         `json:"agentName"`
}

func (c *Client) GetTriggers(ctx context.Context) ([]TriggerInfo, error) {
	c.wg.Add(1)
	defer c.wg.Done()
	id := getID("request_triggers")
	req := Request{
		ID:     id,
		Method: "listTriggers",
		Body:   nil,
	}
	output, err := c.SendAndWaitForResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	if !output.Message.Success {
		return nil, output.Message.ErrorObj
	}
	var triggerListResponse []TriggerInfo
	if err := json.Unmarshal(output.Message.Result, &triggerListResponse); err != nil {
		return nil, err
	}
	log.Info("Received %d triggers: %v", len(triggerListResponse), triggerListResponse)
	return triggerListResponse, nil
}

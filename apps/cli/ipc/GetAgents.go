package ipc

import (
	"context"
	"encoding/json"

	"github.com/shuttl-ai/cli/log"
)

/*
{
"id":"list_agents","
"success":true,"
"result":[

	{
		"name":"TestAgent",
		"systemPrompt":"You are a helpful test agent for integration testing.",
		"model":{"identifier":"test-model","key":"test-key-12345"},
		"toolkits":["UtilityToolkit","AsyncToolkit"]
	}}]}
*/
type AgentInfo struct {
	Name         string   `json:"name"`
	SystemPrompt string   `json:"systemPrompt"`
	Model        Model    `json:"model"`
	Toolkits     []string `json:"toolkits"`
}

type Model struct {
	Identifier string `json:"identifier"`
	Key        string `json:"key"`
}

func (c *Client) GetAgents(ctx context.Context) ([]AgentInfo, error) {
	id := getID(RequestListAgents)
	req := Request{
		ID:     id,
		Method: "listAgents",
		Body:   nil,
	}
	err := c.Send(req)
	if err != nil {
		return nil, err
	}
	log.Debug("Waiting for agents response...")
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

			var agentListResponse []AgentInfo
			if err := json.Unmarshal(response.Message.Result, &agentListResponse); err != nil {
				return nil, err
			}
			log.Debug("Received %d agents: %v", len(agentListResponse), agentListResponse)
			return agentListResponse, nil
		}
	}

}

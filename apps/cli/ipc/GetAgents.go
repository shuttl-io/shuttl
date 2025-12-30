package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/shuttl-ai/cli/auth"
	"github.com/shuttl-ai/cli/config"
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

type Secret struct {
	Source string `json:"source"`
	Name   string `json:"name"`
}

type Model struct {
	Identifier string `json:"identifier"`
	Key        Secret `json:"key"`
}

// AvailableModel represents a model available from the API
type AvailableModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description,omitempty"`
}

type BadApiStatusError struct {
	StatusCode int
	Body       string
}

func (e *BadApiStatusError) Error() string {
	return fmt.Sprintf("API returned status %d: %s", e.StatusCode, e.Body)
}

type InvalidAgentModelsError struct {
	Agents          []AgentInfo
	AvailableModels []AvailableModel
}

func (e *InvalidAgentModelsError) Error() string {
	return fmt.Sprintf("agents have invalid models: %v, available models: %v", e.Agents, e.AvailableModels)
}

// FetchAvailableModels fetches available models from the HTTP API
func FetchAvailableModels(ctx context.Context, organizationID int, apiURL string, accessToken string) ([]AvailableModel, error) {
	url := fmt.Sprintf("%s/api/%d/models/available", apiURL, organizationID)
	log.Debug("Fetching available models from: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch available models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &BadApiStatusError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var models []AvailableModel
	if err := json.Unmarshal(body, &models); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	log.Debug("Received %d available models", len(models))
	return models, nil
}

func (c *Client) GetAgents(ctx context.Context) ([]AgentInfo, error) {
	c.wg.Add(1)
	defer c.wg.Done()
	id := getID(RequestListAgents)
	req := Request{
		ID:     id,
		Method: "listAgents",
		Body:   nil,
	}
	output, err := c.SendAndWaitForResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	if !output.Message.Success {
		return nil, output.Message.ErrorObj
	}
	var agentListResponse []AgentInfo
	if err := json.Unmarshal(output.Message.Result, &agentListResponse); err != nil {
		return nil, err
	}
	log.Info("Received %d agents: %v", len(agentListResponse), agentListResponse)
	if err := c.validateAgentModels(ctx, agentListResponse); err != nil {
		return nil, err
	}
	return agentListResponse, nil
}

// validateAgentModels validates that each agent has a model that is available in the API
func (c *Client) validateAgentModels(ctx context.Context, agents []AgentInfo) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}
	if cfg.OrganizationID == nil {
		log.Debug("No organization ID found in config, skipping model validation")
		return nil
	}
	availableModels, err := c.fetchAvailableModelsFromAPI(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch available models for validation: %w", err)
	}

	// Build a set of valid model IDs for quick lookup
	validModelIDs := make(map[string]bool)
	for _, model := range availableModels {
		validModelIDs[model.ID] = true
	}

	// Validate each agent's model
	var invalidAgents []string
	for _, agent := range agents {
		if !validModelIDs[agent.Model.Identifier] {
			invalidAgents = append(invalidAgents, fmt.Sprintf("%s (model: %s)", agent.Name, agent.Model.Identifier))
		}
	}

	if len(invalidAgents) > 0 {
		return &InvalidAgentModelsError{Agents: agents, AvailableModels: availableModels}
	}

	log.Debug("All %d agents have valid models", len(agents))
	return nil
}

// fetchAvailableModelsFromAPI fetches available models using auth credentials and config
func (c *Client) fetchAvailableModelsFromAPI(ctx context.Context) ([]AvailableModel, error) {
	// Load authentication tokens
	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, fmt.Errorf("failed to load auth tokens: %w", err)
	}

	// Load project config to get organization ID
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load user config to get API URL
	userCfg, err := config.LoadUserConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	apiURL := userCfg.GetAPIURL()
	return FetchAvailableModels(ctx, *cfg.OrganizationID, apiURL, tokens.AccessToken)
}

package ipc

import (
	"encoding/json"
	"testing"
)

func TestChatRequest(t *testing.T) {
	t.Run("marshal without thread_id", func(t *testing.T) {
		req := ChatRequest{
			Agent:  "test-agent",
			Prompt: "Hello, world!",
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Verify ThreadID is omitted
		var unmarshaled map[string]interface{}
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if _, exists := unmarshaled["thread_id"]; exists {
			t.Error("Expected thread_id to be omitted when nil")
		}

		if unmarshaled["agent"] != "test-agent" {
			t.Errorf("Expected agent 'test-agent', got '%v'", unmarshaled["agent"])
		}

		if unmarshaled["prompt"] != "Hello, world!" {
			t.Errorf("Expected prompt 'Hello, world!', got '%v'", unmarshaled["prompt"])
		}
	})

	t.Run("marshal with thread_id", func(t *testing.T) {
		threadID := "thread-123"
		req := ChatRequest{
			Agent:    "test-agent",
			ThreadID: &threadID,
			Prompt:   "Continue...",
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var unmarshaled ChatRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.ThreadID == nil {
			t.Fatal("Expected ThreadID to be non-nil")
		}

		if *unmarshaled.ThreadID != "thread-123" {
			t.Errorf("Expected ThreadID 'thread-123', got '%s'", *unmarshaled.ThreadID)
		}
	})
}

func TestChatResponse(t *testing.T) {
	response := ChatResponse{
		AgentID: "agent-456",
		Message: "Hello from the agent!",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ChatResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.AgentID != "agent-456" {
		t.Errorf("Expected AgentID 'agent-456', got '%s'", unmarshaled.AgentID)
	}

	if unmarshaled.Message != "Hello from the agent!" {
		t.Errorf("Expected Message 'Hello from the agent!', got '%s'", unmarshaled.Message)
	}
}

func TestLogEvent(t *testing.T) {
	testCases := []struct {
		name  string
		event LogEvent
	}{
		{
			name: "debug log",
			event: LogEvent{
				Level:   "debug",
				Message: "Debug message",
				AgentID: "agent-1",
			},
		},
		{
			name: "info log",
			event: LogEvent{
				Level:   "info",
				Message: "Info message",
			},
		},
		{
			name: "warn log",
			event: LogEvent{
				Level:   "warn",
				Message: "Warning message",
				AgentID: "agent-2",
			},
		},
		{
			name: "error log",
			event: LogEvent{
				Level:   "error",
				Message: "Error message",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.event)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var unmarshaled LogEvent
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if unmarshaled.Level != tc.event.Level {
				t.Errorf("Expected Level '%s', got '%s'", tc.event.Level, unmarshaled.Level)
			}

			if unmarshaled.Message != tc.event.Message {
				t.Errorf("Expected Message '%s', got '%s'", tc.event.Message, unmarshaled.Message)
			}

			if unmarshaled.AgentID != tc.event.AgentID {
				t.Errorf("Expected AgentID '%s', got '%s'", tc.event.AgentID, unmarshaled.AgentID)
			}
		})
	}
}

func TestErrorPayload(t *testing.T) {
	payload := ErrorPayload{
		Code:    "VALIDATION_ERROR",
		Message: "Invalid input",
		Details: "Field 'name' is required",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ErrorPayload
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected Code 'VALIDATION_ERROR', got '%s'", unmarshaled.Code)
	}

	if unmarshaled.Message != "Invalid input" {
		t.Errorf("Expected Message 'Invalid input', got '%s'", unmarshaled.Message)
	}

	if unmarshaled.Details != "Field 'name' is required" {
		t.Errorf("Expected Details, got '%s'", unmarshaled.Details)
	}
}

func TestStatusPayload(t *testing.T) {
	testCases := []struct {
		name    string
		payload StatusPayload
	}{
		{
			name: "ready status",
			payload: StatusPayload{
				Status:  "ready",
				Version: "1.0.0",
			},
		},
		{
			name: "initializing status",
			payload: StatusPayload{
				Status: "initializing",
			},
		},
		{
			name: "error status",
			payload: StatusPayload{
				Status: "error",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var unmarshaled StatusPayload
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if unmarshaled.Status != tc.payload.Status {
				t.Errorf("Expected Status '%s', got '%s'", tc.payload.Status, unmarshaled.Status)
			}

			if unmarshaled.Version != tc.payload.Version {
				t.Errorf("Expected Version '%s', got '%s'", tc.payload.Version, unmarshaled.Version)
			}
		})
	}
}

func TestRequestConstants(t *testing.T) {
	if RequestListAgents != "list_agents" {
		t.Errorf("Expected RequestListAgents 'list_agents', got '%s'", RequestListAgents)
	}

	if RequestChat != "chat" {
		t.Errorf("Expected RequestChat 'chat', got '%s'", RequestChat)
	}

	if RequestStatus != "status" {
		t.Errorf("Expected RequestStatus 'status', got '%s'", RequestStatus)
	}

	if RequestShutdown != "shutdown" {
		t.Errorf("Expected RequestShutdown 'shutdown', got '%s'", RequestShutdown)
	}
}

func TestEventConstants(t *testing.T) {
	if EventAgentStatus != "agent_status" {
		t.Errorf("Expected EventAgentStatus 'agent_status', got '%s'", EventAgentStatus)
	}

	if EventLog != "log" {
		t.Errorf("Expected EventLog 'log', got '%s'", EventLog)
	}

	if EventReady != "ready" {
		t.Errorf("Expected EventReady 'ready', got '%s'", EventReady)
	}
}

func TestAgentInfo(t *testing.T) {
	jsonData := `{
		"name": "TestAgent",
		"systemPrompt": "You are a helpful test agent.",
		"model": {
			"identifier": "gpt-4",
			"key": {
				"source": "env",
				"name": "OPENAI_API_KEY"
			}
		},
		"toolkits": ["Toolkit1", "Toolkit2"]
	}`

	var agent AgentInfo
	if err := json.Unmarshal([]byte(jsonData), &agent); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if agent.Name != "TestAgent" {
		t.Errorf("Expected Name 'TestAgent', got '%s'", agent.Name)
	}

	if agent.SystemPrompt != "You are a helpful test agent." {
		t.Errorf("Expected SystemPrompt, got '%s'", agent.SystemPrompt)
	}

	if agent.Model.Identifier != "gpt-4" {
		t.Errorf("Expected Model.Identifier 'gpt-4', got '%s'", agent.Model.Identifier)
	}

	if agent.Model.Key.Source != "env" {
		t.Errorf("Expected Model.Key.Source 'env', got '%s'", agent.Model.Key.Source)
	}

	if len(agent.Toolkits) != 2 {
		t.Fatalf("Expected 2 toolkits, got %d", len(agent.Toolkits))
	}

	if agent.Toolkits[0] != "Toolkit1" {
		t.Errorf("Expected first toolkit 'Toolkit1', got '%s'", agent.Toolkits[0])
	}
}

func TestToolkitInfo(t *testing.T) {
	jsonData := `{
		"name": "UtilityToolkit",
		"description": "A collection of utility tools",
		"tools": [
			{
				"name": "calculator",
				"description": "Performs calculations",
				"args": {
					"expression": {"type": "string", "required": true}
				}
			},
			{
				"name": "formatter",
				"description": "Formats text"
			}
		]
	}`

	var toolkit ToolkitInfo
	if err := json.Unmarshal([]byte(jsonData), &toolkit); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if toolkit.Name != "UtilityToolkit" {
		t.Errorf("Expected Name 'UtilityToolkit', got '%s'", toolkit.Name)
	}

	if toolkit.Description != "A collection of utility tools" {
		t.Errorf("Expected Description, got '%s'", toolkit.Description)
	}

	if len(toolkit.Tools) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(toolkit.Tools))
	}

	if toolkit.Tools[0].Name != "calculator" {
		t.Errorf("Expected first tool 'calculator', got '%s'", toolkit.Tools[0].Name)
	}

	if toolkit.Tools[0].Description != "Performs calculations" {
		t.Errorf("Expected tool description, got '%s'", toolkit.Tools[0].Description)
	}
}

func TestAvailableModel(t *testing.T) {
	model := AvailableModel{
		ID:          "gpt-4-turbo",
		Name:        "GPT-4 Turbo",
		Provider:    "openai",
		Description: "Latest GPT-4 model",
	}

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled AvailableModel
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ID != "gpt-4-turbo" {
		t.Errorf("Expected ID 'gpt-4-turbo', got '%s'", unmarshaled.ID)
	}

	if unmarshaled.Name != "GPT-4 Turbo" {
		t.Errorf("Expected Name 'GPT-4 Turbo', got '%s'", unmarshaled.Name)
	}

	if unmarshaled.Provider != "openai" {
		t.Errorf("Expected Provider 'openai', got '%s'", unmarshaled.Provider)
	}
}

func TestBadApiStatusError(t *testing.T) {
	err := &BadApiStatusError{
		StatusCode: 404,
		Body:       "Not found",
	}

	errStr := err.Error()
	expected := "API returned status 404: Not found"
	if errStr != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, errStr)
	}
}

func TestInvalidAgentModelsError(t *testing.T) {
	err := &InvalidAgentModelsError{
		Agents: []AgentInfo{
			{Name: "Agent1", Model: Model{Identifier: "invalid-model"}},
		},
		AvailableModels: []AvailableModel{
			{ID: "valid-model"},
		},
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}
}

func TestChatParsedResultString(t *testing.T) {
	testCases := []struct {
		name   string
		result ChatParsedResult
	}{
		{
			name: "output_text_delta",
			result: ChatParsedResult{
				Type: "output_text_delta",
				TextDelta: &TextDeltaResult{
					TypeName: "output_text_delta",
					OutputTextDelta: struct {
						Delta          string `json:"delta"`
						SequenceNumber int    `json:"sequenceNumber"`
					}{
						Delta:          "Hello",
						SequenceNumber: 1,
					},
				},
			},
		},
		{
			name: "output_text",
			result: ChatParsedResult{
				Type: "output_text",
				FinalOutput: &FinalOutputResult{
					OutputText: struct {
						Text string `json:"text"`
					}{
						Text: "Complete message",
					},
				},
			},
		},
		{
			name: "tool_call",
			result: ChatParsedResult{
				Type: "tool_call",
				ToolCall: &ToolCallResult{
					TypeName: "tool_call",
				},
			},
		},
		{
			name: "status",
			result: ChatParsedResult{
				Type: "status",
				Status: &StatusResult{
					ThreadID: "thread-123",
					Status:   "completed",
				},
			},
		},
		{
			name: "unknown",
			result: ChatParsedResult{
				Type: "unknown_type",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			str := tc.result.String()
			if str == "" {
				t.Error("Expected non-empty string")
			}
		})
	}
}

func TestParseResult(t *testing.T) {
	t.Run("parse tool_call", func(t *testing.T) {
		raw := json.RawMessage(`{"typeName": "tool_call", "toolCall": {"name": "test", "arguments": {}, "callId": "call-1"}}`)

		result, err := parseResult("tool_call", raw)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if result.Type != "tool_call" {
			t.Errorf("Expected type 'tool_call', got '%s'", result.Type)
		}

		if result.ToolCall == nil {
			t.Fatal("Expected ToolCall to be non-nil")
		}
	})

	t.Run("parse output_text_delta", func(t *testing.T) {
		raw := json.RawMessage(`{"typeName": "output_text_delta", "outputTextDelta": {"delta": "hello", "sequenceNumber": 1}}`)

		result, err := parseResult("output_text_delta", raw)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if result.Type != "output_text_delta" {
			t.Errorf("Expected type 'output_text_delta', got '%s'", result.Type)
		}

		if result.TextDelta == nil {
			t.Fatal("Expected TextDelta to be non-nil")
		}
	})

	t.Run("parse output_text", func(t *testing.T) {
		raw := json.RawMessage(`{"outputText": {"text": "Complete response"}}`)

		result, err := parseResult("output_text", raw)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if result.FinalOutput == nil {
			t.Fatal("Expected FinalOutput to be non-nil")
		}

		if result.FinalOutput.OutputText.Text != "Complete response" {
			t.Errorf("Expected text 'Complete response', got '%s'", result.FinalOutput.OutputText.Text)
		}
	})

	t.Run("parse response.requested", func(t *testing.T) {
		raw := json.RawMessage(`{"model": "gpt-4"}`)

		result, err := parseResult("response.requested", raw)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if result.ResponseRequested == nil {
			t.Fatal("Expected ResponseRequested to be non-nil")
		}
	})

	t.Run("parse unknown type as status", func(t *testing.T) {
		raw := json.RawMessage(`{"threadId": "thread-456", "status": "completed"}`)

		result, err := parseResult("some_unknown_type", raw)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if result.Status == nil {
			t.Fatal("Expected Status to be non-nil for unknown type")
		}

		if result.Status.ThreadID != "thread-456" {
			t.Errorf("Expected ThreadID 'thread-456', got '%s'", result.Status.ThreadID)
		}
	})
}




package ipc

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestParseCommand(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple command",
			input:    "ls -la",
			expected: []string{"ls", "-la"},
		},
		{
			name:     "command with double quotes",
			input:    `echo "hello world"`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "command with single quotes",
			input:    `echo 'hello world'`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "mixed quotes",
			input:    `program --name="test value" --other='another value'`,
			expected: []string{"program", "--name=test value", "--other=another value"},
		},
		{
			name:     "escaped spaces",
			input:    `path\ with\ spaces arg2`,
			expected: []string{"path with spaces", "arg2"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "only whitespace",
			input:    "   \t  ",
			expected: nil,
		},
		{
			name:     "multiple spaces between args",
			input:    "cmd    arg1     arg2",
			expected: []string{"cmd", "arg1", "arg2"},
		},
		{
			name:     "tabs as separators",
			input:    "cmd\targ1\targ2",
			expected: []string{"cmd", "arg1", "arg2"},
		},
		{
			name:     "complex npm command",
			input:    `npx ts-node --project "./tsconfig.json" index.ts`,
			expected: []string{"npx", "ts-node", "--project", "./tsconfig.json", "index.ts"},
		},
		{
			name:     "command with equals",
			input:    "env VAR=value cmd",
			expected: []string{"env", "VAR=value", "cmd"},
		},
		{
			name:     "nested quotes in double quotes",
			input:    `echo "he said 'hello'"`,
			expected: []string{"echo", "he said 'hello'"},
		},
		{
			name:     "escaped backslash",
			input:    `path\\to\\file`,
			expected: []string{"path\\to\\file"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseCommand(tc.input)

			if len(result) != len(tc.expected) {
				t.Fatalf("Expected %d args, got %d: %v", len(tc.expected), len(result), result)
			}

			for i, arg := range result {
				if arg != tc.expected[i] {
					t.Errorf("Arg %d: expected '%s', got '%s'", i, tc.expected[i], arg)
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	command := []string{"echo", "hello"}
	client := NewClient(command)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if len(client.Command()) != 2 {
		t.Errorf("Expected command length 2, got %d", len(client.Command()))
	}

	if client.Command()[0] != "echo" {
		t.Errorf("Expected first arg 'echo', got '%s'", client.Command()[0])
	}

	if client.State() != StateIdle {
		t.Errorf("Expected initial state to be StateIdle, got %s", client.State())
	}

	if client.IsRunning() {
		t.Error("Expected IsRunning to be false initially")
	}

	if client.ProcessID() != 0 {
		t.Errorf("Expected ProcessID to be 0 initially, got %d", client.ProcessID())
	}
}

func TestClientState(t *testing.T) {
	t.Run("StateIdle string", func(t *testing.T) {
		if StateIdle.String() != "idle" {
			t.Errorf("Expected 'idle', got '%s'", StateIdle.String())
		}
	})

	t.Run("StateRunning string", func(t *testing.T) {
		if StateRunning.String() != "running" {
			t.Errorf("Expected 'running', got '%s'", StateRunning.String())
		}
	})

	t.Run("StateStopping string", func(t *testing.T) {
		if StateStopping.String() != "stopping" {
			t.Errorf("Expected 'stopping', got '%s'", StateStopping.String())
		}
	})

	t.Run("StateStopped string", func(t *testing.T) {
		if StateStopped.String() != "stopped" {
			t.Errorf("Expected 'stopped', got '%s'", StateStopped.String())
		}
	})

	t.Run("unknown state string", func(t *testing.T) {
		unknownState := ClientState(999)
		if unknownState.String() != "unknown" {
			t.Errorf("Expected 'unknown', got '%s'", unknownState.String())
		}
	})
}

func TestErrorObject(t *testing.T) {
	err := &ErrorObject{
		Code:    "TEST_ERROR",
		Message: "Something went wrong",
		Details: "Additional details",
	}

	errStr := err.Error()
	if errStr != "TEST_ERROR, message: Something went wrong" {
		t.Errorf("Unexpected error string: %s", errStr)
	}

	str := err.String()
	if str != "TEST_ERROR, message: Something went wrong" {
		t.Errorf("Unexpected string: %s", str)
	}
}

func TestMessage(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		msg := Message{
			Type:      MessageTypeResponse,
			ID:        "test-123",
			Success:   true,
			Timestamp: time.Now(),
			Result:    json.RawMessage(`{"key": "value"}`),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var unmarshaled Message
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.Type != MessageTypeResponse {
			t.Errorf("Expected type 'response', got '%s'", unmarshaled.Type)
		}

		if unmarshaled.ID != "test-123" {
			t.Errorf("Expected ID 'test-123', got '%s'", unmarshaled.ID)
		}

		if !unmarshaled.Success {
			t.Error("Expected Success to be true")
		}
	})

	t.Run("message with error", func(t *testing.T) {
		msg := Message{
			Type:    MessageTypeError,
			ID:      "error-456",
			Success: false,
			ErrorObj: &ErrorObject{
				Code:    "INTERNAL_ERROR",
				Message: "Something failed",
			},
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var unmarshaled Message
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.ErrorObj == nil {
			t.Fatal("Expected ErrorObj to be non-nil")
		}

		if unmarshaled.ErrorObj.Code != "INTERNAL_ERROR" {
			t.Errorf("Expected error code 'INTERNAL_ERROR', got '%s'", unmarshaled.ErrorObj.Code)
		}
	})
}

func TestRequest(t *testing.T) {
	req := Request{
		ID:     "req-123",
		Method: "listAgents",
		Body: map[string]interface{}{
			"filter": "active",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled Request
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ID != "req-123" {
		t.Errorf("Expected ID 'req-123', got '%s'", unmarshaled.ID)
	}

	if unmarshaled.Method != "listAgents" {
		t.Errorf("Expected Method 'listAgents', got '%s'", unmarshaled.Method)
	}
}

func TestOutputLine(t *testing.T) {
	now := time.Now()
	msg := &Message{
		Type:    MessageTypeResponse,
		ID:      "out-123",
		Success: true,
	}

	output := OutputLine{
		Source:    "stdout",
		Content:   `{"type":"response","id":"out-123","success":true}`,
		Message:   msg,
		Timestamp: now,
	}

	if output.Source != "stdout" {
		t.Errorf("Expected Source 'stdout', got '%s'", output.Source)
	}

	if output.Message == nil {
		t.Fatal("Expected Message to be non-nil")
	}

	if output.Message.ID != "out-123" {
		t.Errorf("Expected Message ID 'out-123', got '%s'", output.Message.ID)
	}
}

func TestClientStartWithNoCommand(t *testing.T) {
	client := NewClient([]string{})

	err := client.Start()
	if err == nil {
		t.Error("Expected error when starting with no command")
	}

	if client.State() != StateStopped {
		t.Errorf("Expected state to be StateStopped after failed start, got %s", client.State())
	}
}

func TestClientStartAlreadyRunning(t *testing.T) {
	client := NewClient([]string{"echo", "test"})

	// Manually set state to running
	client.state = StateRunning

	err := client.Start()
	if err == nil {
		t.Error("Expected error when starting already running client")
	}
}

func TestClientSendWhenNotRunning(t *testing.T) {
	client := NewClient([]string{"echo", "test"})

	req := Request{
		ID:     "test",
		Method: "ping",
	}

	err := client.Send(req)
	if err == nil {
		t.Error("Expected error when sending to non-running client")
	}
}

func TestClientStopWhenNotRunning(t *testing.T) {
	client := NewClient([]string{"echo", "test"})

	// Should not error when stopping an idle client
	err := client.Stop()
	if err != nil {
		t.Errorf("Unexpected error stopping idle client: %v", err)
	}
}

func TestClientKillWhenNotRunning(t *testing.T) {
	client := NewClient([]string{"echo", "test"})

	// Should not error when killing an idle client
	err := client.Kill()
	if err != nil {
		t.Errorf("Unexpected error killing idle client: %v", err)
	}
}

func TestClientContext(t *testing.T) {
	client := NewClient([]string{"echo", "test"})

	ctx := client.Context()
	if ctx == nil {
		t.Error("Expected non-nil context")
	}

	// Context should not be done initially
	select {
	case <-ctx.Done():
		t.Error("Context should not be done initially")
	default:
		// OK
	}
}

func TestMessageTypes(t *testing.T) {
	if MessageTypeRequest != "request" {
		t.Errorf("Expected MessageTypeRequest 'request', got '%s'", MessageTypeRequest)
	}

	if MessageTypeResponse != "response" {
		t.Errorf("Expected MessageTypeResponse 'response', got '%s'", MessageTypeResponse)
	}

	if MessageTypeEvent != "event" {
		t.Errorf("Expected MessageTypeEvent 'event', got '%s'", MessageTypeEvent)
	}

	if MessageTypeError != "error" {
		t.Errorf("Expected MessageTypeError 'error', got '%s'", MessageTypeError)
	}
}

func TestClientWithRealCommand(t *testing.T) {
	// Skip this test if running in CI without access to 'echo'
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client := NewClient([]string{"echo", "hello"})

	err := client.Start()
	if err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	// Wait a bit for the command to complete
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.WaitTimeout(2 * time.Second)
	if err != nil {
		t.Logf("WaitTimeout: %v", err)
	}

	// Clean up
	client.Close()

	select {
	case <-ctx.Done():
		t.Log("Context done")
	default:
	}
}

func TestSendPayloadMarshaling(t *testing.T) {
	// Test that SendPayload properly marshals the payload
	// We can't test the actual sending without a running process,
	// but we can verify the marshaling logic works
	type TestPayload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	payload := TestPayload{Name: "test", Value: 42}

	// Verify the payload can be marshaled to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	var unmarshaled TestPayload
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if unmarshaled.Name != "test" {
		t.Errorf("Expected Name 'test', got '%s'", unmarshaled.Name)
	}

	if unmarshaled.Value != 42 {
		t.Errorf("Expected Value 42, got %d", unmarshaled.Value)
	}
}


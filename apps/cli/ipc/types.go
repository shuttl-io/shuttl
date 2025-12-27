package ipc

// Common payload types for IPC communication with Shuttl applications

// FileAttachment represents a file attached to a chat message
type FileAttachment struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content,omitempty"`  // Base64 encoded content
	MimeType string `json:"mime_type,omitempty"`
}

// ChatRequest represents a chat message request payload
type ChatRequest struct {
	Agent       string           `json:"agent"`
	ThreadID    *string          `json:"thread_id,omitempty"`
	Prompt      string           `json:"prompt"`
	Attachments []FileAttachment `json:"attachments,omitempty"`
}

// ChatResponse represents a chat message response payload
type ChatResponse struct {
	AgentID string `json:"agent_id"`
	Message string `json:"message"`
}

// LogEvent represents a log event from the application
type LogEvent struct {
	Level   string `json:"level"` // "debug", "info", "warn", "error"
	Message string `json:"message"`
	AgentID string `json:"agent_id,omitempty"`
}

// ErrorPayload represents an error response payload
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// StatusPayload represents application status
type StatusPayload struct {
	Status  string `json:"status"` // "ready", "initializing", "error"
	Version string `json:"version,omitempty"`
}

// Common message IDs for requests
const (
	RequestListAgents = "list_agents"
	RequestChat       = "chat"
	RequestStatus     = "status"
	RequestShutdown   = "shutdown"
)

// Common event types
const (
	EventAgentStatus = "agent_status"
	EventLog         = "log"
	EventReady       = "ready"
)

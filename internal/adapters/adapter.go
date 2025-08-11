package adapters

import "net/http"

// UnifiedChatRequest is a provider-agnostic representation of a chat request.
// It's designed to be a superset of the common fields across different APIs.
type UnifiedChatRequest struct {
	Model       string
	Messages    []UnifiedMessage
	Stream      bool
	Tools       []UnifiedTool
	ToolChoice  interface{}
	// Parameters holds provider-specific parameters that don't have a common mapping.
	Parameters map[string]interface{}
}

// UnifiedMessage is a single message in a chat conversation.
type UnifiedMessage struct {
	Role         string
	Content      string
	ToolCalls    []UnifiedToolCall
	ToolCallID   string
	Name         string
}

// UnifiedToolCall represents a call to a tool function.
type UnifiedToolCall struct {
	ID       string
	Type     string
	Function UnifiedFunctionCall
}

// UnifiedFunctionCall represents a call to a function.
type UnifiedFunctionCall struct {
	Name      string
	Arguments string
}

// UnifiedTool represents a tool that the model can call.
type UnifiedTool struct {
	Type     string
	Function UnifiedFunction
}

// UnifiedFunction represents the definition of a function tool.
type UnifiedFunction struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// UnifiedChatResponse is a placeholder for the response. The actual implementation
// will involve streaming chunks of data.
// For now, we will focus on the request side.
type UnifiedChatResponse struct {
	ID         string
	Model      string
	Role       string
	Content    string
	ToolCalls  []UnifiedToolCall
	StopReason string
	Usage      UnifiedUsage
}

// UnifiedUsage represents token usage information.
type UnifiedUsage struct {
	InputTokens  int
	OutputTokens int
}

// UnifiedEmbeddingRequest is a provider-agnostic representation of an embedding request.
type UnifiedEmbeddingRequest struct {
	Input []string
	Model string
}

// UnifiedEmbeddingResponse is a provider-agnostic representation of an embedding response.
type UnifiedEmbeddingResponse struct {
	Embeddings [][]float32
	Model      string
}

// Adapter defines the full suite of translation capabilities.
// A provider's adapter only needs to implement methods for the operations it supports.
type Adapter interface {
	// --- Chat Completion Operations ---
	ClientChatToUnified(*http.Request) (*UnifiedChatRequest, error)
	UnifiedChatToBackend(*UnifiedChatRequest, string) (*http.Request, error)
	BackendChatToUnified(*http.Response) (*UnifiedChatResponse, error)
	UnifiedChatToClient(*UnifiedChatResponse, http.ResponseWriter) error

	// --- Embedding Operations ---
	ClientEmbeddingToUnified(*http.Request) (*UnifiedEmbeddingRequest, error)
	UnifiedEmbeddingToBackend(*UnifiedEmbeddingRequest, string) (*http.Request, error)
	BackendEmbeddingToUnified(*http.Response) (*UnifiedEmbeddingResponse, error)
	UnifiedEmbeddingToClient(*UnifiedEmbeddingResponse, http.ResponseWriter) error

	// --- Error Translation ---
	// Translates a backend HTTP response into a client-facing error body.
	TranslateError(backendResp *http.Response) []byte
}


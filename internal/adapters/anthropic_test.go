package adapters

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAnthropicAdapter_ClientChatToUnified(t *testing.T) {
	adapter := &AnthropicAdapter{}

	// Test basic Anthropic request
	reqBody := `{
		"model": "claude-3-haiku-20240307",
		"messages": [
			{"role": "user", "content": "Hello"}
		]
	}`

	req, err := http.NewRequest("POST", "/v1/messages", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	unified, err := adapter.ClientChatToUnified(req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if unified.Model != "claude-3-haiku-20240307" {
		t.Errorf("Expected model claude-3-haiku-20240307, got: %s", unified.Model)
	}

	if len(unified.Messages) != 1 {
		t.Errorf("Expected 1 message, got: %d", len(unified.Messages))
	}

	if unified.Messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got: %s", unified.Messages[0].Role)
	}

	if unified.Messages[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got: %s", unified.Messages[0].Content)
	}
}

func TestAnthropicAdapter_BackendChatToUnified(t *testing.T) {
	adapter := &AnthropicAdapter{}

	// Mock Anthropic response
	respBody := `{
		"id": "msg_01EhbVbqzGnzEFNbqFjpqVEs",
		"type": "message",
		"role": "assistant",
		"content": [
			{
				"type": "text",
				"text": "Hello! How can I help you today?"
			}
		],
		"model": "claude-3-haiku-20240307",
		"stop_reason": "end_turn",
		"usage": {
			"input_tokens": 10,
			"output_tokens": 15
		}
	}`

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(respBody)),
	}

	unified, err := adapter.BackendChatToUnified(resp)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if unified.ID != "msg_01EhbVbqzGnzEFNbqFjpqVEs" {
		t.Errorf("Expected correct ID, got: %s", unified.ID)
	}

	if unified.Model != "claude-3-haiku-20240307" {
		t.Errorf("Expected model claude-3-haiku-20240307, got: %s", unified.Model)
	}

	if unified.Role != "assistant" {
		t.Errorf("Expected role assistant, got: %s", unified.Role)
	}

	if unified.Content != "Hello! How can I help you today?" {
		t.Errorf("Expected correct content, got: %s", unified.Content)
	}

	if unified.Usage.InputTokens != 10 {
		t.Errorf("Expected 10 input tokens, got: %d", unified.Usage.InputTokens)
	}

	if unified.Usage.OutputTokens != 15 {
		t.Errorf("Expected 15 output tokens, got: %d", unified.Usage.OutputTokens)
	}
}

func TestAnthropicAdapter_EmbeddingMethodsReturnErrors(t *testing.T) {
	adapter := &AnthropicAdapter{}

	// Test that embedding methods return appropriate errors
	_, err := adapter.ClientEmbeddingToUnified(nil)
	if err == nil {
		t.Error("Expected error for embedding request, got nil")
	}

	_, err = adapter.UnifiedEmbeddingToBackend(nil, "")
	if err == nil {
		t.Error("Expected error for embedding backend conversion, got nil")
	}

	_, err = adapter.BackendEmbeddingToUnified(nil)
	if err == nil {
		t.Error("Expected error for embedding response conversion, got nil")
	}

	err = adapter.UnifiedEmbeddingToClient(nil, nil)
	if err == nil {
		t.Error("Expected error for embedding client response, got nil")
	}
}
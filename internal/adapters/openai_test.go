package adapters

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOpenAIAdapter_ClientChatToUnified(t *testing.T) {
	adapter := &OpenAIAdapter{}

	// Test basic chat request
	reqBody := `{
		"model": "gpt-4",
		"messages": [
			{"role": "user", "content": "Hello"}
		],
		"stream": false
	}`

	req, err := http.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	unified, err := adapter.ClientChatToUnified(req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if unified.Model != "gpt-4" {
		t.Errorf("Expected model gpt-4, got: %s", unified.Model)
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

func TestOpenAIAdapter_ClientEmbeddingToUnified(t *testing.T) {
	adapter := &OpenAIAdapter{}

	// Test basic embedding request
	reqBody := `{
		"input": ["Hello", "World"],
		"model": "text-embedding-ada-002"
	}`

	req, err := http.NewRequest("POST", "/v1/embeddings", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	unified, err := adapter.ClientEmbeddingToUnified(req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if unified.Model != "text-embedding-ada-002" {
		t.Errorf("Expected model text-embedding-ada-002, got: %s", unified.Model)
	}

	if len(unified.Input) != 2 {
		t.Errorf("Expected 2 inputs, got: %d", len(unified.Input))
	}

	if unified.Input[0] != "Hello" {
		t.Errorf("Expected first input 'Hello', got: %s", unified.Input[0])
	}

	if unified.Input[1] != "World" {
		t.Errorf("Expected second input 'World', got: %s", unified.Input[1])
	}
}

func TestOpenAIAdapter_UnifiedChatToBackend(t *testing.T) {
	adapter := &OpenAIAdapter{}

	unified := &UnifiedChatRequest{
		Model: "gpt-4",
		Messages: []UnifiedMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
		Stream: false,
	}

	req, err := adapter.UnifiedChatToBackend(unified, "https://api.openai.com/v1/chat/completions")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Expected POST method, got: %s", req.Method)
	}

	if req.URL.String() != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("Expected correct URL, got: %s", req.URL.String())
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got: %s", req.Header.Get("Content-Type"))
	}
}

func TestOpenAIAdapter_BackendChatToUnified(t *testing.T) {
	adapter := &OpenAIAdapter{}

	// Mock OpenAI response
	respBody := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1677652288,
		"model": "gpt-4",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Hello! How can I help you today?"
				},
				"finish_reason": "stop"
			}
		],
		"usage": {
			"prompt_tokens": 9,
			"completion_tokens": 12,
			"total_tokens": 21
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

	if unified.ID != "chatcmpl-123" {
		t.Errorf("Expected ID chatcmpl-123, got: %s", unified.ID)
	}

	if unified.Model != "gpt-4" {
		t.Errorf("Expected model gpt-4, got: %s", unified.Model)
	}

	if unified.Role != "assistant" {
		t.Errorf("Expected role assistant, got: %s", unified.Role)
	}

	if unified.Content != "Hello! How can I help you today?" {
		t.Errorf("Expected correct content, got: %s", unified.Content)
	}

	if unified.Usage.InputTokens != 9 {
		t.Errorf("Expected 9 input tokens, got: %d", unified.Usage.InputTokens)
	}

	if unified.Usage.OutputTokens != 12 {
		t.Errorf("Expected 12 output tokens, got: %d", unified.Usage.OutputTokens)
	}
}
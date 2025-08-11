package workflows

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lmbroker/internal/adapters"
	"lmbroker/internal/config"
)

func TestHandleTranslation(t *testing.T) {
	// Create a mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock OpenAI response
		response := `{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"model": "gpt-4",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello from mock backend!"
					},
					"finish_reason": "stop"
				}
			],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 5,
				"total_tokens": 15
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer backendServer.Close()

	// Create test request (OpenAI format)
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

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create adapters
	clientAdapter := &adapters.OpenAIAdapter{}
	backendAdapter := &adapters.OpenAIAdapter{}

	// Create mock model config
	mockModel := &config.Model{
		Alias: "gpt-4",
		Type:  "openai",
		Target: config.TargetConfig{
			URL:   backendServer.URL,
			Model: "gpt-4",
		},
	}

	// Call the translation handler
	HandleTranslation(rr, req, clientAdapter, backendAdapter, backendServer.URL+"/v1/chat/completions", mockModel)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", rr.Code)
	}

	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got: %s", rr.Header().Get("Content-Type"))
	}

	// Check if response body contains expected content
	body := rr.Body.String()
	if !strings.Contains(body, "Hello from mock backend!") {
		t.Errorf("Expected response to contain mock message, got: %s", body)
	}
}

func TestHandlePassthrough(t *testing.T) {
	// Create a mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back some info about the request
		response := `{"message": "passthrough test", "method": "` + r.Method + `"}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer backendServer.Close()

	// Create test request
	reqBody := `{"test": "data"}`
	req, err := http.NewRequest("POST", "/test", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create mock model config
	mockModel := &config.Model{
		Alias: "test-model",
		Type:  "openai",
		Target: config.TargetConfig{
			URL:   backendServer.URL,
			Model: "test-model",
		},
	}

	// Call the passthrough handler
	HandlePassthrough(rr, req, backendServer.URL+"/test", mockModel)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "passthrough test") {
		t.Errorf("Expected response to contain passthrough test, got: %s", body)
	}

	if !strings.Contains(body, "POST") {
		t.Errorf("Expected response to contain POST method, got: %s", body)
	}
}
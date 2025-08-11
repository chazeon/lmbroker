package adapters

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

// OpenAIAdapter implements the Adapter interface for the OpenAI API.
type OpenAIAdapter struct{}


// --- Chat Completion Operations ---

func (a *OpenAIAdapter) ClientChatToUnified(r *http.Request) (*UnifiedChatRequest, error) {
	var openaiReq struct {
		Model    string `json:"model"`
		Messages []struct {
			Role         string `json:"role"`
			Content      string `json:"content"`
			ToolCalls    []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
			ToolCallID   string `json:"tool_call_id"`
			Name         string `json:"name"`
		} `json:"messages"`
		Tools    []UnifiedTool `json:"tools"`
		ToolChoice interface{} `json:"tool_choice"`
		Stream   bool   `json:"stream"`
		// Add other OpenAI-specific fields here if needed
		// Parameters map[string]interface{} `json:"-"` // Handled separately
	}

	if err := json.NewDecoder(r.Body).Decode(&openaiReq); err != nil {
		return nil, err
	}

	unifiedMessages := make([]UnifiedMessage, len(openaiReq.Messages))
	for i, msg := range openaiReq.Messages {
		unifiedMessages[i].Role = msg.Role
		unifiedMessages[i].Content = msg.Content
		unifiedMessages[i].ToolCallID = msg.ToolCallID
		unifiedMessages[i].Name = msg.Name

		if len(msg.ToolCalls) > 0 {
			unifiedToolCalls := make([]UnifiedToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				unifiedToolCalls[j] = UnifiedToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: UnifiedFunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
			unifiedMessages[i].ToolCalls = unifiedToolCalls
		}
	}

	unifiedReq := &UnifiedChatRequest{
		Model:    openaiReq.Model,
		Messages: unifiedMessages,
		Stream:   openaiReq.Stream,
		Tools:    openaiReq.Tools,
		// ToolChoice: openaiReq.ToolChoice, // ToolChoice needs special handling
	}

	// Handle ToolChoice separately as it can be a string or an object
	if tcStr, ok := openaiReq.ToolChoice.(string); ok {
		unifiedReq.ToolChoice = tcStr
	} else if tcMap, ok := openaiReq.ToolChoice.(map[string]interface{}); ok {
		unifiedReq.ToolChoice = tcMap
	}

	return unifiedReq, nil
}

func (a *OpenAIAdapter) UnifiedChatToBackend(unifiedReq *UnifiedChatRequest, backendURL string) (*http.Request, error) {
	openaiMessages := make([]map[string]interface{}, len(unifiedReq.Messages))
	for i, msg := range unifiedReq.Messages {
		openaiMsg := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.ToolCallID != "" {
			openaiMsg["tool_call_id"] = msg.ToolCallID
		}
		if msg.Name != "" {
			openaiMsg["name"] = msg.Name
		}

		if len(msg.ToolCalls) > 0 {
			openaiToolCalls := make([]map[string]interface{}, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				// Validate that arguments is valid JSON
				args := tc.Function.Arguments
				if args == "" {
					args = "{}" // Default to empty object if no arguments
				} else {
					// Test if it's valid JSON
					var testJSON interface{}
					if err := json.Unmarshal([]byte(args), &testJSON); err != nil {
						// If not valid JSON, wrap it as a string value
						argBytes, _ := json.Marshal(args)
						args = string(argBytes)
					}
				}
				
				openaiToolCalls[j] = map[string]interface{}{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": args,
					},
				}
			}
			openaiMsg["tool_calls"] = openaiToolCalls
		}
		openaiMessages[i] = openaiMsg
	}

	openaiReq := map[string]interface{}{
		"model":    unifiedReq.Model,
		"messages": openaiMessages,
		"stream":   unifiedReq.Stream,
	}

	if len(unifiedReq.Tools) > 0 {
		openaiReq["tools"] = unifiedReq.Tools
	}

	if unifiedReq.ToolChoice != nil {
		openaiReq["tool_choice"] = unifiedReq.ToolChoice
	}

	// Add any extra parameters
	for k, v := range unifiedReq.Parameters {
		openaiReq[k] = v
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", backendURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *OpenAIAdapter) BackendChatToUnified(backendResp *http.Response) (*UnifiedChatResponse, error) {
	// Read the response body for debugging
	bodyBytes, err := io.ReadAll(backendResp.Body)
	if err != nil {
		return nil, err
	}
	backendResp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	
	// Log backend response at debug level for troubleshooting
	slog.Debug("received backend response", "response", string(bodyBytes))
	
	var openaiResp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role         string `json:"role"`
				Content      string `json:"content"`
				ToolCalls    []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(backendResp.Body).Decode(&openaiResp); err != nil {
		return nil, err
	}

	unifiedResp := &UnifiedChatResponse{
		ID:    openaiResp.ID,
		Model: openaiResp.Model,
		Usage: UnifiedUsage{
			InputTokens:  openaiResp.Usage.PromptTokens,
			OutputTokens: openaiResp.Usage.CompletionTokens,
		},
	}

	if len(openaiResp.Choices) > 0 {
		choice := openaiResp.Choices[0]
		unifiedResp.Role = choice.Message.Role
		unifiedResp.Content = choice.Message.Content
		unifiedResp.StopReason = choice.FinishReason
		
		// Handle tool calls from OpenAI response
		if len(choice.Message.ToolCalls) > 0 {
			unifiedResp.ToolCalls = make([]UnifiedToolCall, len(choice.Message.ToolCalls))
			for i, toolCall := range choice.Message.ToolCalls {
				unifiedResp.ToolCalls[i] = UnifiedToolCall{
					ID:   toolCall.ID,
					Type: toolCall.Type,
					Function: UnifiedFunctionCall{
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
					},
				}
			}
		}
	}

	return unifiedResp, nil
}

func (a *OpenAIAdapter) UnifiedChatToClient(unifiedResp *UnifiedChatResponse, w http.ResponseWriter) error {
	openaiResp := map[string]interface{}{
		"id":      unifiedResp.ID,
		"object":  "chat.completion",
		"created": 0, // Current timestamp could be added here
		"model":   unifiedResp.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": func() map[string]interface{} {
					msg := map[string]interface{}{
						"role":    unifiedResp.Role,
						"content": unifiedResp.Content,
					}
					
					// Add tool calls if present
					if len(unifiedResp.ToolCalls) > 0 {
						toolCalls := make([]map[string]interface{}, len(unifiedResp.ToolCalls))
						for i, tc := range unifiedResp.ToolCalls {
							toolCalls[i] = map[string]interface{}{
								"id":   tc.ID,
								"type": tc.Type,
								"function": map[string]interface{}{
									"name":      tc.Function.Name,
									"arguments": tc.Function.Arguments,
								},
							}
						}
						msg["tool_calls"] = toolCalls
					}
					
					return msg
				}(),
				"finish_reason": unifiedResp.StopReason,
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     unifiedResp.Usage.InputTokens,
			"completion_tokens": unifiedResp.Usage.OutputTokens,
			"total_tokens":      unifiedResp.Usage.InputTokens + unifiedResp.Usage.OutputTokens,
		},
	}

	respBody, err := json.Marshal(openaiResp)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBody)
	return nil
}

// --- Error Translation ---

func (a *OpenAIAdapter) TranslateError(backendResp *http.Response) []byte {
	var openaiError struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	// Try to decode the backend error
	if err := json.NewDecoder(backendResp.Body).Decode(&openaiError); err != nil {
		// If we can't decode, return a generic error
		return []byte(`{"error": {"message": "An error occurred at the backend.", "type": "broker_error"}}`)
	}

	// Return the error in OpenAI format (passthrough since it's already OpenAI)
	errorResp := map[string]interface{}{
		"error": map[string]string{
			"message": openaiError.Error.Message,
			"type":    openaiError.Error.Type,
			"code":    openaiError.Error.Code,
		},
	}

	errorBody, _ := json.Marshal(errorResp)
	return errorBody
}

// --- Embedding Operations ---

func (a *OpenAIAdapter) ClientEmbeddingToUnified(r *http.Request) (*UnifiedEmbeddingRequest, error) {
	var openaiReq struct {
		Input []string `json:"input"`
		Model string   `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&openaiReq); err != nil {
		return nil, err
	}

	return &UnifiedEmbeddingRequest{
		Input: openaiReq.Input,
		Model: openaiReq.Model,
	}, nil
}

func (a *OpenAIAdapter) UnifiedEmbeddingToBackend(unifiedReq *UnifiedEmbeddingRequest, backendURL string) (*http.Request, error) {
	openaiReq := map[string]interface{}{
		"input": unifiedReq.Input,
		"model": unifiedReq.Model,
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", backendURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *OpenAIAdapter) BackendEmbeddingToUnified(backendResp *http.Response) (*UnifiedEmbeddingResponse, error) {
	var openaiResp struct {
		Object string `json:"object"`
		Data   []struct {
			Object    string    `json:"object"`
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Model string `json:"model"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(backendResp.Body).Decode(&openaiResp); err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(openaiResp.Data))
	for i, data := range openaiResp.Data {
		embeddings[i] = data.Embedding
	}

	return &UnifiedEmbeddingResponse{
		Embeddings: embeddings,
		Model:      openaiResp.Model,
	}, nil
}

func (a *OpenAIAdapter) UnifiedEmbeddingToClient(unifiedResp *UnifiedEmbeddingResponse, w http.ResponseWriter) error {
	data := make([]map[string]interface{}, len(unifiedResp.Embeddings))
	for i, embedding := range unifiedResp.Embeddings {
		data[i] = map[string]interface{}{
			"object":    "embedding",
			"index":     i,
			"embedding": embedding,
		}
	}

	openaiResp := map[string]interface{}{
		"object": "list",
		"data":   data,
		"model":  unifiedResp.Model,
		"usage": map[string]int{
			"prompt_tokens": len(unifiedResp.Embeddings), // Approximation
			"total_tokens":  len(unifiedResp.Embeddings),
		},
	}

	respBody, err := json.Marshal(openaiResp)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBody)
	return nil
}

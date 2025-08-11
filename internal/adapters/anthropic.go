package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// AnthropicAdapter implements the Adapter interface for the Anthropic API.
type AnthropicAdapter struct{}

// --- Chat Completion Operations ---

func (a *AnthropicAdapter) ClientChatToUnified(r *http.Request) (*UnifiedChatRequest, error) {
	var anthropicReq struct {
		Model      string `json:"model"`
		MaxTokens  int    `json:"max_tokens"`
		Messages   []struct {
			Role    string      `json:"role"`
			Content interface{} `json:"content"` // Can be string or []map[string]interface{}
		} `json:"messages"`
		Tools      []struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			InputSchema map[string]interface{} `json:"input_schema"`
		} `json:"tools"`
		ToolChoice interface{} `json:"tool_choice"`
	}

	if err := json.NewDecoder(r.Body).Decode(&anthropicReq); err != nil {
		return nil, err
	}

	unifiedMessages := make([]UnifiedMessage, len(anthropicReq.Messages))
	for i, msg := range anthropicReq.Messages {
		unifiedMessages[i].Role = msg.Role

		// Anthropic content can be a string or an array of content blocks
		if contentStr, ok := msg.Content.(string); ok {
			unifiedMessages[i].Content = contentStr
		} else if contentBlocks, ok := msg.Content.([]interface{}); ok {
			// Handle text blocks first
			var textContent string
			for _, block := range contentBlocks {
				if blockMap, isMap := block.(map[string]interface{}); isMap {
					if blockType, hasType := blockMap["type"]; hasType && blockType == "text" {
						if text, hasText := blockMap["text"]; hasText {
							textContent += fmt.Sprintf("%v", text)
						}
					}
				}
			}
			unifiedMessages[i].Content = textContent
			// Handle tool_use and tool_result blocks
			for _, block := range contentBlocks {
				if blockMap, isMap := block.(map[string]interface{}); isMap {
					if blockType, hasType := blockMap["type"]; hasType && blockType == "tool_use" {
						if toolUseID, hasID := blockMap["id"]; hasID {
							if name, hasName := blockMap["name"]; hasName {
								if input, hasInput := blockMap["input"]; hasInput {
									// Convert input to JSON string for UnifiedFunctionCall.Arguments
									// Handle all JSON value types (object, array, string, number, boolean, null)
									var inputBytes []byte
									var err error
									
									if inputBytes, err = json.Marshal(input); err != nil {
										// If marshaling fails, convert to string
										inputBytes = []byte(fmt.Sprintf("%v", input))
									}
									
									unifiedMessages[i].ToolCalls = append(unifiedMessages[i].ToolCalls, UnifiedToolCall{
										ID:   fmt.Sprintf("%v", toolUseID),
										Type: "function",
										Function: UnifiedFunctionCall{
											Name:      fmt.Sprintf("%v", name),
											Arguments: string(inputBytes),
										},
									})
								}
							}
						}
					} else if blockType, hasType := blockMap["type"]; hasType && blockType == "tool_result" {
						if toolUseID, hasID := blockMap["tool_use_id"]; hasID {
							if content, hasContent := blockMap["content"]; hasContent {
								// Convert content to JSON string for UnifiedMessage.Content
								contentBytes, _ := json.Marshal(content)
								unifiedMessages[i].ToolCallID = fmt.Sprintf("%v", toolUseID)
								unifiedMessages[i].Content = string(contentBytes)
							}
						}
					}
				}
			}
		}
	}

	// Convert Anthropic tools to unified format
	unifiedTools := make([]UnifiedTool, len(anthropicReq.Tools))
	for i, tool := range anthropicReq.Tools {
		unifiedTools[i] = UnifiedTool{
			Type: "function",
			Function: UnifiedFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		}
	}

	unifiedReq := &UnifiedChatRequest{
		Model:      anthropicReq.Model,
		Messages:   unifiedMessages,
		Tools:      unifiedTools,
		ToolChoice: anthropicReq.ToolChoice,
		// Anthropic does not have a direct 'stream' field in the request body,
		// but it's handled by the HTTP client.
	}

	return unifiedReq, nil
}

func (a *AnthropicAdapter) UnifiedChatToBackend(unifiedReq *UnifiedChatRequest, backendURL string) (*http.Request, error) {
	anthropicMessages := make([]map[string]interface{}, len(unifiedReq.Messages))
	for i, msg := range unifiedReq.Messages {
		anthropicMsg := map[string]interface{}{
			"role": msg.Role,
		}

		if msg.Content != "" {
			anthropicMsg["content"] = msg.Content
		}

		if len(msg.ToolCalls) > 0 {
			// Convert UnifiedToolCalls to Anthropic tool_use blocks
			contentBlocks := []map[string]interface{}{
				{
					"type": "tool_use",
					"id":   msg.ToolCalls[0].ID, // Assuming one tool call per message for simplicity
					"name": msg.ToolCalls[0].Function.Name,
					"input": json.RawMessage(msg.ToolCalls[0].Function.Arguments), // Arguments are JSON string
				},
			}
			anthropicMsg["content"] = contentBlocks
		} else if msg.ToolCallID != "" && msg.Content != "" {
			// Convert Unified tool_result to Anthropic tool_result block
			contentBlocks := []map[string]interface{}{
				{
					"type": "tool_result",
					"tool_use_id": msg.ToolCallID,
					"content": json.RawMessage(msg.Content), // Content is JSON string
				},
			}
			anthropicMsg["content"] = contentBlocks
		}

		anthropicMessages[i] = anthropicMsg
	}

	anthropicReq := map[string]interface{}{
		"model":    unifiedReq.Model,
		"messages": anthropicMessages,
		"max_tokens": 4096, // Anthropic requires max_tokens
	}

	// Handle tools (function definitions) - Anthropic expects these at the top level
	if len(unifiedReq.Tools) > 0 {
		anthropicTools := make([]map[string]interface{}, len(unifiedReq.Tools))
		for i, tool := range unifiedReq.Tools {
			anthropicTools[i] = map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"input_schema": tool.Function.Parameters, // Anthropic uses input_schema
			}
		}
		anthropicReq["tools"] = anthropicTools
	}

	body, err := json.Marshal(anthropicReq)
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

func (a *AnthropicAdapter) BackendChatToUnified(backendResp *http.Response) (*UnifiedChatResponse, error) {
	var anthropicResp struct {
		ID           string        `json:"id"`
		Type         string        `json:"type"`
		Role         string        `json:"role"`
		Content      []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Model        string        `json:"model"`
		StopReason   string        `json:"stop_reason"`
		StopSequence interface{}   `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(backendResp.Body).Decode(&anthropicResp); err != nil {
		return nil, err
	}

	unifiedResp := &UnifiedChatResponse{
		ID:         anthropicResp.ID,
		Model:      anthropicResp.Model,
		Role:       anthropicResp.Role,
		StopReason: anthropicResp.StopReason,
		Usage: UnifiedUsage{
			InputTokens:  anthropicResp.Usage.InputTokens,
			OutputTokens: anthropicResp.Usage.OutputTokens,
		},
	}

	// Extract content
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			unifiedResp.Content += block.Text
		}
	}

	return unifiedResp, nil
}

func (a *AnthropicAdapter) UnifiedChatToClient(unifiedResp *UnifiedChatResponse, w http.ResponseWriter) error {
	// Build content array with text and tool_use blocks
	var contentBlocks []map[string]interface{}
	
	// Add text content if present
	if unifiedResp.Content != "" {
		contentBlocks = append(contentBlocks, map[string]interface{}{
			"type": "text",
			"text": unifiedResp.Content,
		})
	}
	
	// Add tool calls as tool_use blocks
	for _, toolCall := range unifiedResp.ToolCalls {
		// Parse the arguments JSON string back to object
		var input interface{}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
			// If parsing fails, use the raw string
			input = toolCall.Function.Arguments
		}
		
		contentBlocks = append(contentBlocks, map[string]interface{}{
			"type":  "tool_use",
			"id":    toolCall.ID,
			"name":  toolCall.Function.Name,
			"input": input,
		})
	}
	
	anthropicResp := map[string]interface{}{
		"id":          unifiedResp.ID,
		"type":        "message",
		"role":        unifiedResp.Role,
		"content":     contentBlocks,
		"model":       unifiedResp.Model,
		"stop_reason": unifiedResp.StopReason,
		"usage": map[string]int{
			"input_tokens":  unifiedResp.Usage.InputTokens,
			"output_tokens": unifiedResp.Usage.OutputTokens,
		},
	}

	respBody, err := json.Marshal(anthropicResp)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBody)
	return nil
}


// --- Error Translation ---

func (a *AnthropicAdapter) TranslateError(backendResp *http.Response) []byte {
	// In a real implementation, we would parse the backend error
	// and create a new error JSON in the client's expected format.
	// For now, we return a generic error.
	return []byte(`{"error": {"message": "An error occurred at the backend.", "type": "broker_error"}}`)
}

// --- Embedding Operations ---

func (a *AnthropicAdapter) ClientEmbeddingToUnified(r *http.Request) (*UnifiedEmbeddingRequest, error) {
	return nil, fmt.Errorf("Anthropic does not support embedding requests")
}

func (a *AnthropicAdapter) UnifiedEmbeddingToBackend(unifiedReq *UnifiedEmbeddingRequest, backendURL string) (*http.Request, error) {
	return nil, fmt.Errorf("Anthropic does not support embedding requests")
}

func (a *AnthropicAdapter) BackendEmbeddingToUnified(backendResp *http.Response) (*UnifiedEmbeddingResponse, error) {
	return nil, fmt.Errorf("Anthropic does not support embedding responses")
}

func (a *AnthropicAdapter) UnifiedEmbeddingToClient(unifiedResp *UnifiedEmbeddingResponse, w http.ResponseWriter) error {
	return fmt.Errorf("Anthropic does not support embedding responses")
}
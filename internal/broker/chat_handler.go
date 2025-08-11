package broker

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"lmbroker/internal/adapters"
	"lmbroker/internal/broker/workflows"
	"lmbroker/internal/config"
)

// Broker holds the state for the broker, including the configuration
// and a map of initialized adapters.
type Broker struct {
	cfg      *config.Config
	adapters map[string]adapters.Adapter
}

// New creates a new Broker instance.
func New(cfg *config.Config) *Broker {
	// Initialize all the adapters we support.
	initializedAdapters := make(map[string]adapters.Adapter)
	initializedAdapters["openai"] = &adapters.OpenAIAdapter{}
	initializedAdapters["anthropic"] = &adapters.AnthropicAdapter{}

	return &Broker{
		cfg:      cfg,
		adapters: initializedAdapters,
	}
}

// extractModelFromRequest extracts the model name from the request body
func (b *Broker) extractModelFromRequest(r *http.Request) (string, error) {
	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	
	// Restore the body for later use
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	
	// Parse JSON to extract model
	var reqData struct {
		Model string `json:"model"`
	}
	
	if err := json.Unmarshal(body, &reqData); err != nil {
		return "", err
	}
	
	return reqData.Model, nil
}

// findModelConfig finds the model configuration for the specified alias
func (b *Broker) findModelConfig(modelAlias string) (*config.Model, bool) {
	model, ok := b.cfg.Models[modelAlias]
	if !ok {
		return nil, false
	}
	return &model, true
}

// HandleChatCompletions is the main handler for all chat completion requests.
func (b *Broker) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	slog.Info("received chat completion request")
	// 1. Identify the client adapter from the request path.
	var clientAdapterType string
	if r.URL.Path == "/v1/chat/completions" {
		clientAdapterType = "openai"
	} else if r.URL.Path == "/v1/messages" {
		clientAdapterType = "anthropic"
	} else {
		http.Error(w, "unsupported endpoint", http.StatusNotFound)
		return
	}

	// 2. Extract model name from request body
	modelName, err := b.extractModelFromRequest(r)
	if err != nil {
		slog.Error("failed to extract model from request", "error", err)
		http.Error(w, "failed to parse request body", http.StatusBadRequest)
		return
	}
	
	// 3. Find model configuration for this alias
	modelConfig, ok := b.findModelConfig(modelName)
	if !ok {
		slog.Error("no model configuration found", "alias", modelName)
		http.Error(w, "model not supported", http.StatusNotFound)
		return
	}
	slog.Info("routing to provider", "alias", modelName, "target_model", modelConfig.Target.Model, "provider_type", modelConfig.Type, "target_url", modelConfig.Target.URL)

	// 4. Compare client and provider types.
	if clientAdapterType == modelConfig.Type {
		slog.Info("performing passthrough")
		// If they match, use the efficient passthrough workflow.
		workflows.HandlePassthrough(w, r, modelConfig.Target.URL+"chat/completions", modelConfig)
	} else {
		slog.Info("performing translation")
		// If they don't match, use the translation workflow.
		clientAdapter := b.adapters[clientAdapterType]
		providerAdapter := b.adapters[modelConfig.Type]
		workflows.HandleTranslation(w, r, clientAdapter, providerAdapter, modelConfig.Target.URL+"chat/completions", modelConfig)
	}
}

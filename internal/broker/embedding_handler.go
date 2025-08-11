package broker

import (
	"net/http"

	"lmbroker/internal/broker/workflows"
)

// HandleEmbeddings is the main handler for all embedding requests.
func (b *Broker) HandleEmbeddings(w http.ResponseWriter, r *http.Request) {
	// 1. Identify the client adapter from the request path.
	// Embeddings are currently only supported in OpenAI format
	clientAdapterType := "openai"

	// 2. Extract model name from request body
	modelName, err := b.extractModelFromRequest(r)
	if err != nil {
		http.Error(w, "failed to parse request body", http.StatusBadRequest)
		return
	}
	
	// 3. Find model configuration for this alias
	modelConfig, ok := b.findModelConfig(modelName)
	if !ok {
		http.Error(w, "embedding model not supported", http.StatusNotFound)
		return
	}

	// 4. Compare client and provider types.
	if clientAdapterType == modelConfig.Type {
		// If they match, use the efficient passthrough workflow.
		workflows.HandlePassthrough(w, r, modelConfig.Target.URL+"embeddings", modelConfig)
	} else {
		// If they don't match, use the translation workflow.
		clientAdapter := b.adapters[clientAdapterType]
		providerAdapter := b.adapters[modelConfig.Type]
		workflows.HandleEmbeddingTranslation(w, r, clientAdapter, providerAdapter, modelConfig.Target.URL+"embeddings", modelConfig)
	}
}

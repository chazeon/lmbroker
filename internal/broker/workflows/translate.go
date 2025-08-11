package workflows

import (
	"net/http"

	"lmbroker/internal/adapters"
	"lmbroker/internal/config"
)

// HandleTranslation is the workflow for when the client and provider
// speak different API languages. It uses the adapter interfaces to
// perform a four-step translation with model rewriting.
func HandleTranslation(w http.ResponseWriter, r *http.Request, clientAdapter, providerAdapter adapters.Adapter, providerURL string, modelConfig *config.Model) {
	// 1. Decode the client's request into our internal format.
	unifiedReq, err := clientAdapter.ClientChatToUnified(r)
	if err != nil {
		http.Error(w, "failed to translate client request to unified format", http.StatusInternalServerError)
		return
	}

	// 1.5. Rewrite the model field in the unified request
	unifiedReq.Model = modelConfig.Target.Model

	// 2. Encode our internal request into the format for the target provider.
	providerReq, err := providerAdapter.UnifiedChatToBackend(unifiedReq, providerURL)
	if err != nil {
		http.Error(w, "failed to translate unified request to provider format", http.StatusInternalServerError)
		return
	}

	// 2.5. Add API key if configured
	if modelConfig.Target.APIKey != "" {
		providerReq.Header.Set("Authorization", "Bearer "+modelConfig.Target.APIKey)
	}

	// Make the request to the provider.
	client := &http.Client{}
	providerResp, err := client.Do(providerReq)
	if err != nil {
		http.Error(w, "failed to make request to provider", http.StatusBadGateway)
		return
	}
	defer providerResp.Body.Close()

	// 3. Decode the provider's response into our internal format.
	unifiedResp, err := providerAdapter.BackendChatToUnified(providerResp)
	if err != nil {
		http.Error(w, "failed to translate provider response to unified format", http.StatusInternalServerError)
		return
	}

	// 4. Encode our internal response into the format for the original client.
	if err := clientAdapter.UnifiedChatToClient(unifiedResp, w); err != nil {
		// The error is already written to the response writer in the adapter.
		return
	}
}

// HandleEmbeddingTranslation is the workflow for embedding translation
// between different API formats with model rewriting.
func HandleEmbeddingTranslation(w http.ResponseWriter, r *http.Request, clientAdapter, providerAdapter adapters.Adapter, providerURL string, modelConfig *config.Model) {
	// 1. Decode the client's request into our internal format.
	unifiedReq, err := clientAdapter.ClientEmbeddingToUnified(r)
	if err != nil {
		http.Error(w, "failed to translate client embedding request to unified format", http.StatusInternalServerError)
		return
	}

	// 1.5. Rewrite the model field in the unified request
	unifiedReq.Model = modelConfig.Target.Model

	// 2. Encode our internal request into the format for the target provider.
	providerReq, err := providerAdapter.UnifiedEmbeddingToBackend(unifiedReq, providerURL)
	if err != nil {
		http.Error(w, "failed to translate unified embedding request to provider format", http.StatusInternalServerError)
		return
	}

	// 2.5. Add API key if configured
	if modelConfig.Target.APIKey != "" {
		providerReq.Header.Set("Authorization", "Bearer "+modelConfig.Target.APIKey)
	}

	// Make the request to the provider.
	client := &http.Client{}
	providerResp, err := client.Do(providerReq)
	if err != nil {
		http.Error(w, "failed to make embedding request to provider", http.StatusBadGateway)
		return
	}
	defer providerResp.Body.Close()

	// 3. Decode the provider's response into our internal format.
	unifiedResp, err := providerAdapter.BackendEmbeddingToUnified(providerResp)
	if err != nil {
		http.Error(w, "failed to translate provider embedding response to unified format", http.StatusInternalServerError)
		return
	}

	// 4. Encode our internal response into the format for the original client.
	if err := clientAdapter.UnifiedEmbeddingToClient(unifiedResp, w); err != nil {
		// The error is already written to the response writer in the adapter.
		return
	}
}

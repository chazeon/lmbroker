package workflows

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
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
		slog.Error("failed to translate client request to unified format", "error", err)
		http.Error(w, "failed to translate client request to unified format", http.StatusInternalServerError)
		return
	}

	// 1.5. Rewrite the model field in the unified request
	unifiedReq.Model = modelConfig.Target.Model

	// 2. Encode our internal request into the format for the target provider.
	providerReq, err := providerAdapter.UnifiedChatToBackend(unifiedReq, providerURL)
	if err != nil {
		slog.Error("failed to translate unified request to provider format", "error", err)
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
		slog.Error("failed to make request to provider", "error", err)
		http.Error(w, "failed to make request to provider", http.StatusBadGateway)
		return
	}
	defer providerResp.Body.Close()

	// 3. Check if backend returned an error and handle appropriately
	if providerResp.StatusCode >= 400 {
		// Read and preserve the error response body
		bodyBytes, err := io.ReadAll(providerResp.Body)
		if err != nil {
			slog.Error("failed to read error response body", "error", err)
			http.Error(w, "failed to read error response", http.StatusInternalServerError)
			return
		}
		// Restore the body for the adapter
		providerResp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		
		slog.Error("backend returned error", "status", providerResp.StatusCode)
		
		// Translate error directly since we already have the bytes
		var errorResp map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &errorResp); err != nil {
			slog.Error("failed to parse backend error JSON", "error", err)
			errorResp = map[string]interface{}{
				"error": map[string]string{
					"message": "An error occurred at the backend.",
					"type":    "broker_error",
				},
			}
		}
		
		errorBody, _ := json.Marshal(errorResp)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(providerResp.StatusCode)
		w.Write(errorBody)
		return
	}

	// 3. Decode the provider's response into our internal format.
	unifiedResp, err := providerAdapter.BackendChatToUnified(providerResp)
	if err != nil {
		slog.Error("failed to translate provider response to unified format", "error", err)
		http.Error(w, "failed to translate provider response to unified format", http.StatusInternalServerError)
		return
	}

	// 4. Encode our internal response into the format for the original client.
	if err := clientAdapter.UnifiedChatToClient(unifiedResp, w); err != nil {
		slog.Error("failed to translate unified response to client format", "error", err)
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

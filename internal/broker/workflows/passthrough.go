package workflows

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"lmbroker/internal/config"
)



// HandlePassthrough is an optimized workflow for when the client and provider
// speak the same API language. It rewrites the model field and streams the
// request and response directly without translation, which is efficient.
func HandlePassthrough(w http.ResponseWriter, r *http.Request, providerURL string, modelConfig *config.Model) {
	// Read and potentially modify the request body to rewrite the model field
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}

	// Rewrite the model field if the target model is different from the alias
	if modelConfig.Target.Model != modelConfig.Alias {
		var reqData map[string]interface{}
		if err := json.Unmarshal(body, &reqData); err != nil {
			http.Error(w, "failed to parse request JSON", http.StatusBadRequest)
			return
		}
		
		// Replace the model field with the target model
		reqData["model"] = modelConfig.Target.Model
		
		// Marshal back to JSON
		if body, err = json.Marshal(reqData); err != nil {
			http.Error(w, "failed to encode request JSON", http.StatusInternalServerError)
			return
		}
	}

	// Create a new request to the provider.
	backendReq, err := http.NewRequest(r.Method, providerURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "failed to create provider request", http.StatusInternalServerError)
		return
	}

	// Copy headers from the original request to the provider request.
	// Important headers like Content-Type, Authorization, etc., are preserved.
	backendReq.Header = r.Header.Clone()
	
	// Add API key if configured
	if modelConfig.Target.APIKey != "" {
		backendReq.Header.Set("Authorization", "Bearer "+modelConfig.Target.APIKey)
	}

	// Make the request to the backend.
	client := &http.Client{}
	backendResp, err := client.Do(backendReq)
	if err != nil {
		http.Error(w, "failed to make request to backend", http.StatusBadGateway)
		return
	}
	defer backendResp.Body.Close()

	// Copy the backend's response headers to our response writer.
	for key, values := range backendResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set the status code of our response to match the backend's response.
	w.WriteHeader(backendResp.StatusCode)

	// Stream the backend response directly to the client.
	_, _ = io.Copy(w, backendResp.Body)
}

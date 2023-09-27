package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// modifyURL modifies the request URL to point to the configured OpenAI API.
func modifyURL(openAI OpenAISettings, req *http.Request) error {
	u, err := url.Parse(openAI.URL)
	if err != nil {
		log.DefaultLogger.Error("Unable to parse OpenAI URL", "err", err)
		return fmt.Errorf("parse OpenAI URL: %w", err)
	}
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	return nil
}

// openAIProxy is a reverse proxy for OpenAI API calls.
// It modifies the request to point to the configured OpenAI API, returning
// a 400 error if the URL in settings cannot be parsed, then proxies the request
// using the configured API key and OpenAI organization.
type openAIProxy struct {
	settings Settings
	// rp is a reverse proxy handling the modified request. Use this rather than
	// our own client, since it handles things like buffering.
	rp *httputil.ReverseProxy
}

func (a *openAIProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	err := modifyURL(a.settings.OpenAI, req)
	if err != nil {
		// Attempt to write the error as JSON.
		jd, err := json.Marshal(map[string]string{"error": err.Error()})
		if err != nil {
			// We can't write JSON, so just write the error string.
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(err.Error()))
			if err != nil {
				log.DefaultLogger.Error("Unable to write error response", "err", err)
			}
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write(jd)
		if err != nil {
			log.DefaultLogger.Error("Unable to write error response", "err", err)
		}
	}
	a.rp.ServeHTTP(w, req)
}

func newOpenAIProxy(settings Settings) http.Handler {
	director := func(req *http.Request) {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/openai")
		req.Header.Add("Authorization", "Bearer "+settings.OpenAI.apiKey)
		req.Header.Add("OpenAI-Organization", settings.OpenAI.OrganizationID)
	}
	return &openAIProxy{
		settings: settings,
		rp:       &httputil.ReverseProxy{Director: director},
	}
}

// azureOpenAIProxy is a reverse proxy for Azure OpenAI API calls.
// It modifies the request to point to the configured Azure OpenAI API, returning
// a 400 error if the URL in settings cannot be parsed or if the request refers
// to a model without a corresponding deployment in settings. It then proxies the request
// using the configured API key and deployment.
type azureOpenAIProxy struct {
	settings Settings
	// rp is a reverse proxy handling the modified request. Use this rather than
	// our own client, since it handles things like buffering.
	rp *httputil.ReverseProxy
}

func (a *azureOpenAIProxy) modifyRequest(req *http.Request) error {
	err := modifyURL(a.settings.OpenAI, req)
	if err != nil {
		return fmt.Errorf("modify url: %w", err)
	}

	// Read the body so we can determine the deployment to use
	// by mapping the model in the request to a deployment in settings.
	// Azure OpenAI API requires this deployment name in the URL.
	bodyBytes, _ := io.ReadAll(req.Body)
	var requestBody map[string]interface{}
	err = json.Unmarshal(bodyBytes, &requestBody)
	if err != nil {
		return fmt.Errorf("unmarshal request body: %w", err)
	}

	// Find the deployment for the model.
	// Models are mapped to deployments in settings.OpenAI.AzureMapping.
	var deployment string = ""
	for _, v := range a.settings.OpenAI.AzureMapping {
		if val, ok := requestBody["model"].(string); ok && val == v[0] {
			deployment = v[1]
			break
		}
	}

	if deployment == "" {
		return fmt.Errorf("no deployment found for model: %s", requestBody["model"])
	}

	// We've got a deployment, so finish modifying the request.
	req.URL.Path = fmt.Sprintf("/openai/deployments/%s/%s", deployment, strings.TrimPrefix(req.URL.Path, "/openai/v1/"))
	req.Header.Add("api-key", a.settings.OpenAI.apiKey)
	req.URL.RawQuery = "api-version=2023-03-15-preview"

	// Remove extra fields
	delete(requestBody, "model")

	newBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("unmarshal request body: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewBuffer(newBodyBytes))
	req.ContentLength = int64(len(newBodyBytes))
	return nil
}

func (a *azureOpenAIProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	err := a.modifyRequest(req)
	if err != nil {
		// Attempt to write the error as JSON.
		jd, err := json.Marshal(map[string]string{"error": err.Error()})
		if err != nil {
			// We can't write JSON, so just write the error string.
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(err.Error()))
			if err != nil {
				log.DefaultLogger.Error("Unable to write error response", "err", err)
			}
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write(jd)
		if err != nil {
			log.DefaultLogger.Error("Unable to write error response", "err", err)
		}
		return
	}
	a.rp.ServeHTTP(w, req)
}

func newAzureOpenAIProxy(settings Settings) http.Handler {
	// We make all of the actual modifications in ServeHTTP, since they can fail
	// and we want to early-return from HTTP requests in that case.
	director := func(req *http.Request) {}
	return &azureOpenAIProxy{
		settings: settings,
		rp: &httputil.ReverseProxy{
			Director: director,
		},
	}
}

type vectorSearchRequest struct {
	Query      string `json:"query"`
	Collection string `json:"collection"`
	TopK       uint64 `json:"topK"`
}

type vectorSearchResponse struct {
	Results []store.SearchResult `json:"results"`
}

func (app *App) handleVectorSearch(w http.ResponseWriter, req *http.Request) {
	if app.vectorService == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body := vectorSearchRequest{}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.TopK == 0 {
		body.TopK = 10
	}
	results, err := app.vectorService.Search(req.Context(), body.Collection, body.Query, body.TopK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := vectorSearchResponse{Results: results}
	bodyJSON, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//nolint:errcheck // Just do our best to write.
	w.Write(bodyJSON)
}

// registerRoutes takes a *http.ServeMux and registers some HTTP handlers.
func (a *App) registerRoutes(mux *http.ServeMux, settings Settings) {
	switch settings.OpenAI.Provider {
	case openAIProviderOpenAI:
		mux.Handle("/openai/", newOpenAIProxy(settings))
	case openAIProviderAzure:
		mux.Handle("/openai/", newAzureOpenAIProxy(settings))
	default:
		log.DefaultLogger.Warn("Unknown OpenAI provider configured", "provider", settings.OpenAI.Provider)
	}
	mux.HandleFunc("/vector/search", a.handleVectorSearch)
}

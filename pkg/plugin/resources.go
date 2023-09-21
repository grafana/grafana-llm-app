package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

// /api/plugins/app-with-backend/resources/ping

// handlePing is an example HTTP GET resource that returns a {"message": "ok"} JSON response.
func (a *App) handlePing(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"message": "pong"}`)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.DefaultLogger.Info("ping received")
	w.WriteHeader(http.StatusOK)
}

// handleEcho is an example HTTP POST resource that accepts a JSON with a "message" key and
// returns to the client whatever it is sent.
func (a *App) handleEcho(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func newOpenAIProxy() http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			config := httpadapter.PluginConfigFromContext(r.In.Context())
			settings := loadSettings(*config.AppInstanceSettings)
			u, _ := url.Parse(settings.OpenAI.URL)
			r.SetURL(u)
			r.Out.Header.Set("Authorization", "Bearer "+settings.OpenAI.apiKey)
			organizationID := settings.OpenAI.OrganizationID
			r.Out.Header.Set("OpenAI-Organization", organizationID)
			r.Out.URL.Path = strings.TrimPrefix(r.In.URL.Path, "/openai")
			log.DefaultLogger.Info("proxying to url", "url", r.Out.URL.String())
		},
	}
}

func newOpenAIProxy2() http.Handler {
	director := func(req *http.Request) {
		config := httpadapter.PluginConfigFromContext(req.Context())
		settings := loadSettings(*config.AppInstanceSettings)

		u, _ := url.Parse(settings.OpenAI.URL)
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host

		if settings.OpenAI.UseAzure {
			// Map model to deployment
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			var requestBody map[string]interface{}
			json.Unmarshal(bodyBytes, &requestBody)

			deployment := settings.OpenAI.AzureMapping[requestBody["model"].(string)]

			req.URL.Path = fmt.Sprintf("/openai/deployments/%s/%s", deployment, strings.TrimPrefix(req.URL.Path, "/azure"))
			req.Header.Add("api-key", settings.OpenAI.apiKey)
			req.URL.RawQuery = "api-version=2023-03-15-preview"

			// Remove extra fields
			delete(requestBody, "model")

			newBodyBytes, _ := json.Marshal(requestBody)
			req.Body = ioutil.NopCloser(bytes.NewBuffer(newBodyBytes))
			req.ContentLength = int64(len(newBodyBytes))
		} else {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/openai")
			req.Header.Add("Authorization", "Bearer "+settings.OpenAI.apiKey)
			req.Header.Add("OpenAI-Organization", settings.OpenAI.OrganizationID)

		}
	}
	return &httputil.ReverseProxy{Director: director}
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
func (a *App) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ping", a.handlePing)
	mux.HandleFunc("/echo", a.handleEcho)
	mux.Handle("/openai/", newOpenAIProxy())
	mux.HandleFunc("/vector/search", a.handleVectorSearch)
}

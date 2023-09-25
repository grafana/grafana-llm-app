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

func newOpenAIProxy() http.Handler {
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

			var deployment string = ""
			for _, v := range settings.OpenAI.AzureMapping {
				if val, ok := requestBody["model"].(string); ok && val == v[0] {
					deployment = v[1]
					break
				}
			}

			if deployment == "" {
				log.DefaultLogger.Error(fmt.Sprintf("No deployment found for model: %s", requestBody["model"]))
				deployment = "DEPLOYMENT_IS_MISSING"
			}

			req.URL.Path = fmt.Sprintf("/openai/deployments/%s/%s", deployment, strings.TrimPrefix(req.URL.Path, "/openai/v1"))
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
	mux.Handle("/openai/", newOpenAIProxy())
	mux.HandleFunc("/vector/search", a.handleVectorSearch)
}

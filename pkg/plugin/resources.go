package plugin

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/llm/pkg/plugin/vector/store"
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
			u, _ := url.Parse(settings.OpenAIURL)
			r.SetURL(u)
			r.Out.Header.Set("Authorization", "Bearer "+settings.openAIKey)
			organizationID := settings.OpenAIOrganizationID
			r.Out.Header.Set("OpenAI-Organization", organizationID)
			r.Out.URL.Path = strings.TrimPrefix(r.In.URL.Path, "/openai")
			log.DefaultLogger.Info("proxying to url", "url", r.Out.URL.String())
		},
	}
}

type vectorSearchRequest struct {
	Text       string `json:"text"`
	Collection string `json:"collection"`
}

type vectorSearchResponse struct {
	Results []store.SearchResult `json:"results"`
}

func (app *App) handleVectorSearch(w http.ResponseWriter, req *http.Request) {
	body := vectorSearchRequest{}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	results, err := app.vectorService.Search(req.Context(), body.Collection, body.Text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	resp := vectorSearchResponse{Results: results}
	bodyJSON, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

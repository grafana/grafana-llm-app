package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

// modifyURL modifies the request URL to point to the configured OpenAI API.
func modifyURL(openAIUrl string, req *http.Request) error {
	u, err := url.Parse(openAIUrl)
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
	err := modifyURL(a.settings.OpenAI.URL, req)
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
	err := modifyURL(a.settings.OpenAI.URL, req)
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

// grafanaOpenAIProxy is a reverse proxy for OpenAI API calls, that proxies all
// requests via the llm-gateway.
type grafanaOpenAIProxy struct {
	settings Settings
	// rp is a reverse proxy handling the modified request. Use this rather than
	// our own client, since it handles things like buffering.
	rp *httputil.ReverseProxy
}

func (a *grafanaOpenAIProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	err := modifyURL(a.settings.LLMGateway.URL+"/openai", req) // GER: FIXME - not durable to / added
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

func newGrafanaOpenAIProxy(settings Settings) http.Handler {
	director := func(req *http.Request) {
		req.Header.Add("Authorization", "Bearer "+settings.LLMGateway.apiKey)
		req.Header.Add("X-Scope-OrgID", settings.Tenant)
	}

	return &grafanaOpenAIProxy{
		settings: settings,
		rp:       &httputil.ReverseProxy{Director: director},
	}
}

type vectorSearchRequest struct {
	Query      string                 `json:"query"`
	Collection string                 `json:"collection"`
	TopK       uint64                 `json:"topK"`
	Filter     map[string]interface{} `json:"filter"`
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
	results, err := app.vectorService.Search(req.Context(), body.Collection, body.Query, body.TopK, body.Filter)
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

// SaveLLMOptInDataToGrafanaCom persists Grafana-managed LLM opt-in data to grafana.com.
// This is required because provisioned plugins' settings do not survive a restart, and are
// always reset to the state in grafana.com's provisioned-plugins endpoint.
//
// This function (and getPluginID) use the [/api/gnet][] endpoint, which [proxies requests][] to
// the Grafana instance's configured [`GrafanaComUrl` setting][], which is set to the correct
// value for the environment in Hosted Grafana instances.
//
// [/api/gnet]: https://github.com/grafana/grafana/blob/4d8287b319514b750617c20c130ffc424a3ecf2c/pkg/api/api.go#L677
// [proxies requests]: https://github.com/grafana/grafana/blob/4bc582570ef7e713599ab3f2009fa75c27bb8a02/pkg/api/grafana_com_proxy.go#L28
// [`GrafanaComUrl` setting]: https://github.com/grafana/grafana/blob/460be702619428e455ba74f8fb3bb563c1bea43a/pkg/setting/setting.go#L1088
// Handles requests to /save-llm-state endpoint, and pushes state to GCom.
func (app *App) handleSaveLLMState(w http.ResponseWriter, req *http.Request) {
	log.DefaultLogger.Debug("Handling request to save LLM state to gcom..")
	if app.saToken == "" {
		// not available in Grafana < 10.2.3 or if externalServiceAccounts feature flag is not enabled
		log.DefaultLogger.Warn("Service account token not available; cannot save LLM state to gcom")
		return
	}

	if req.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	requestData := SaveLLMStateData{}
	if req.Body != nil {
		defer func() {
			if err := req.Body.Close(); err != nil {
				log.DefaultLogger.Warn("Failed to close response body", "err", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}()
		b, err := io.ReadAll(req.Body)
		if err != nil {
			log.DefaultLogger.Error("Failed to read request body to bytes", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			err := json.Unmarshal(b, &requestData)
			if err != nil {
				log.DefaultLogger.Error("Failed to unmarshal request body to JSON", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		log.DefaultLogger.Warn("Request body is nil")
	}

	notHG := os.Getenv("NOT_HG")
	if notHG != "" {
		log.DefaultLogger.Info("NOT_HG variable found; skipping saving settings to gcom")
		return
	}

	// turn the optIn bool into a string
	var optIn string
	if requestData.OptIn {
		optIn = "1"
	} else {
		optIn = "0"
	}

	user := httpadapter.UserFromContext(req.Context())

	if user == nil || user.Email == "" {
		http.Error(w, "Valid user not found", http.StatusUnauthorized)
		return
	}

	if user.Role != "Admin" {
		http.Error(w, "Only admins can opt-in to Grafana managed LLM", http.StatusForbidden)
		return
	}

	optInData := instanceLLMOptInData{
		IsOptIn:        optIn,
		OptInChangedBy: user.Email,
	}

	// Prepare the request to gcom
	jsonData, err := json.Marshal(optInData)
	if err != nil {
		log.DefaultLogger.Error("Failed to marshal plugin jsonData", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	gcomPath := fmt.Sprintf("/api/gnet/instances/%s", app.settings.Tenant)
	proxyReq, err := http.NewRequestWithContext(req.Context(), "POST", app.grafanaAppURL+gcomPath, bytes.NewReader(jsonData))
	if err != nil {
		log.DefaultLogger.Error("Failed to create http request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Set the headers with the service account token
	proxyReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", app.saToken))
	proxyReq.Header.Set("X-Api-Key", app.settings.GrafanaComAPIKey)
	proxyReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		log.DefaultLogger.Error("Error sending request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.DefaultLogger.Error("Error response from gcom", "status", resp.Status)
		// parse the response body and return it
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			log.DefaultLogger.Error("Failed to read response body to bytes", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Error(w, string(b), resp.StatusCode)
		return
	}
	log.DefaultLogger.Debug("Saved state in gcom", "status", resp.Status)

	// write a success response body since backendSrv.* needs a valid json response body
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(`{"status": "Success"}`))
	if err != nil {
		log.DefaultLogger.Error("Failed to write response body", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// registerRoutes takes a *http.ServeMux and registers some HTTP handlers.
func (a *App) registerRoutes(mux *http.ServeMux, settings Settings) {
	switch settings.OpenAI.Provider {
	case openAIProviderOpenAI:
		mux.Handle("/openai/", newOpenAIProxy(settings))
	case openAIProviderAzure:
		mux.Handle("/openai/", newAzureOpenAIProxy(settings))
	case openAIProviderGrafana:
		mux.Handle("/openai/", newGrafanaOpenAIProxy(settings))
	default:
		log.DefaultLogger.Warn("Unknown OpenAI provider configured", "provider", settings.OpenAI.Provider)
	}
	mux.HandleFunc("/vector/search", a.handleVectorSearch)
	mux.HandleFunc("/save-llm-state", a.handleSaveLLMState)
}

package plugin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

func handleError(w http.ResponseWriter, err error, status int) {
	log.DefaultLogger.Error(err.Error())
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
	w.WriteHeader(status)
	_, err = w.Write(jd)
	if err != nil {
		log.DefaultLogger.Error("Unable to write error response", "err", err)
	}
}

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
		req.SetBasicAuth(settings.Tenant, settings.GrafanaComAPIKey)
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

type llmGatewayResponseData struct {
	Allowed       bool   `json:"allowed"`
	LastUpdatedBy string `json:"lastUpdatedBy"`
}

type llmGatewayResponse struct {
	Status string                 `json:"status"`
	Data   llmGatewayResponseData `json:"data"`
}

func getLLMOptInState(ctx context.Context, settings *Settings) (llmGatewayResponse, error) {
	path := settings.LLMGateway.URL + "/vendor/api/v1/vendors/openai" // hard-coded to openai for now
	proxyReq, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return llmGatewayResponse{}, fmt.Errorf("failed to create http request %w", err)
	}
	// Set the headers with the service account token
	// proxyReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s:%s", app.settings.GComToken))
	proxyReq.SetBasicAuth(settings.Tenant, settings.GrafanaComAPIKey)
	proxyReq.Header.Add("X-Scope-OrgID", settings.Tenant)
	proxyReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		return llmGatewayResponse{}, fmt.Errorf("failed to send request to llm-gateway %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.DefaultLogger.Error("Error response from llm-gateway", "status", resp.Status)
		// parse the response body and return it
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return llmGatewayResponse{}, fmt.Errorf("failed to read response body to bytes %w", err)
		}
		return llmGatewayResponse{}, fmt.Errorf("failed to read state in llm-gateway: %s %s", resp.Status, string(b))
	}
	log.DefaultLogger.Debug("Read opt-in state from llm-gateway", "status", resp.Status)

	body := llmGatewayResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return llmGatewayResponse{}, err
	}

	return body, nil
}

func (app *App) handleGetLLMOptInState(w http.ResponseWriter, req *http.Request) {
	log.DefaultLogger.Debug("Handling request to get LLM state from llm-gateway..")

	llmState, err := getLLMOptInState(req.Context(), app.settings)
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}

	bodyJSON, err := json.Marshal(llmState)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(bodyJSON)
	if err != nil {
		handleError(w, fmt.Errorf("failed to write response body %w", err), http.StatusInternalServerError)
		return
	}
}

// InstanceLLMOptInData contains the LLM opt-in state and the last user who changed it
type setLLMOptInState struct {
	Allowed   bool   `json:"allowed"`
	UserEmail string `json:"userEmail"`
}

type llmOptInState struct {
	Allowed *bool `json:"allowed"`
}

// handleSaveLLMOptInState persists Grafana-managed LLM opt-in state to the llm-gateway.
func (app *App) handleSaveLLMOptInState(w http.ResponseWriter, req *http.Request) {
	log.DefaultLogger.Debug("Handling request to save LLM state to llm-gateway..")

	// Read the request body
	if req.Body == nil {
		log.DefaultLogger.Warn("Request body is nil")
		handleError(w, errors.New("request body required"), http.StatusBadRequest)
		return
	}
	requestData := llmOptInState{}
	defer func() {
		if err := req.Body.Close(); err != nil {
			handleError(w, fmt.Errorf("failed to close request body %w", err), http.StatusInternalServerError)
			return
		}
	}()
	b, err := io.ReadAll(req.Body)
	if err != nil {
		handleError(w, fmt.Errorf("failed to read request body to bytes %w", err), http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(b, &requestData)
	if err != nil {
		handleError(w, fmt.Errorf("failed to unmarshal request body to JSON %w", err), http.StatusInternalServerError)
		return
	}
	if requestData.Allowed == nil {
		handleError(w, errors.New("`allowed` field is required"), http.StatusBadRequest)
		return
	}

	user := httpadapter.UserFromContext(req.Context())

	if user == nil || user.Email == "" {
		handleError(w, fmt.Errorf("valid user not found (please sign in and retry)"), http.StatusUnauthorized)
		return
	}

	if user.Role != "Admin" {
		handleError(w, fmt.Errorf("only admins can change opt-in state for the Grafana managed LLM"), http.StatusForbidden)
		return
	}

	newOptInState := setLLMOptInState{
		Allowed:   *requestData.Allowed,
		UserEmail: user.Email,
	}

	// Prepare the request to llm-gateway
	jsonData, err := json.Marshal(newOptInState)
	if err != nil {
		handleError(w, fmt.Errorf("failed to marshal plugin jsonData %w", err), http.StatusInternalServerError)
		return
	}

	path := app.settings.LLMGateway.URL + "/vendor/api/v1/vendors/openai" // hard-coded to openai for now
	proxyReq, err := http.NewRequestWithContext(req.Context(), "POST", path, bytes.NewReader(jsonData))
	if err != nil {
		handleError(w, fmt.Errorf("failed to create http request %w", err), http.StatusBadRequest)
		return
	}
	// Basic auth for use with Grafana Cloud.
	proxyReq.SetBasicAuth(app.settings.Tenant, app.settings.GrafanaComAPIKey)
	// X-Scope-OrgID for use in local settings.
	proxyReq.Header.Add("X-Scope-OrgID", app.settings.Tenant)
	proxyReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		handleError(w, fmt.Errorf("failed to send request to llm-gateway %w", err), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		log.DefaultLogger.Error("Error response from llm-gateway", "status", resp.Status)
		// parse the response body and return it
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			handleError(w, fmt.Errorf("failed to read response body to bytes %w", err), http.StatusInternalServerError)
			return
		}
		handleError(w, fmt.Errorf("failed to save state in llm-gateway: %s %s", resp.Status, string(b)), http.StatusInternalServerError)
		return
	}
	log.DefaultLogger.Debug("Saved state in llm-gateway", "status", resp.Status)

	// write a success response body since backendSrv.* needs a valid json response body
	w.WriteHeader(http.StatusOK)
	// No need (or real ability) to handle an error after already writing a success header.
	_, _ = w.Write([]byte(`{"status": "Success"}`))
}

func (a *App) handleLLMState(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		a.handleGetLLMOptInState(w, req)
	case "POST":
		a.handleSaveLLMOptInState(w, req)
	default:
		handleError(w, fmt.Errorf("method not allowed: %s", req.Method), http.StatusMethodNotAllowed)
		return
	}
}

// provisionedPlugin is the response returned by a call to grafana.com's provisioned-plugin's endpoint
// for an specific instance's plugin.
type provisionedPlugin struct {
	ID int `json:"id"`
}

// getPluginID gets the *ID* of the *provisioned plugin* from grafana.com.
// Note that this differs to the plugin ID referred to by the `backend.CallResourceRequest`,
// which is 'grafana-llm-app'
func getPluginID(ctx context.Context, slug string, grafanaAppURL string, saToken string, gcomAPIKey string) (int, error) {
	gcomPath := "/api/gnet/instances/" + slug + "/provisioned-plugins/grafana-llm-app"
	req, err := http.NewRequestWithContext(ctx, "GET", grafanaAppURL+gcomPath, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", saToken))
	req.Header.Set("X-Api-Key", gcomAPIKey)
	if err != nil {
		return 0, fmt.Errorf("create http request: %w", err)
	}
	respBody, err := doRequest(req)
	if err != nil {
		return 0, err
	}
	plugin := provisionedPlugin{}
	err = json.Unmarshal(respBody, &plugin)
	if err != nil {
		return 0, fmt.Errorf("unmarshal json: %w", err)
	}
	return plugin.ID, nil
}

type pluginSettings struct {
	JSONData       map[string]interface{} `json:"jsonData"`
	SecureJSONData map[string]string      `json:"secureJsonData"`
}

func (a *App) insertProvisionedToken(body io.ReadCloser) (url.Values, error) {
	// Read the request body
	if body == nil {
		return nil, errors.New("request body required")
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body to bytes %w", err)
	}
	// Unmarshal the request body to JSON
	var requestData pluginSettings
	err = json.Unmarshal(b, &requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request body to JSON %w", err)
	}

	// Insert the provisioned token into the request body
	requestData.SecureJSONData["base64EncodedAccessToken"] = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", a.settings.Tenant, a.settings.GrafanaComAPIKey)))
	// Marshal the request body back to JSON
	jsonData, err := json.Marshal(requestData.JSONData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request jsonData %w", err)
	}

	secureJSONData, err := json.Marshal(requestData.SecureJSONData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request secureJSONData %w", err)
	}

	newBody := url.Values{}
	newBody.Set("jsonData", string(jsonData))
	newBody.Set("secureJsonData", string(secureJSONData))

	return newBody, nil
}

func (a *App) handleSavePluginSettings(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		handleError(w, fmt.Errorf("method not allowed: %s", req.Method), http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	if req.Body == nil {
		log.DefaultLogger.Warn("Request body is nil")
		handleError(w, errors.New("request body required"), http.StatusBadRequest)
		return
	}

	if !a.settings.EnableGrafanaManagedLLM {
		log.DefaultLogger.Info("Plugin not provisioned; skipping saving settings to grafana.com")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "Success"}`))
		return
	}

	log.DefaultLogger.Debug("Getting provisioned plugin ID from grafana.com")
	pluginID, err := getPluginID(req.Context(), a.settings.Tenant, a.grafanaAppURL, a.saToken, a.settings.GrafanaComAPIKey)
	if err != nil {
		handleError(w, fmt.Errorf("get plugin ID: %w", err), http.StatusInternalServerError)
		return
	}

	gcomPath := fmt.Sprintf("/api/gnet/instances/%s/provisioned-plugins/%d", a.settings.Tenant, pluginID)
	newReqBody, err := a.insertProvisionedToken(req.Body)
	if err != nil {
		handleError(w, fmt.Errorf("insert provisioned token: %w", err), http.StatusInternalServerError)
		return
	}
	gcomReq, err := http.NewRequestWithContext(req.Context(), "POST", a.grafanaAppURL+gcomPath, strings.NewReader(newReqBody.Encode()))
	if err != nil {
		handleError(w, fmt.Errorf("create gcom request: %w", err), http.StatusInternalServerError)
		return
	}
	gcomReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.saToken))
	gcomReq.Header.Set("X-Api-Key", a.settings.GrafanaComAPIKey)
	gcomReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err = doRequest(gcomReq)
	if err != nil {
		handleError(w, fmt.Errorf("saving plugin setting to gcom: %w", err), http.StatusInternalServerError)
		return
	}

	// write a success response body since backendSrv.* needs a valid json response body
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "Success"}`))
}

func doRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send http request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read http response: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return respBody, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}
	return respBody, nil
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
	mux.HandleFunc("/grafana-llm-state", a.handleLLMState)
	mux.HandleFunc("/save-plugin-settings", a.handleSavePluginSettings)
}

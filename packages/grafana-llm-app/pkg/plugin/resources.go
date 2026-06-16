package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
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
	defer resp.Body.Close() //nolint:errcheck

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

	user := backend.UserFromContext(req.Context())

	devMode := os.Getenv("DEV_MODE") != ""
	if devMode {
		user.Email = "admin@localhost.com"
	}

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
	defer resp.Body.Close() //nolint:errcheck
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

func (a *App) mergeSecureJSONData(b []byte) (url.Values, error) {
	// Unmarshal the request body to JSON
	var requestData pluginSettings
	err := json.Unmarshal(b, &requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request body to JSON %w", err)
	}

	// Insert existing plugin secureJSONData fields if missing from request
	for key, value := range a.settings.DecryptedSecureJSONData {
		if _, exists := requestData.SecureJSONData[key]; !exists {
			requestData.SecureJSONData[key] = value
		}
	}

	// Update mandatory fields
	requestData.SecureJSONData[encodedTenantAndTokenKey] = a.settings.DecryptedSecureJSONData[encodedTenantAndTokenKey]

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

	// DEV_MODE is only used for local dev to avoid sending request to gcom
	devMode := os.Getenv("DEV_MODE")
	if !a.settings.EnableGrafanaManagedLLM || devMode != "" {
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

	// Read the request body
	if req.Body == nil {
		handleError(w, errors.New("request body required"), http.StatusBadRequest)
		return
	}
	b, err := io.ReadAll(req.Body)
	if err != nil {
		handleError(w, fmt.Errorf("failed to read request body to bytes %w", err), http.StatusInternalServerError)
		return
	}
	newReqBody, err := a.mergeSecureJSONData(b)
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

	log.DefaultLogger.Debug("Sending request to Grafana.com", "url", gcomReq.URL)
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
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send http request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read http response: %w", err)
	}
	if resp.StatusCode == 409 {
		// Retry usually helps if this happens
		return doRequest(req)
	} else if resp.StatusCode/100 != 2 {
		return respBody, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}
	return respBody, nil
}

func (a *App) handleModels() http.HandlerFunc {
	llmProvider, err := createProvider(a.settings)

	return func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			log.DefaultLogger.Error("LLM provider has invalid configuration", "err", err)
			handleError(w, errors.New("LLM provider has invalid configuration"), http.StatusUnprocessableEntity)
		}
		if llmProvider == nil {
			handleError(w, errors.New("must configure an LLM provider"), http.StatusUnprocessableEntity)
			return
		}
		if r.Method != http.MethodGet {
			handleError(w, errors.New("only GET method allowed"), http.StatusMethodNotAllowed)
			return
		}
		models, err := llmProvider.Models(r.Context())
		if errors.Is(err, errBadRequest) {
			handleError(w, err, http.StatusBadRequest)
		} else if err != nil {
			handleError(w, err, http.StatusInternalServerError)
			return
		}

		resp, err := json.Marshal(models)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		// Do our best to write.
		//nolint:errcheck
		w.Write(resp)
	}
}

func (a *App) handleChatCompletionsStream(
	ctx context.Context,
	llmProvider LLMProvider,
	req ChatCompletionRequest,
	w http.ResponseWriter,
) {
	log.DefaultLogger.Info("handling stream request")
	c, err := llmProvider.ChatCompletionStream(ctx, req)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	var writeErr error
	for resp := range c {
		// Clear the queue without doing anything in the event of a failed write or error.
		if writeErr != nil {
			continue
		}
		if resp.Error != nil {
			writeErr = handleStreamError(w, resp.Error, http.StatusInternalServerError)
			continue
		}
		if a.ignoreResponsePadding {
			resp.ignorePadding = true
		}
		chunk, err := json.Marshal(resp)
		if err != nil {
			writeErr = handleStreamError(w, errors.New("failed to marshal streaming response"), http.StatusInternalServerError)
			continue
		}

		// Write the data as a SSE. If writing fails we finish reading the
		// channel to avoid a memory leak and handle the error outside of the
		// loop.
		data := "data: " + string(chunk) + "\n\n"
		_, writeErr = w.Write([]byte(data))
	}
	if writeErr != nil {
		log.DefaultLogger.Warn("failed to write stream", "err", writeErr)
		return
	}
	// Channel has closed, send a DONE SSE.
	//nolint:errcheck
	w.Write([]byte("data: [DONE]\n\n"))
}

func handleStreamError(w http.ResponseWriter, err error, code int) error {
	// See if the error we passed in is an openai error, if so pass that back
	// to the user, otherwise fill in the information as best we can.
	oaiErr := &openai.APIError{}
	if !errors.As(err, &oaiErr) {
		oaiErr = &openai.APIError{
			Code:           code,
			Message:        err.Error(),
			HTTPStatusCode: code,
		}
	}

	errResp := openai.ErrorResponse{
		Error: oaiErr,
	}
	resp, err := json.Marshal(errResp)
	if err != nil {
		return fmt.Errorf("marshaling error response: %w", err)
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", string(resp))
	if err != nil {
		return fmt.Errorf("writing error response: %w", err)
	}
	return nil
}

func (a *App) handleChatCompletions() http.HandlerFunc {
	llmProvider, err := createProvider(a.settings)

	return func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			handleError(w, errors.New("LLM provider has invalid configuration"), http.StatusUnprocessableEntity)
		}
		if llmProvider == nil {
			handleError(w, errors.New("must configure an LLM provider"), http.StatusUnprocessableEntity)
			return
		}
		if r.Method != http.MethodPost {
			handleError(w, errors.New("only POST method allowed"), http.StatusMethodNotAllowed)
			return
		}
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError)
			return
		}
		req := ChatCompletionRequest{}
		err = json.Unmarshal(reqBody, &req)
		if err != nil {
			handleError(w, fmt.Errorf("could not decode request: %w", err), http.StatusBadRequest)
			return
		}

		if req.Stream {
			a.handleChatCompletionsStream(r.Context(), llmProvider, req, w)
			return
		}

		resp, err := llmProvider.ChatCompletion(r.Context(), req)
		if errors.Is(err, errBadRequest) {
			handleError(w, err, http.StatusBadRequest)
		} else if err != nil {
			handleError(w, err, http.StatusInternalServerError)
			return
		}

		respBody, err := json.Marshal(resp)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		//nolint:errcheck
		w.Write(respBody)
	}
}

// registerRoutes takes a *http.ServeMux and registers some HTTP handlers.
func (a *App) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/openai/v1/models", a.handleModels())                    // Deprecated
	mux.HandleFunc("/openai/v1/chat/completions", a.handleChatCompletions()) // Deprecated
	mux.HandleFunc("/llm/v1/chat/completions", a.handleChatCompletions())
	mux.HandleFunc("/llm/v1/models", a.handleModels())
	mux.HandleFunc("/vector/search", a.handleVectorSearch)
	mux.HandleFunc("/grafana-llm-state", a.handleLLMState)
	mux.HandleFunc("/save-plugin-settings", a.handleSavePluginSettings)

	if a.mcpServer != nil {
		log.DefaultLogger.Debug("Registering Grafana MCP endpoints on /mcp/grafana")
		mux.HandleFunc("/mcp/grafana", a.mcpServer.HTTPServer.ServeHTTP)
	}
}

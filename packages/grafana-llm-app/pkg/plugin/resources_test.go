package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
)

// mockCallResourceResponseSender implements backend.CallResourceResponseSender
// for use in tests.
type mockCallResourceResponseSender struct {
	response *backend.CallResourceResponse
}

// Send sets the received *backend.CallResourceResponse to s.response
func (s *mockCallResourceResponseSender) Send(response *backend.CallResourceResponse) error {
	s.response = response
	return nil
}

// TestCallResource tests CallResource calls, using backend.CallResourceRequest and backend.CallResourceResponse.
// This ensures the httpadapter for CallResource works correctly.
func TestCallResource(t *testing.T) {
	ctx := context.Background()
	// Initialize app
	inst, err := NewApp(ctx, backend.AppInstanceSettings{})
	if err != nil {
		t.Fatalf("new app: %s", err)
	}
	if inst == nil {
		t.Fatal("inst must not be nil")
	}
	app, ok := inst.(*App)
	if !ok {
		t.Fatal("inst must be of type *App")
	}

	// Set up and run test cases
	for _, tc := range []struct {
		name string

		method  string
		headers map[string][]string
		path    string
		body    []byte

		expStatus int
		expBody   []byte
	}{
		{
			name:      "get non existing handler 404",
			method:    http.MethodGet,
			path:      "not_found",
			expStatus: http.StatusNotFound,
		},
		{
			name:   "get mcp grafana handler 200",
			method: http.MethodPost,
			headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			body:      []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`),
			path:      "/mcp/grafana",
			expStatus: http.StatusOK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Request by calling CallResource. This tests the httpadapter.
			var r mockCallResourceResponseSender
			err = app.CallResource(ctx, &backend.CallResourceRequest{
				Method:  tc.method,
				Headers: tc.headers,
				Path:    tc.path,
				Body:    tc.body,
			}, &r)
			if err != nil {
				t.Fatalf("CallResource error: %s", err)
			}
			if r.response == nil {
				t.Fatal("no response received from CallResource")
			}
			if tc.expStatus > 0 && tc.expStatus != r.response.Status {
				t.Errorf("response status should be %d, got %d", tc.expStatus, r.response.Status)
			}
			if len(tc.expBody) > 0 {
				if tb := bytes.TrimSpace(r.response.Body); !bytes.Equal(tb, tc.expBody) {
					t.Errorf("response body should be %s, got %s", tc.expBody, tb)
				}
			}
		})
	}
}

func TestMergeSecureJSONData(t *testing.T) {
	ctx := context.Background()
	// Initialize app
	inst, err := NewApp(ctx, backend.AppInstanceSettings{
		DecryptedSecureJSONData: map[string]string{
			openAIKey:                "abcd1234",
			encodedTenantAndTokenKey: "MTIzOmFiY2QxMjM0",
		},
	})
	if err != nil {
		t.Fatalf("new app: %s", err)
	}
	if inst == nil {
		t.Fatal("inst must not be nil")
	}
	app, ok := inst.(*App)
	if !ok {
		t.Fatal("inst must be of type *App")
	}

	// Set up and run test cases
	for _, tc := range []struct {
		name string

		secureJSONData []byte

		expMerged map[string]string
	}{
		{
			name: "empty",

			secureJSONData: []byte(`{}`),

			expMerged: map[string]string{
				openAIKey:                "abcd1234",
				encodedTenantAndTokenKey: "MTIzOmFiY2QxMjM0",
			},
		},
		{
			name: "override",

			secureJSONData: []byte(`{"openAIKey": "value1"}`),

			expMerged: map[string]string{
				openAIKey:                "value1",
				encodedTenantAndTokenKey: "MTIzOmFiY2QxMjM0",
			},
		},
		{
			name: "addition",

			secureJSONData: []byte(`{"someOtherKey": "test"}`),

			expMerged: map[string]string{
				openAIKey:                "abcd1234",
				encodedTenantAndTokenKey: "MTIzOmFiY2QxMjM0",
				"someOtherKey":           "test",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(`{"secureJsonData": ` + string(tc.secureJSONData) + `}`)
			merged, err := app.mergeSecureJSONData(body)

			require.NoError(t, err)

			secureJsonString := merged.Get("secureJsonData")
			var updatedSecureJson map[string]string
			err = json.Unmarshal([]byte(secureJsonString), &updatedSecureJson)
			require.NoError(t, err)

			require.Equal(t, tc.expMerged, updatedSecureJson)
		})
	}
}

type mockServer struct {
	server      *httptest.Server
	request     *http.Request
	requestBody []byte
}

func newMockOpenAIServer() *mockServer {
	server := &mockServer{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.request = r
		server.requestBody, _ = io.ReadAll(r.Body)
		streaming := r.Header.Get("Accept") == "text/event-stream"
		if streaming {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			//nolint:errcheck
			w.Write([]byte("data: {}\n\n"))
			//nolint:errcheck
			w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		//nolint:errcheck
		w.Write([]byte("{}"))

	})
	server.server = httptest.NewServer(handler)
	return server
}

func TestCallOpenAIProxy(t *testing.T) {
	// Set up and run test cases
	for _, tc := range []struct {
		name string

		settings Settings
		apiKey   string

		method string
		path   string
		body   []byte

		// Expected proxied request values.
		expNilRequest bool
		expReqHeaders http.Header
		expReqPath    string
		expReqBody    []byte

		// Expected proxied response values.
		expStatus int
		expBody   []byte
	}{
		{
			name: "openai",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       ProviderTypeOpenAI,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "gpt-3.5-turbo", "messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"Authorization":       {"Bearer abcd1234"},
				"OpenAI-Organization": {"myOrg"},
			},
			expReqPath: "/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "messages": [{"content":"some stuff"}]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "openai - empty model",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       ProviderTypeOpenAI,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"Authorization":       {"Bearer abcd1234"},
				"OpenAI-Organization": {"myOrg"},
			},
			expReqPath: "/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "messages": [{"content":"some stuff"}]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "openai - streaming",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       ProviderTypeOpenAI,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "base", "stream": true, "messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"Authorization":       {"Bearer abcd1234"},
				"OpenAI-Organization": {"myOrg"},
			},
			expReqPath: "/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "stream": true, "messages": [{"content":"some stuff"}]}`),

			expStatus: http.StatusOK,

			// We need to use regular strings rather than raw strings here otherwise the double
			// newlines (required by the SSE spec) are escaped.
			expBody: []byte("data: {\"id\":\"\",\"object\":\"\",\"created\":0,\"model\":\"\",\"choices\":null,\"system_fingerprint\":\"\"}\n\ndata: [DONE]\n\n"),
		},
		{
			name: "openai - streaming - empty model",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       ProviderTypeOpenAI,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"stream": true, "messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"Authorization":       {"Bearer abcd1234"},
				"OpenAI-Organization": {"myOrg"},
			},
			expReqPath: "/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "stream": true, "messages": [{"content":"some stuff"}]}`),

			expStatus: http.StatusOK,

			// We need to use regular strings rather than raw strings here otherwise the double
			// newlines (required by the SSE spec) are escaped.
			expBody: []byte("data: {\"id\":\"\",\"object\":\"\",\"created\":0,\"model\":\"\",\"choices\":null,\"system_fingerprint\":\"\"}\n\ndata: [DONE]\n\n"),
		},
		{
			name: "azure",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       ProviderTypeAzure,
					AzureMapping: [][]string{
						{"gpt-3.5-turbo", "gpt-35-turbo"},
					},
				},
			},

			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "gpt-3.5-turbo", "messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"api-key": {"abcd1234"},
			},
			expReqPath: "/openai/deployments/gpt-35-turbo/chat/completions",
			// the 'model' field should have been removed.
			expReqBody: []byte(`{"messages":[{"content":"some stuff"}]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "azure - abstract model",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       ProviderTypeAzure,
					AzureMapping: [][]string{
						{"gpt-3.5-turbo", "gpt-35-turbo"},
					},
				},
			},

			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "base", "messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"api-key": {"abcd1234"},
			},
			expReqPath: "/openai/deployments/gpt-35-turbo/chat/completions",
			// the 'model' field should have been removed.
			expReqBody: []byte(`{"messages":[{"content":"some stuff"}]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "azure - empty model",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       ProviderTypeAzure,
					AzureMapping: [][]string{
						{"gpt-3.5-turbo", "gpt-35-turbo"},
					},
				},
			},

			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"api-key": {"abcd1234"},
			},
			expReqPath: "/openai/deployments/gpt-35-turbo/chat/completions",
			// the 'model' field should have been removed.
			expReqBody: []byte(`{"messages":[{"content":"some stuff"}]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "azure invalid deployment",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       ProviderTypeAzure,
					AzureMapping: [][]string{
						{"gpt-3.5-turbo", "gpt-35-turbo"},
					},
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			// note no gpt-4 in AzureMapping.
			body: []byte(`{"model": "gpt-4-turbo", "messages": [{"content":"some stuff"}]}`),

			expNilRequest: true,

			expStatus: http.StatusBadRequest,
		},
		{
			name: "grafana-managed llm gateway - opt in not set",

			settings: Settings{
				Tenant:           "123",
				GrafanaComAPIKey: "abcd1234",
				OpenAI: OpenAISettings{
					Provider: ProviderTypeGrafana,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "gpt-3.5-turbo", "messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"Authorization": {"Bearer 123:abcd1234"},
				"X-Scope-OrgID": {"123"},
			},
			expReqPath: "/llm/openai/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "messages": [{"content":"some stuff"}]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "grafana-managed llm gateway",

			settings: Settings{
				Tenant:           "123",
				GrafanaComAPIKey: "abcd1234",
				OpenAI: OpenAISettings{
					Provider: ProviderTypeGrafana,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "gpt-3.5-turbo", "messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"Authorization": {"Bearer 123:abcd1234"},
				"X-Scope-OrgID": {"123"},
			},
			expReqPath: "/llm/openai/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "messages": [{"content":"some stuff"]}}`),

			expStatus: http.StatusOK,
		},
		{
			name: "grafana-managed llm gateway - empty model",

			settings: Settings{
				Tenant:           "123",
				GrafanaComAPIKey: "abcd1234",
				OpenAI: OpenAISettings{
					Provider: ProviderTypeGrafana,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"messages": [{"content":"some stuff"}]}`),

			expReqHeaders: http.Header{
				"Authorization": {"Bearer 123:abcd1234"},
				"X-Scope-OrgID": {"123"},
			},
			expReqPath: "/llm/openai/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "messages": [{"content":"some stuff"]}}`),

			expStatus: http.StatusOK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			// Start up a mock server that just captures the request and sends a 200 OK response.
			server := newMockOpenAIServer()

			// Update the OpenAI/LLMGateway URL with the mock server's URL.
			if tc.settings.OpenAI.Provider == ProviderTypeGrafana {
				// Make sure our tests work when the LLM gateway is at a subpath.
				tc.settings.LLMGateway.URL = server.server.URL + "/llm"
			} else {
				tc.settings.OpenAI.URL = server.server.URL
			}

			// Initialize app
			jsonData, err := json.Marshal(tc.settings)
			if err != nil {
				t.Fatalf("json marshal: %s", err)
			}
			// Set the API keys
			appSettings := backend.AppInstanceSettings{
				JSONData: jsonData,
				DecryptedSecureJSONData: map[string]string{
					"openAIKey": tc.apiKey,
				},
			}
			inst, err := NewApp(ctx, appSettings)
			if err != nil {
				t.Fatalf("new app: %s", err)
			}
			if inst == nil {
				t.Fatal("inst must not be nil")
			}
			app, ok := inst.(*App)
			if !ok {
				t.Fatal("inst must be of type *App")
			}

			app.ignoreResponsePadding = true

			var r mockCallResourceResponseSender
			err = app.CallResource(ctx, &backend.CallResourceRequest{
				PluginContext: backend.PluginContext{
					AppInstanceSettings: &appSettings,
				},
				Method: tc.method,
				Path:   tc.path,
				Body:   tc.body,
			}, &r)
			if err != nil {
				t.Fatalf("CallResource error: %s", err)
			}
			if r.response == nil {
				t.Fatal("no response received from CallResource")
			}

			// Proxied request assertions.
			if tc.expNilRequest && server.request != nil {
				t.Fatalf("request should not have been proxied, got %v", server.request)
			}
			if len(tc.expReqHeaders) > 0 {
				for k, values := range tc.expReqHeaders {
					if got := server.request.Header.Get(k); got != values[0] {
						t.Errorf("proxied request header %s should have value %s, got %s", k, values[0], got)
					}
				}
			}
			if tc.expReqPath != "" {
				if server.request.URL.Path != tc.expReqPath {
					t.Errorf("proxied request path should be %s, got %s", tc.expReqPath, server.request.URL.Path)
				}
			}

			// Response assertions.
			if tc.expStatus > 0 && tc.expStatus != r.response.Status {
				t.Errorf("response status should be %d, got %d", tc.expStatus, r.response.Status)
			}
			if len(tc.expBody) > 0 {
				if !bytes.Equal(r.response.Body, tc.expBody) {
					t.Errorf("response body should be %s, got %s", tc.expBody, r.response.Body)
				}
			}
		})
	}
}

func TestChatCompletionsFinalModel(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name string

		model    Model
		settings backend.AppInstanceSettings

		expectedModel string
	}{
		{
			name: "anthropic-no-mapping-base",

			model: ModelBase,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "anthropic"
				}`),
			},

			expectedModel: defaultModelSettings(ProviderTypeAnthropic).Mapping[ModelBase],
		},
		{
			name: "anthropic-no-mapping-large",

			model: ModelLarge,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "anthropic"
				}`),
			},
			expectedModel: defaultModelSettings(ProviderTypeAnthropic).Mapping[ModelLarge],
		},
		{
			name: "anthropic-partial-mapping-base",

			model: ModelBase,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "anthropic",
					"models": {
						"mapping": {
							"base": "claude-4-sonnet"
						}
					}
				}`),
			},
			expectedModel: "claude-4-sonnet",
		},
		{
			name: "anthropic-partial-mapping-large",

			model: ModelLarge,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "anthropic",
					"models": {
						"mapping": {
							"base": "claude-4-sonnet"
						}
					}
				}`),
			},
			expectedModel: defaultModelSettings(ProviderTypeAnthropic).Mapping[ModelLarge],
		},
		{
			name: "anthropic-full-mapping-base",

			model: ModelBase,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "anthropic",
					"models": {
						"mapping": {
							"base": "claude-4-sonnet",
							"large": "claude-4-sonnet"
						}
					}
				}`),
			},
			expectedModel: "claude-4-sonnet",
		},
		{
			name: "anthropic-full-mapping-large",

			model: ModelLarge,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "anthropic",
					"models": {
						"mapping": {
							"base": "claude-4-sonnet",
							"large": "claude-4-sonnet"
						}
					}
				}`),
			},
			expectedModel: "claude-4-sonnet",
		},
		{
			name: "openai-no-mapping-base",

			model: ModelBase,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "openai"
				}`),
			},
			expectedModel: defaultModelSettings(ProviderTypeOpenAI).Mapping[ModelBase],
		},
		{
			name: "openai-no-mapping-large",

			model: ModelLarge,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "openai"
				}`),
			},
			expectedModel: defaultModelSettings(ProviderTypeOpenAI).Mapping[ModelLarge],
		},
		{
			name: "openai-partial-mapping-base",

			model: ModelBase,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "openai",
					"models": {
						"mapping": {
							"base": "gpt-4"
						}
					}
				}`),
			},
			expectedModel: "gpt-4",
		},
		{
			name: "openai-partial-mapping-large",

			model: ModelLarge,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "openai",
					"models": {
						"mapping": {
							"base": "gpt-4"
						}
					}
				}`),
			},
			expectedModel: defaultModelSettings(ProviderTypeOpenAI).Mapping[ModelLarge],
		},
		{
			name: "openai-full-mapping-base",

			model: ModelBase,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "openai",
					"models": {
						"mapping": {
							"base": "gpt-4",
							"large": "gpt-4"
						}
					}
				}`),
			},
			expectedModel: "gpt-4",
		},
		{
			name: "openai-full-mapping-large",

			model: ModelLarge,
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"provider": "openai",
					"models": {
						"mapping": {
							"base": "gpt-4",
							"large": "gpt-4"
						}
					}
				}`),
			},
			expectedModel: "gpt-4",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newMockOpenAIServer()
			settings := withProviderURL(t, tc.settings, server.server.URL)
			inst, err := NewApp(ctx, settings)
			require.NoError(t, err)
			require.NotNil(t, inst)
			app, ok := inst.(*App)
			require.True(t, ok)

			var r mockCallResourceResponseSender
			req := &backend.CallResourceRequest{
				PluginContext: backend.PluginContext{
					AppInstanceSettings: &tc.settings,
				},
				Method: http.MethodPost,
				Path:   "/openai/v1/chat/completions",
				Body:   []byte(fmt.Sprintf(`{"model": "%s", "messages": [{"content":"some stuff"}]}`, tc.model)),
			}
			err = app.CallResource(ctx, req, &r)
			require.NoError(t, err)
			require.NotNil(t, r.response)

			oReq := &openai.ChatCompletionRequest{}
			err = json.Unmarshal(server.requestBody, oReq)
			require.NoError(t, err)
			require.Equal(t, tc.expectedModel, oReq.Model)
		})
	}
}

func withProviderURL(t *testing.T, s backend.AppInstanceSettings, url string) backend.AppInstanceSettings {
	t.Helper()
	var jd map[string]any
	err := json.Unmarshal(s.JSONData, &jd)
	if err != nil {
		panic(err)
	}
	if jd["provider"] == string(ProviderTypeOpenAI) {
		jd["openAI"] = map[string]any{"url": url}
	} else if jd["provider"] == string(ProviderTypeAnthropic) {
		jd["anthropic"] = map[string]any{"url": url}
	}
	s.JSONData, err = json.Marshal(jd)
	if err != nil {
		panic(err)
	}
	return s
}

package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
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

		method string
		path   string
		body   []byte

		expStatus int
		expBody   []byte
	}{
		{
			name:      "get non existing handler 404",
			method:    http.MethodGet,
			path:      "not_found",
			expStatus: http.StatusNotFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Request by calling CallResource. This tests the httpadapter.
			var r mockCallResourceResponseSender
			err = app.CallResource(ctx, &backend.CallResourceRequest{
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

type mockServer struct {
	server  *httptest.Server
	request *http.Request
}

func newMockOpenAIServer(t *testing.T) *mockServer {
	server := &mockServer{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.request = r
		w.WriteHeader(http.StatusOK)
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
					Provider:       openAIProviderOpenAI,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "gpt-3.5-turbo", "messages": ["some stuff"]}`),

			expReqHeaders: http.Header{
				"Authorization":       {"Bearer abcd1234"},
				"OpenAI-Organization": {"myOrg"},
			},
			expReqPath: "/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "messages": ["some stuff"]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "azure",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       openAIProviderAzure,
					AzureMapping: [][]string{
						{"gpt-3.5-turbo", "gpt-35-turbo"},
					},
				},
			},

			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "gpt-3.5-turbo", "messages": ["some stuff"]}`),

			expReqHeaders: http.Header{
				"api-key": {"abcd1234"},
			},
			expReqPath: "/openai/deployments/gpt-35-turbo/chat/completions",
			// the 'model' field should have been removed.
			expReqBody: []byte(`{"messages":["some stuff"]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "azure invalid deployment",

			settings: Settings{
				OpenAI: OpenAISettings{
					OrganizationID: "myOrg",
					Provider:       openAIProviderAzure,
					AzureMapping: [][]string{
						{"gpt-3.5-turbo", "gpt-35-turbo"},
					},
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			// note no gpt-4 in AzureMapping.
			body: []byte(`{"model": "gpt-4", "messages": ["some stuff"]}`),

			expNilRequest: true,

			expStatus: http.StatusBadRequest,
		},
		{
			name: "grafana-managed llm gateway - opt in not set",

			settings: Settings{
				Tenant: "123",
				OpenAI: OpenAISettings{
					Provider: openAIProviderGrafana,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "gpt-3.5-turbo", "messages": ["some stuff"]}`),

			expReqHeaders: http.Header{
				"Authorization": {"Bearer abcd1234"},
				"X-Scope-OrgID": {"123"},
			},
			expReqPath: "/openai/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "messages": ["some stuff"]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "grafana-managed llm gateway - opt in set to true",

			settings: Settings{
				Tenant: "123",
				OpenAI: OpenAISettings{
					Provider: openAIProviderGrafana,
				},
				LLMGateway: LLMGatewaySettings{
					IsOptIn: true,
				},
			},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "gpt-3.5-turbo", "messages": ["some stuff"]}`),

			expReqHeaders: http.Header{
				"Authorization": {"Bearer abcd1234"},
				"X-Scope-OrgID": {"123"},
			},
			expReqPath: "/openai/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "messages": ["some stuff"]}`),

			expStatus: http.StatusOK,
		},
		{
			name: "grafana-managed llm gateway - opt in set to false",

			settings: Settings{
				Tenant: "123",
				OpenAI: OpenAISettings{
					Provider: openAIProviderGrafana,
				},
				LLMGateway: LLMGatewaySettings{
					IsOptIn: false,
				}},
			apiKey: "abcd1234",

			method: http.MethodPost,
			path:   "/openai/v1/chat/completions",
			body:   []byte(`{"model": "gpt-3.5-turbo", "messages": ["some stuff"]}`),

			expReqHeaders: http.Header{
				"Authorization": {"Bearer abcd1234"},
				"X-Scope-OrgID": {"123"},
			},
			expReqPath: "/openai/v1/chat/completions",
			expReqBody: []byte(`{"model": "gpt-3.5-turbo", "messages": ["some stuff"]}`),

			expStatus: http.StatusOK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			// Start up a mock server that just captures the request and sends a 200 OK response.
			server := newMockOpenAIServer(t)

			// Update the OpenAI/LLMGateway URL with the mock server's URL.
			if tc.settings.OpenAI.Provider == openAIProviderGrafana {
				tc.settings.LLMGateway.URL = server.server.URL
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
				if tb := bytes.TrimSpace(r.response.Body); !bytes.Equal(tb, tc.expBody) {
					t.Errorf("response body should be %s, got %s", tc.expBody, tb)
				}
			}
		})
	}
}

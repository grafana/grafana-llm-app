package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testOpenAIChatCompletionsPath = "openai/v1/chat/completions"
	testLLMChatCompletionsPath    = "llm/v1/chat/completions"
)

type mockStreamServer struct {
	server  *httptest.Server
	request *http.Request
}

func newMockOpenAIStreamServer(t *testing.T, statusCode int, includeDone bool) *mockStreamServer {
	server := &mockStreamServer{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("mock server got request: %s", r.URL.String())
		server.request = r

		if statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
			_, _ = fmt.Fprintf(w, `{"error": {"code": %d}}`, statusCode)
			flusher := w.(http.Flusher)
			flusher.Flush()
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		streamMessages := []byte{}
		for i := 0; i < 10; i++ {
			// Actual body isn't really important here.
			data := fmt.Sprintf(`{"choices":[{"delta":{"content":"response%d"},"finish_reason":null,"index":0,"logprobs":null}],"id":"mock-chat-id","model":"gpt-4-turbo","object":"chat.completion.chunk","p":"p","system_fingerprint":"abc"}`, i)
			dataBytes := []byte("data: " + data + "\n\n")
			streamMessages = append(streamMessages, dataBytes...)
		}

		// final message has finish reason
		streamMessages = append(streamMessages, []byte(`{"choices":[{"delta":{},"finish_reason":"stop","index":0,"logprobs":null}],"created":1714142715,"id":"mock-chat-id","model":"gpt-4-turbo","object":"chat.completion.chunk","p":"ppppppppppp","system_fingerprint":"abc"}}}`)...)

		// done messages
		if includeDone {
			streamMessages = append(streamMessages, []byte("event: done\n")...)
			streamMessages = append(streamMessages, []byte("data: [DONE]\n\n")...)
		}
		_, err := w.Write(streamMessages)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.(http.Flusher).Flush()
	})
	server.server = httptest.NewServer(handler)
	return server
}

type mockStreamPacketSender struct {
	messages []json.RawMessage
}

func (s *mockStreamPacketSender) Send(packet *backend.StreamPacket) error {
	s.messages = append(s.messages, packet.Data)
	return nil
}

func TestRunStream(t *testing.T) {
	body := []byte(`{
		"model": "gpt-3.5-turbo",
		"messages": []
	}`)

	testCases := []struct {
		name        string
		settings    Settings
		statusCode  int
		includeDone bool

		expErr          string
		expMessageCount int
	}{
		{
			name:       "bad auth",
			settings:   Settings{OpenAI: OpenAISettings{Provider: ProviderTypeOpenAI}},
			statusCode: http.StatusUnauthorized,

			expErr:          "401",
			expMessageCount: 0,
		},
		{
			name: "grafana managed key",
			settings: Settings{
				OpenAI: OpenAISettings{Provider: ProviderTypeGrafana},
			},
			statusCode: http.StatusUnauthorized,

			expErr:          "401",
			expMessageCount: 0,
		},
		{
			name:        "happy path",
			settings:    Settings{OpenAI: OpenAISettings{Provider: ProviderTypeOpenAI}},
			statusCode:  http.StatusOK,
			includeDone: true,

			expErr:          "",
			expMessageCount: 11, // 9 messages + 1 finish reason + 1 done
		},
		{
			name:       "happy path without EOF",
			settings:   Settings{OpenAI: OpenAISettings{Provider: ProviderTypeOpenAI}},
			statusCode: http.StatusOK,

			expErr:          "",
			expMessageCount: 11, // 9 messages + 1 finish reason + 1 done
		},
		{
			name:       "deprecated openai path compatibility",
			settings:   Settings{OpenAI: OpenAISettings{Provider: ProviderTypeOpenAI}},
			statusCode: http.StatusOK,

			expErr:          "",
			expMessageCount: 11, // 9 messages + 1 finish reason + 1 done
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			// Start up a mock server that just captures the request and sends a 200 OK response.
			server := newMockOpenAIStreamServer(t, tc.statusCode, tc.includeDone)

			// Initialize app (need to set OpenAISettings:URL in here)
			settings := tc.settings
			if settings.OpenAI.Provider == ProviderTypeGrafana {
				settings.LLMGateway.URL = server.server.URL
			} else {
				settings.OpenAI.URL = server.server.URL
			}

			jsonData, err := json.Marshal(settings)
			if err != nil {
				t.Fatalf("json marshal: %s", err)
			}
			appSettings := backend.AppInstanceSettings{
				JSONData: jsonData,
				DecryptedSecureJSONData: map[string]string{
					"openAIKey": "abcd1234",
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

			r := mockStreamPacketSender{messages: []json.RawMessage{}}
			sender := backend.NewStreamSender(&r)

			err = app.RunStream(ctx, &backend.RunStreamRequest{
				PluginContext: backend.PluginContext{
					AppInstanceSettings: &appSettings,
				},
				Path: func() string {
					if tc.name == "deprecated openai path compatibility" {
						return testOpenAIChatCompletionsPath + "/abcd1234"
					}
					return testLLMChatCompletionsPath + "/abcd1234"
				}(),
				Data: body,
			}, sender)
			log.DefaultLogger.Info("RunStream finished")
			if err != nil {
				t.Fatalf("RunStream error: %s", err)
			}

			n := len(r.messages)
			if tc.expErr != "" {
				var got EventError
				if err = json.Unmarshal(r.messages[n-1], &got); err != nil {
					t.Fatalf("got non-JSON error message %s", r.messages[n-1])
				}
				if !strings.Contains(got.Error, tc.expErr) {
					t.Fatalf("expected error to contain %q, got %q", tc.expErr, got.Error)
				}
				if tc.expMessageCount != n-1 {
					t.Fatalf("expected %d non-error messages, got %d", tc.expMessageCount, n-1)
				}
				return
			}

			// expect the right number of messages
			if tc.expMessageCount != n {
				t.Fatalf("expected %d messages, got %d", tc.expMessageCount, n)
			}
		})
	}
}

type mockGrafanaServer struct {
	server    *httptest.Server
	requests  *[]http.Request
	responses []string
}

func newMockGrafanaServer(responses []string) mockGrafanaServer {
	requests := []http.Request{}
	i := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, *r)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if i < len(responses) {
			_, _ = w.Write([]byte(responses[i]))
			i++
		}
	})
	return mockGrafanaServer{
		server:    httptest.NewServer(handler),
		requests:  &requests,
		responses: responses,
	}
}

func TestRunStreamMCP(t *testing.T) {
	s := newMockGrafanaServer([]string{`[]`})
	defer s.server.Close()

	ctx := context.Background()
	fmt.Println("server URL", s.server.URL)
	ctx = backend.WithGrafanaConfig(ctx, backend.NewGrafanaCfg(map[string]string{
		backend.AppURL:          s.server.URL,
		backend.AppClientSecret: "abcd1234",
	}))

	// Initialize app (need to set OpenAISettings:URL in here)
	settings := Settings{
		OpenAI: OpenAISettings{Provider: ProviderTypeOpenAI},
		MCP:    MCPSettings{Disabled: false},
	}

	jsonData, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("json marshal: %s", err)
	}
	appSettings := backend.AppInstanceSettings{
		JSONData: jsonData,
		DecryptedSecureJSONData: map[string]string{
			"openAIKey": "abcd1234",
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

	r := mockStreamPacketSender{messages: []json.RawMessage{}}
	sender := backend.NewStreamSender(&r)

	path := "mcp/abcd1234"

	go func() {
		err = app.RunStream(ctx, &backend.RunStreamRequest{
			PluginContext: backend.PluginContext{
				AppInstanceSettings: &appSettings,
			},
			Path: path,
			Data: []byte(""),
		}, sender)
		require.NoError(t, err)
	}()
	time.Sleep(time.Millisecond)

	resp, err := app.PublishStream(ctx, &backend.PublishStreamRequest{
		Path: path,
		Data: []byte(`{
      "jsonrpc": "2.0",
      "id": 1,
      "method": "initialize",
      "params": {
        "protocolVersion": "",
        "capabilities": {},
        "clientInfo": {"name": "mcp-golang", "version": "0.1.2"}
      }
    }`),
	})
	require.NoError(t, err)
	require.Equal(t, backend.PublishStreamStatusOK, resp.Status)

	resp, err = app.PublishStream(ctx, &backend.PublishStreamRequest{
		Path: path,
		Data: []byte(`{
      "jsonrpc": "2.0",
      "id": 1,
      "method": "tools/list",
      "params": {
        "request": {
          "method": "search/dashboards",
          "params": {
            "query": "test"
          }
        }
      }
    }`),
	})
	require.NoError(t, err)
	require.Equal(t, backend.PublishStreamStatusOK, resp.Status)

	resp, err = app.PublishStream(ctx, &backend.PublishStreamRequest{
		Path: path,
		Data: []byte(`{
      "jsonrpc": "2.0",
      "id": 1,
      "method": "tools/call",
      "params": {
        "name": "search_dashboards",
        "params": {
          "query": "test"
        }
      }
    }`),
	})
	require.NoError(t, err)
	require.Equal(t, backend.PublishStreamStatusOK, resp.Status)

	app.mcpServer.Close()
	require.Len(t, r.messages, 3)
	require.Len(t, *s.requests, 1)
	require.Equal(t, "/api/search?limit=50&page=1", (*s.requests)[0].URL.String())
	require.Equal(t, `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"dashboards\":[],\"total\":0,\"hasMore\":false}"}]}}`, string(r.messages[2]))
}

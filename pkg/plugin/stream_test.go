package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type mockStreamServer struct {
	server  *httptest.Server
	request *http.Request
}

func newMockOpenAIStreamServer(t *testing.T, statusCode int, finish chan (struct{})) *mockStreamServer {
	server := &mockStreamServer{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("mock server got request: %s", r.URL.String())
		server.request = r

		if statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
			_, _ = w.Write([]byte(fmt.Sprintf("error %d", statusCode)))
			flusher := w.(http.Flusher)
			flusher.Flush()
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		for i := 0; i < 10; i++ {
			// Actual body isn't really important here.
			body := fmt.Sprintf(`data: {"choices": [{"text": "%d"}]}\n`, i)
			_, err := w.Write([]byte(body))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.(http.Flusher).Flush()
		}

		_, _ = w.Write([]byte(`data: [DONE]`))
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
	for _, tc := range []struct {
		name       string
		settings   Settings
		statusCode int

		expErr          string
		expMessageCount int
	}{
		{
			name:       "bad auth",
			settings:   Settings{OpenAI: OpenAISettings{Provider: openAIProviderOpenAI}},
			statusCode: http.StatusUnauthorized,

			expErr:          "401",
			expMessageCount: 0,
		},
		{
			name: "grafana managed key but no opt in raises error",
			settings: Settings{
				OpenAI:         OpenAISettings{Provider: openAIProviderGrafana},
				LLMOptInStatus: false,
			},
			statusCode: http.StatusUnauthorized,

			expErr:          "proxy: stream: error creating request: Grafana Provided LLM access is not permitted. We require explicit Opt-In of the feature to continue",
			expMessageCount: 0,
		},
		{
			name: "grafana managed key with opt in succeeds",
			settings: Settings{
				OpenAI:         OpenAISettings{Provider: openAIProviderGrafana},
				LLMOptInStatus: true,
			},
			statusCode: http.StatusPartialContent,

			expErr:          "206",
			expMessageCount: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			finish := make(chan struct{})
			// Start up a mock server that just captures the request and sends a 200 OK response.
			server := newMockOpenAIStreamServer(t, tc.statusCode, finish)

			// Initialize app (need to set OpenAISettings:URL in here)
			settings := tc.settings
			if settings.OpenAI.Provider == openAIProviderGrafana {
				settings.LLMGatewayURL = server.server.URL
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
				Path: openAIChatCompletionsPath + "/abcd1234",
				Data: body,
			}, sender)
			log.DefaultLogger.Info("RunStream finished")
			if err != nil {
				t.Fatalf("RunStream error: %s", err)
			}

			if tc.expErr != "" {
				n := len(r.messages)
				var got EventError
				if err = json.Unmarshal(r.messages[n-1], &got); err != nil {
					t.Fatalf("got non-JSON error message %s", r.messages[n-1])
				}
				if !strings.HasSuffix(got.Error, tc.expErr) {
					t.Fatalf("expected error to end with %q, got %q", tc.expErr, got.Error)
				}
				if tc.expMessageCount != n-1 {
					t.Fatalf("expected %d non-error messages, got %d", tc.expMessageCount, n-1)
				}
			}
		})
	}
}

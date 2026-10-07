package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	mcpgrafana "github.com/grafana/mcp-grafana/v2"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	initializeMsg  = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	initializedMsg = `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	callProbeMsg   = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"probe","arguments":{}}}`
)

// newTestMCP returns an MCP with no Grafana toolsets and a "probe" tool that
// reports the Grafana config it sees in its context.
func newTestMCP(t *testing.T) *MCP {
	t.Helper()
	m, err := New(Settings{IsToolsetEnabled: func(Toolset) bool { return false }}, "test")
	require.NoError(t, err)
	t.Cleanup(m.Close)
	mcpsdk.AddTool(m.Server, &mcpsdk.Tool{Name: "probe"}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, any, error) {
		cfg := mcpgrafana.GrafanaConfigFromContext(ctx)
		return &mcpsdk.CallToolResult{Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: cfg.URL + "|" + cfg.APIKey}}}, nil, nil
	})
	return m
}

func grafanaCtx() context.Context {
	return config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
		config.AppURL:          "http://grafana.test",
		config.AppClientSecret: "secret",
	}))
}

// requireProbeResult checks a tools/call response for the probe tool.
func requireProbeResult(t *testing.T, body []byte) {
	t.Helper()
	var resp struct {
		Result struct {
			Content []struct{ Text string } `json:"content"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), string(body))
	require.Len(t, resp.Result.Content, 1, string(body))
	assert.Equal(t, "http://grafana.test|secret", resp.Result.Content[0].Text)
}

func TestHTTPServerPassesGrafanaContextToTools(t *testing.T) {
	m := newTestMCP(t)
	post := func(msg string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(grafanaCtx(), http.MethodPost, "/mcp/grafana", strings.NewReader(msg))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		rec := httptest.NewRecorder()
		m.HTTPServer.ServeHTTP(rec, req)
		return rec
	}

	rec := post(initializeMsg)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"serverInfo"`)

	rec = post(callProbeMsg)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	requireProbeResult(t, rec.Body.Bytes())
}

// packetRecorder collects packets sent over a Grafana Live stream.
type packetRecorder struct{ packets chan *backend.StreamPacket }

func (p packetRecorder) Send(pkt *backend.StreamPacket) error {
	p.packets <- pkt
	return nil
}

func TestLiveServerPassesGrafanaContextToTools(t *testing.T) {
	m := newTestMCP(t)
	rec := packetRecorder{packets: make(chan *backend.StreamPacket, 10)}

	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = m.LiveServer.HandleStream(streamCtx, &backend.RunStreamRequest{Path: "mcp/1/subscribe"}, backend.NewStreamSender(rec))
	}()

	publish := func(msg string) *backend.StreamPacket {
		t.Helper()
		var err error
		// Retry until HandleStream has registered the session.
		require.Eventually(t, func() bool {
			err = m.LiveServer.HandleMessage(grafanaCtx(), &backend.PublishStreamRequest{Path: "mcp/1/publish", Data: json.RawMessage(msg)})
			return err != ErrStreamNotFound
		}, time.Second, 10*time.Millisecond)
		require.NoError(t, err)
		return <-rec.packets
	}

	assert.Contains(t, string(publish(initializeMsg).Data), `"serverInfo"`)
	assert.Empty(t, publish(initializedMsg).Data, "notifications get an empty response")
	requireProbeResult(t, publish(callProbeMsg).Data)
}

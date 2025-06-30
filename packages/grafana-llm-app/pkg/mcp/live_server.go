package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/incident-go"
	mcpgrafana "github.com/grafana/mcp-grafana"

	"github.com/go-openapi/strfmt"
	"github.com/mark3labs/mcp-go/server"
)

const (
	// subscribeSuffix is the suffix for the subscribe channel endpoint.
	subscribeSuffix = "/subscribe"
	// publishSuffix is the suffix for the publish channel endpoint.
	publishSuffix = "/publish"

	// accessTokenHeader is the HTTP header key for the access token.
	accessTokenHeader = "X-Access-Token"
)

// ErrStreamNotFound is an error returned when a publish message is sent to a path
// without a corresponding session (i.e. a stream without any subscribers).
var ErrStreamNotFound = errors.New("stream not found")

// GrafanaLiveContextFunc is a function that takes an existing context and returns
// a potentially modified context.
// pCtx is the plugin context for the current request. This will contain
// some user specific information.
type GrafanaLiveContextFunc func(ctx context.Context, pCtx *backend.PluginContext, accessToken, grafanaIdToken string) context.Context

// GrafanaLiveServer wraps an MCPServer and coordinates Grafana Live connections
// to the MCP server.
//
// It is effectively a custom MCP transport, similar to SSE, which:
//
//   - accepts new long-lived connections using the `RunStream` handler, over which
//     the MCP server will send messages
//   - accepts JSON-RPC messages over the `PublishStream` handler, which MCP clients
//     can use to perform standard MCP operations (list tools, call tool, etc.)
type GrafanaLiveServer struct {
	// server is the MCP server that will handle the MCP messages.
	server *server.MCPServer
	// Whether we are running in Grafana Cloud.
	isGrafanaCloud bool
	// accessTokenClient is the client for getting access tokens.
	acc *accessTokenClient
	// sessions is a map of active Grafana Live connections, keyed by the path
	// of the channel with the suffix "/subscribe" or "/publish" removed.
	sessions sync.Map
	// contextFunc is a function that will be called to modify the context before
	// handling each MCP message.
	contextFunc GrafanaLiveContextFunc
	// done is a channel that will be closed when the Grafana Live server is
	// shutting down.
	done chan struct{}
}

// GrafanaLiveOption defines a function type for configuring the GrafanaLiveServer.
type GrafanaLiveOption func(*GrafanaLiveServer)

// WithIsGrafanaCloud returns a GrafanaLiveOption that sets whether the server
// is running in Grafana Cloud environment.
func WithIsGrafanaCloud(enabled bool) GrafanaLiveOption {
	return func(s *GrafanaLiveServer) {
		s.isGrafanaCloud = enabled
	}
}

// NewGrafanaLiveServer creates a new GrafanaLiveServer.
func NewGrafanaLiveServer(server *server.MCPServer, acc *accessTokenClient, opts ...GrafanaLiveOption) *GrafanaLiveServer {
	s := &GrafanaLiveServer{
		server: server,
		acc:    acc,
		done:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.contextFunc = composedGrafanaLiveContextFunc

	return s
}

// SetContextFunc sets the context function for the GrafanaLiveServer.
func (s *GrafanaLiveServer) SetContextFunc(contextFunc GrafanaLiveContextFunc) {
	s.contextFunc = contextFunc
}

// Close closes the GrafanaLiveServer.
func (s *GrafanaLiveServer) Close() {
	close(s.done)
}

// liveSession represents an active Grafana Live session for MCP communication.
type liveSession struct {
	// sender is the StreamSender for the Grafana Live session. It is used to send
	// JSON-RPC responses back to the client.
	sender *backend.StreamSender
}

// HandleStream handles a new Grafana Live session for MCP communication.
// It creates a session, stores it in the sessions map, and blocks until the stream
// is closed or the server is shutting down.
func (s *GrafanaLiveServer) HandleStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	ls := &liveSession{
		sender: sender,
	}

	// Store the session in the sessions map.
	path := strings.TrimSuffix(req.Path, subscribeSuffix)
	s.sessions.Store(path, ls)
	defer s.sessions.Delete(path)
	// Block until the stream is closed or the Grafana Live server is shutting down.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.done:
			return nil
		}
	}
}

// HandleMessage handles a Grafana Live message sent via the PublishStream handler.
// It processes MCP JSON-RPC messages from clients and sends responses back through
// the corresponding Live session.
func (s *GrafanaLiveServer) HandleMessage(ctx context.Context, req *backend.PublishStreamRequest) error {
	path := strings.TrimSuffix(req.Path, publishSuffix)
	// Get the session from the sessions map.
	sessionI, ok := s.sessions.Load(path)
	if !ok {
		return ErrStreamNotFound
	}
	session := sessionI.(*liveSession)

	accessToken, err := s.acc.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	grafanaIdToken := req.GetHTTPHeader(backend.GrafanaUserSignInTokenHeaderName)
	if s.isGrafanaCloud && grafanaIdToken == "" {
		return fmt.Errorf("grafana id token not found in request headers")
	}

	// Modify the context if a context function is set.
	if s.contextFunc != nil {
		ctx = s.contextFunc(ctx, &req.PluginContext, accessToken, grafanaIdToken)
	}

	log.DefaultLogger.Info("Handling message", "len_access_token", len(accessToken), "len_grafana_id_token", len(grafanaIdToken))

	// Process the message through the MCPServer.
	response := s.server.HandleMessage(ctx, req.Data)

	// Only send response if there is one (not for notifications).
	if response != nil {
		// Marshal the response to JSON. Errors should be impossible since we've
		// just unmarshalled from a JSON-RPC message.
		eventData, _ := json.Marshal(response)
		return session.sender.SendJSON(eventData)
	} else {
		// For notifications, just send nil.
		return session.sender.SendBytes(nil)
	}
}

// extractGrafanaInfoFromGrafanaLiveRequest extracts Grafana configuration from settings
// and adds Grafana info to the context using mcp-grafana helpers. It handles authentication
// with the following priority:
//
// 1. If we have an access token and Grafana ID token, use on-behalf-of auth.
// 2. If we are not using Grafana Cloud, use the API key.
//
// If we can't get an access token (e.g. if token exchange fails), no Grafana
// info is added to the context.
func extractGrafanaInfoFromGrafanaLiveRequest(ctx context.Context, pCtx *backend.PluginContext, accessToken, grafanaIdToken string) context.Context {
	cfg := backend.GrafanaConfigFromContext(ctx)
	if cfg == nil {
		return ctx
	}
	url, err := cfg.AppURL()
	if err != nil {
		return ctx
	}

	gCfg := mcpgrafana.GrafanaConfigFromContext(ctx)
	gCfg.URL = url

	// If we have an access token and grafana id token, use on-behalf-of auth.
	if accessToken != "" && grafanaIdToken != "" {
		// MustWithOnBehalfOfAuth will panic if the access token or grafana id token
		// are empty. That is why we check for empty strings above.
		return mcpgrafana.MustWithOnBehalfOfAuth(mcpgrafana.WithGrafanaConfig(ctx, gCfg), accessToken, grafanaIdToken)
	}

	// If we are not using Grafana Cloud, use the API key.
	gCfg.APIKey, _ = cfg.PluginAppClientSecret()
	return mcpgrafana.WithGrafanaConfig(ctx, gCfg)
}

// extractGrafanaClientFromGrafanaLiveRequest extracts Grafana configuration from settings
// and creates a Grafana API client in the context. It configures the client with the
// appropriate authentication method based on available tokens.
func extractGrafanaClientFromGrafanaLiveRequest(ctx context.Context, pCtx *backend.PluginContext, accessToken, grafanaIdToken string) context.Context {
	t := client.DefaultTransportConfig()

	cfg := backend.GrafanaConfigFromContext(ctx)
	if cfg == nil {
		return ctx
	}
	urlS, err := cfg.AppURL()
	if err != nil {
		return ctx
	}
	url, err := url.Parse(urlS)
	if err != nil {
		return ctx
	}
	if url.Host != "" {
		t.Host = url.Host
	}
	// The Grafana client will always prefer HTTPS even if the URL is HTTP,
	// so we need to limit the schemes to HTTP if the URL is HTTP.
	if url.Scheme == "http" {
		t.Schemes = []string{"http"}
	}

	// If we have an access token, set it in the HTTP headers.
	if len(accessToken) > 0 {
		log.DefaultLogger.Info("Setting access token in grafana client", "len_access_token", len(accessToken))
		t.HTTPHeaders = map[string]string{
			accessTokenHeader:                        accessToken,
			backend.GrafanaUserSignInTokenHeaderName: grafanaIdToken,
		}
	} else {
		if apiKey, err := cfg.PluginAppClientSecret(); err == nil {
			t.APIKey = apiKey
		}
	}

	c := client.NewHTTPClientWithConfig(strfmt.Default, t)
	return mcpgrafana.WithGrafanaClient(ctx, c)
}

// extractIncidentClientFromGrafanaLiveRequest creates an Incident client and adds it to the context.
// Note: The incident client does not support access tokens, so it uses API key authentication only.
func extractIncidentClientFromGrafanaLiveRequest(ctx context.Context, pCtx *backend.PluginContext, accessToken, grafanaIdToken string) context.Context {
	cfg := backend.GrafanaConfigFromContext(ctx)
	if cfg == nil {
		return ctx
	}
	grafanaURL, err := cfg.AppURL()
	if err != nil {
		return ctx
	}
	apiKey, _ := cfg.PluginAppClientSecret()
	incidentUrl := fmt.Sprintf("%s/api/plugins/grafana-incident-app/resources/api/", strings.TrimSuffix(grafanaURL, "/"))
	// TODO: incident client does not support access tokens. For this reason,
	// we will not be enabling Incident tools in Grafana Cloud yet.
	client := incident.NewClient(incidentUrl, apiKey)
	return mcpgrafana.WithIncidentClient(ctx, client)
}

// composeGrafanaLiveContextFuncs composes multiple GrafanaLiveContextFunc functions into a single one.
// The composed function applies each function in order to the context.
func composeGrafanaLiveContextFuncs(funcs ...GrafanaLiveContextFunc) GrafanaLiveContextFunc {
	return func(ctx context.Context, pCtx *backend.PluginContext, accessToken, grafanaIdToken string) context.Context {
		for _, f := range funcs {
			ctx = f(ctx, pCtx, accessToken, grafanaIdToken)
		}
		return ctx
	}
}

// composedGrafanaLiveContextFunc is a GrafanaLiveContextFunc that calls all the context
// extraction functions in order to set up the complete context for MCP requests.
var composedGrafanaLiveContextFunc = composeGrafanaLiveContextFuncs(
	extractGrafanaInfoFromGrafanaLiveRequest,
	extractGrafanaClientFromGrafanaLiveRequest,
	extractIncidentClientFromGrafanaLiveRequest,
)

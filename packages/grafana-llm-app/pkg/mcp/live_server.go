package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/grafana/authlib/authn"
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
type GrafanaLiveContextFunc func(ctx context.Context, pCtx *backend.PluginContext, accessToken string, grafanaIdToken string) context.Context

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
	// tenant is the stack ID (Hosted Grafana ID) of the instance this plugin
	// is running on.
	tenant string
	// llmAppAccessPolicyToken is the token created from the LLM app's access policy.
	llmAppAccessPolicyToken string
	// Note: Even though this flag is named enableGrafanaManagedLLM and was originally added as
	// a setting to choose the Grafana Managed LLM, we are using it as a proxy to check if we are
	// running in Grafana Cloud. This needs to be renamed in the future but it requires a larger
	// refactor.
	enableGrafanaManagedLLM bool
	// tokenExchangeClient is the token exchange client for the Grafana Live session.
	tokenExchangeClient *authn.TokenExchangeClient
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

// WithGrafanaLiveContextFunc sets the context function for the GrafanaLiveServer.
func WithGrafanaLiveContextFunc(contextFunc GrafanaLiveContextFunc) GrafanaLiveOption {
	return func(s *GrafanaLiveServer) {
		s.contextFunc = contextFunc
	}
}

func WithGrafanaTenant(tenant string) GrafanaLiveOption {
	return func(s *GrafanaLiveServer) {
		s.tenant = tenant
	}
}

func WithGrafanaManagedLLM(enabled bool) GrafanaLiveOption {
	return func(s *GrafanaLiveServer) {
		s.enableGrafanaManagedLLM = enabled
	}
}

func WithLLMAppAccessPolicyToken(token string) GrafanaLiveOption {
	return func(s *GrafanaLiveServer) {
		s.llmAppAccessPolicyToken = token
	}
}

// NewGrafanaLiveServer creates a new GrafanaLiveServer.
func NewGrafanaLiveServer(server *server.MCPServer, opts ...GrafanaLiveOption) (*GrafanaLiveServer, error) {
	s := &GrafanaLiveServer{
		server: server,
		done:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}

	if s.enableGrafanaManagedLLM {
		var err error
		s.tokenExchangeClient, err = authn.NewTokenExchangeClient(authn.TokenExchangeConfig{
			Token:            s.llmAppAccessPolicyToken,
			TokenExchangeURL: "http://api-lb.auth.svc.cluster.local./v1/sign-access-token", // TODO: make this configurable.
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create token exchange client: %w", err)
		}
	}
	return s, nil
}

// SetContextFunc sets the context function for the GrafanaLiveServer.
func (s *GrafanaLiveServer) SetContextFunc(contextFunc GrafanaLiveContextFunc) {
	s.contextFunc = contextFunc
}

// Close closes the GrafanaLiveServer.
func (s *GrafanaLiveServer) Close() {
	close(s.done)
}

// liveSession is a Grafana Live session.
type liveSession struct {
	// sender is the StreamSender for the Grafana Live session. It is used to send
	// JSON-RPC responses back to the client.
	sender *backend.StreamSender

	// mu protects concurrent access to the tokens.
	mu sync.RWMutex
	// accessToken is the access token for the Grafana Live session.
	accessToken string
}

// tokenTimeoutSeconds is the expiration time for the Grafana Live session token.
var tokenTimeout = time.Minute * 30

// tokenRefreshInterval is the time between token refreshes.
var tokenRefreshInterval = tokenTimeout - time.Minute

// exchangeToken uses the token exchange API to create an access token for the Grafana Live session.
func (s *GrafanaLiveServer) exchangeToken(ctx context.Context) (*authn.TokenExchangeResponse, error) {
	tokenTimeoutSeconds := int(tokenTimeout.Seconds())
	tr, err := s.tokenExchangeClient.Exchange(ctx, authn.TokenExchangeRequest{
		Namespace: fmt.Sprintf("stack-%s", s.tenant),
		Audiences: []string{"grafana"},
		ExpiresIn: &tokenTimeoutSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to perform token exchange with auth api: %w", err)
	}
	return tr, nil
}

// HandleStream handles a new Grafana Live session.
func (s *GrafanaLiveServer) HandleStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	ls := &liveSession{
		sender: sender,
	}

	// This timer is only used if we're in Grafana Cloud, but we need to declare it here
	// so that it can be used in the select statement below without it panicking (due to
	// the channel being nil).
	// Outside of Grafana Cloud it will just tick occasionally and be a no-op.
	// The timer will fire one minute before the token is due to expire.
	t := time.NewTimer(tokenRefreshInterval)
	defer t.Stop()
	// Only do the token exchange if we are in Grafana Cloud.
	if s.enableGrafanaManagedLLM {
		tokenExchangeResponse, err := s.exchangeToken(ctx)
		if err != nil {
			return err
		}
		ls.mu.Lock()
		ls.accessToken = tokenExchangeResponse.Token
		ls.mu.Unlock()
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
		case <-t.C:
			// Only refresh the token if we're in Grafana Cloud.
			if !s.enableGrafanaManagedLLM {
				continue
			}
			tr, err := s.exchangeToken(ctx)
			if err != nil {
				log.DefaultLogger.Error("failed to refresh token", "error", err)
				return err
			}
			ls.mu.Lock()
			ls.accessToken = tr.Token
			ls.mu.Unlock()
			// Reset the timer so that it fires again one minute before the token is due to be expire.
			t.Reset(tokenRefreshInterval)
		case <-s.done:
			return nil
		}
	}
}

// HandleMessage handles a Grafana Live message, sent via the PublishStream handler.
func (s *GrafanaLiveServer) HandleMessage(ctx context.Context, req *backend.PublishStreamRequest) error {
	path := strings.TrimSuffix(req.Path, publishSuffix)
	// Get the session from the sessions map.
	sessionI, ok := s.sessions.Load(path)
	if !ok {
		return ErrStreamNotFound
	}
	session := sessionI.(*liveSession)

	// Get the tokens with read lock protection.
	session.mu.RLock()
	accessToken := session.accessToken
	session.mu.RUnlock()
	grafanaIdToken := req.GetHTTPHeader(backend.GrafanaUserSignInTokenHeaderName)
	if s.enableGrafanaManagedLLM && grafanaIdToken == "" {
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

func extractGrafanaInfoFromGrafanaLiveRequest(ctx context.Context, pCtx *backend.PluginContext, accessToken string, grafanaIdToken string) context.Context {
	cfg := backend.GrafanaConfigFromContext(ctx)
	if cfg == nil {
		return ctx
	}
	url, err := cfg.AppURL()
	if err != nil {
		return ctx
	}

	// If we have an access token and grafana id token, use on-behalf-of auth.
	if accessToken != "" && grafanaIdToken != "" {
		// MustWithOnBehalfOfAuth will panic if the access token or grafana id token
		// are empty. That is why we check for empty strings above.
		return mcpgrafana.MustWithOnBehalfOfAuth(mcpgrafana.WithGrafanaURL(ctx, url), accessToken, grafanaIdToken)
	}

	// If we are not using Grafana Cloud, use the API key.
	apiKey, _ := cfg.PluginAppClientSecret()
	return mcpgrafana.WithGrafanaAPIKey(mcpgrafana.WithGrafanaURL(ctx, url), apiKey)
}

// ExtractClientFromGrafanaLiveRequest is a GrafanaLiveContextFunc which extracts the Grafana config
// from settings and sets the client in the context.
func extractGrafanaClientFromGrafanaLiveRequest(ctx context.Context, pCtx *backend.PluginContext, accessToken string, grafanaIdToken string) context.Context {
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

func extractIncidentClientFromGrafanaLiveRequest(ctx context.Context, pCtx *backend.PluginContext, accessToken string, grafanaIdToken string) context.Context {
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

// ComposeStdioContextFuncs composes multiple GrafanaLiveContextFunc into a single one.
func composeStdioContextFuncs(funcs ...GrafanaLiveContextFunc) GrafanaLiveContextFunc {
	return func(ctx context.Context, pCtx *backend.PluginContext, accessToken string, grafanaIdToken string) context.Context {
		for _, f := range funcs {
			ctx = f(ctx, pCtx, accessToken, grafanaIdToken)
		}
		return ctx
	}
}

var ContextFunc = composeStdioContextFuncs(
	extractGrafanaInfoFromGrafanaLiveRequest,
	extractGrafanaClientFromGrafanaLiveRequest,
	extractIncidentClientFromGrafanaLiveRequest,
)

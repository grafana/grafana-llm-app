package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sync"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/mark3labs/mcp-go/server"
)

var ErrStreamNotFound = errors.New("stream not found")

// StdioContextFunc is a function that takes an existing context and returns
// a potentially modified context.
// This can be used to inject context values from environment variables,
// for example.
type GrafanaLiveContextFunc func(ctx context.Context, pCtx *backend.PluginContext) context.Context

// GrafanaLive wraps a MCPServer and handles Grafana Live communication.
type GrafanaLiveServer struct {
	server      *server.MCPServer
	sessions    sync.Map
	contextFunc GrafanaLiveContextFunc
	done        chan struct{}
}

func NewGrafanaLiveServer(server *server.MCPServer) GrafanaLiveServer {
	return GrafanaLiveServer{
		server: server,
		done:   make(chan struct{}),
	}
}

func (s *GrafanaLiveServer) SetContextFunc(contextFunc GrafanaLiveContextFunc) {
	s.contextFunc = contextFunc
}

func (s *GrafanaLiveServer) Close() {
	close(s.done)
}

type liveSession struct {
	sender *backend.StreamSender
}

func (s *GrafanaLiveServer) HandleStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	s.sessions.Store(req.Path, &liveSession{
		sender: sender,
	})
	for {
		select {
		case <-ctx.Done():
			s.sessions.Delete(req.Path)
			return ctx.Err()
		case <-s.done:
			s.sessions.Delete(req.Path)
			return nil
		}
	}
}

func (s *GrafanaLiveServer) HandleMessage(ctx context.Context, req *backend.PublishStreamRequest) error {
	sessionI, ok := s.sessions.Load(req.Path)
	if !ok {
		return ErrStreamNotFound
	}
	session := sessionI.(*liveSession)

	if s.contextFunc != nil {
		ctx = s.contextFunc(ctx, &req.PluginContext)
	}

	// Process message through MCPServer
	response := s.server.HandleMessage(ctx, req.Data)

	// Only send response if there is one (not for notifications)
	if response != nil {
		eventData, _ := json.Marshal(response)
		return session.sender.SendJSON(eventData)
	} else {
		// For notifications, just send nil
		// TODO: is this the right thing?
		return session.sender.SendBytes(nil)
	}
}

type clientKey struct{}

var ExtractClientFromGrafanaLiveRequest GrafanaLiveContextFunc = func(ctx context.Context, pCtx *backend.PluginContext) context.Context {
	t := client.DefaultTransportConfig()

	cfg := backend.GrafanaConfigFromContext(ctx)
	if cfg == nil {
		return ctx
	}
	url, err := cfg.AppURL()
	if err != nil {
		return ctx
	}
	if host := hostFromURL(url); host != "" {
		t.Host = host
	}

	// Hmm, we have the user here, but no ID token? How do we make requests as the user?
	// req.PluginContext.User
	// This will authenticate as the plugin not the current user, which isn't what we want,
	// as the plugin may have more permissions than the user.
	if apiKey, err := cfg.PluginAppClientSecret(); err == nil {
		t.APIKey = apiKey
	}

	return ctx
}

func GrafanaClientFromContext(ctx context.Context) *client.GrafanaHTTPAPI {
	c, ok := ctx.Value(clientKey{}).(*client.GrafanaHTTPAPI)
	if !ok {
		return nil
	}
	return c
}

func hostFromURL(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	host := u.Host
	if host == "" {
		host = u.Path
	}
	return host
}

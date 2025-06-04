package mcp

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

// composeHTTPContextFuncs composes multiple server.HTTPContextFuncs into a single one.
func composeHTTPContextFuncs(funcs ...server.HTTPContextFunc) server.HTTPContextFunc {
	return func(ctx context.Context, req *http.Request) context.Context {
		for _, f := range funcs {
			ctx = f(ctx, req)
		}
		return ctx
	}
}

func extractGrafanaInfoFromHTTPRequest(ctx context.Context, req *http.Request) context.Context {
	// TODO: get ID token, use app's access token to exchange for auth, etc, similar to
	// Grafana Live impl.
	// grafanaIdToken = req.Header.Get(backend.GrafanaUserSignInTokenHeaderName)
	return ctx
}

func extractGrafanaClientFromHTTPRequest(ctx context.Context, req *http.Request) context.Context {
	return ctx
}

func extractIncidentClientFromHTTPRequest(ctx context.Context, req *http.Request) context.Context {
	return ctx
}

var HTTPContextFunc = composeHTTPContextFuncs(
	extractGrafanaInfoFromHTTPRequest,
	extractGrafanaClientFromHTTPRequest,
	extractIncidentClientFromHTTPRequest,
)

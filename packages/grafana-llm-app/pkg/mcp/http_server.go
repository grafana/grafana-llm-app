package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/incident-go"
	mcpgrafana "github.com/grafana/mcp-grafana"
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

// extractGrafanaInfoFromHTTPRequest extracts Grafana configuration from settings
// and adds Grafana info to the context using mcp-grafana helpers. It handles authentication
// with the following priority:
//
// 1. If we have an access token and Grafana ID token, use on-behalf-of auth.
// 2. If we are not using Grafana Cloud, use the API key.
//
// If we can't get an access token (e.g. if token exchange fails), no Grafana
// info is added to the context.
func (m *MCP) extractGrafanaInfoFromHTTPRequest(ctx context.Context, req *http.Request) context.Context {
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

	accessToken, err := m.accessTokenClient.getAccessToken(ctx)
	if err != nil {
		return ctx
	}
	grafanaIDToken := req.Header.Get(backend.GrafanaUserSignInTokenHeaderName)

	// If we have an access token and grafana id token, use on-behalf-of auth.
	if accessToken != "" && grafanaIDToken != "" {
		// MustWithOnBehalfOfAuth will panic if the access token or grafana id token
		// are empty. That is why we check for empty strings above.
		return mcpgrafana.MustWithOnBehalfOfAuth(mcpgrafana.WithGrafanaConfig(ctx, gCfg), accessToken, grafanaIDToken)
	}

	// If we are not using Grafana Cloud, use the API key.
	gCfg.APIKey, _ = cfg.PluginAppClientSecret()
	return mcpgrafana.WithGrafanaConfig(ctx, gCfg)
}

// extractGrafanaClientFromHTTPRequest extracts Grafana configuration from settings
// and creates a Grafana API client in the context. It configures the client with the
// appropriate authentication method based on available tokens.
func (m *MCP) extractGrafanaClientFromHTTPRequest(ctx context.Context, req *http.Request) context.Context {
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

	gCfg := mcpgrafana.GrafanaConfigFromContext(ctx)

	// If we have an access token, set it in the HTTP headers.
	if len(gCfg.AccessToken) > 0 {
		log.DefaultLogger.Info("Setting access token in grafana client", "len_access_token", len(gCfg.AccessToken))
		t.HTTPHeaders = map[string]string{
			accessTokenHeader:                        gCfg.AccessToken,
			backend.GrafanaUserSignInTokenHeaderName: gCfg.IDToken,
		}
	} else {
		if apiKey, err := cfg.PluginAppClientSecret(); err == nil {
			t.APIKey = apiKey
		}
	}

	c := client.NewHTTPClientWithConfig(strfmt.Default, t)
	return mcpgrafana.WithGrafanaClient(ctx, c)
}

// extractIncidentClientFromHTTPRequest creates an Incident client and adds it to the context.
// Note: The incident client does not support access tokens, so it uses API key authentication only.
func (m *MCP) extractIncidentClientFromHTTPRequest(ctx context.Context, req *http.Request) context.Context {
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

// httpContextFunc returns a function that can be used to extract
// information from the HTTP request.
// It is a method of the MCP struct, because it needs access to extra state than is
// allowed by the server.HTTPContextFunc signature (crucially the access token client).
func (m *MCP) httpContextFunc() server.HTTPContextFunc {
	return composeHTTPContextFuncs(
		m.extractGrafanaInfoFromHTTPRequest,
		m.extractGrafanaClientFromHTTPRequest,
		m.extractIncidentClientFromHTTPRequest,
	)
}

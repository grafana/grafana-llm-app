package mcp

import (
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/mcp-grafana/v2/tools"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Toolset identifies an MCP tool family that can be enabled or disabled
type Toolset string

const (
	ToolsetSearch      Toolset = "search"
	ToolsetDatasource  Toolset = "datasource"
	ToolsetIncident    Toolset = "incident"
	ToolsetPrometheus  Toolset = "prometheus"
	ToolsetLoki        Toolset = "loki"
	ToolsetAlerting    Toolset = "alerting"
	ToolsetDashboard   Toolset = "dashboard"
	ToolsetOnCall      Toolset = "oncall"
	ToolsetAsserts     Toolset = "asserts"
	ToolsetPyroscope   Toolset = "pyroscope"
	ToolsetNavigation  Toolset = "navigation"
	ToolsetAnnotations Toolset = "annotations"
	ToolsetRendering   Toolset = "rendering"
	ToolsetAdmin       Toolset = "admin"
	// ToolsetClickHouse enables the SQL datasource tools, which cover ClickHouse
	// along with other SQL datasources.
	ToolsetClickHouse    Toolset = "clickhouse"
	ToolsetCloudWatch    Toolset = "cloudwatch"
	ToolsetElasticsearch Toolset = "elasticsearch"
	ToolsetExamples      Toolset = "examples"
	ToolsetFolder        Toolset = "folder"
)

// Settings contains configuration required by the MCP servers.
type Settings struct {
	// AccessToken is a Grafana Cloud access policy token that is exchanged with
	// one that can be used to authenticate with Grafana, if we're using on-behalf-of
	// auth.
	AccessToken string

	// ServiceAccountToken is the token provided by Grafana to the plugin,
	// and is used to authenticate with Grafana if we're not using on-behalf-of
	// auth.
	ServiceAccountToken string

	// Tenant is the Grafana Cloud tenant ID.
	Tenant string

	// IsGrafanaCloud indicates whether this is running in Grafana Cloud environment.
	IsGrafanaCloud bool

	// If nil, all toolsets are enabled.
	IsToolsetEnabled func(toolset Toolset) bool
}

func (s Settings) isToolsetEnabled(toolset Toolset) bool {
	if s.IsToolsetEnabled == nil {
		return true
	}
	return s.IsToolsetEnabled(toolset)
}

// MCP represents the complete MCP (Model Context Protocol) infrastructure for Grafana.
// It manages both the core MCP server and the Grafana Live server for handling
// real-time communication with MCP clients.
type MCP struct {
	// Server is the core MCP server that handles tool registration and execution.
	Server *mcpsdk.Server
	// LiveServer handles Grafana Live connections for MCP communication.
	LiveServer *GrafanaLiveServer
	// HTTPServer is the MCP Streamable HTTP server for handling MCP requests over HTTP
	// via plugin resource endpoints.
	HTTPServer http.Handler
	// Settings contains the configuration for the MCP servers.
	Settings Settings

	// accessTokenClient is the client for exchanging access policy tokens.
	// This is stored here because it may be shared by different Transports in the future.
	accessTokenClient *accessTokenClient
}

// New creates a new MCP instance with the provided settings and plugin version.
// It initializes the MCP server with all Grafana tools and sets up the Live server
// for handling real-time MCP communication.
func New(settings Settings, pluginVersion string) (*MCP, error) {
	log.DefaultLogger.Debug("Initializing MCP server")
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "grafana-llm-app", Version: pluginVersion}, nil)
	if settings.isToolsetEnabled(ToolsetSearch) {
		tools.AddSearchTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetDatasource) {
		tools.AddDatasourceTools(srv, true)
	}
	// Incident and asserts toolsets require Grafana Cloud.
	if settings.IsGrafanaCloud && settings.isToolsetEnabled(ToolsetIncident) {
		tools.AddIncidentTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetPrometheus) {
		tools.AddPrometheusTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetLoki) {
		tools.AddLokiTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetAlerting) {
		tools.AddAlertingTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetDashboard) {
		tools.AddDashboardTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetOnCall) {
		tools.AddOnCallTools(srv, true)
	}
	if settings.IsGrafanaCloud && settings.isToolsetEnabled(ToolsetAsserts) {
		tools.AddAssertsTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetPyroscope) {
		tools.AddPyroscopeTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetNavigation) {
		tools.AddNavigationTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetAnnotations) {
		tools.AddAnnotationTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetRendering) {
		tools.AddRenderingTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetAdmin) {
		tools.AddAdminTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetClickHouse) {
		tools.AddSQLTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetCloudWatch) {
		tools.AddCloudWatchTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetElasticsearch) {
		tools.AddElasticsearchTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetExamples) {
		tools.AddExamplesTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetFolder) {
		tools.AddFolderTools(srv, true)
	}

	acc, err := newAccessTokenClient(settings.AccessToken, settings.Tenant, settings.IsGrafanaCloud)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token client: %w", err)
	}

	// handler serves MCP requests statelessly with plain JSON responses. Both
	// transports go through it: plugin resource requests (HTTPServer) and
	// Grafana Live messages (LiveServer).
	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server { return srv }, &mcpsdk.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
		Logger:       NewSlogLogger(),
		// Requests arrive via Grafana's plugin resource and Live APIs, never
		// from a network listener, so DNS rebinding protection does not apply.
		DisableLocalhostProtection: true,
	})
	m := &MCP{
		Server:            srv,
		LiveServer:        NewGrafanaLiveServer(handler, acc, WithIsGrafanaCloud(settings.IsGrafanaCloud)),
		Settings:          settings,
		accessTokenClient: acc,
	}
	// The SDK derives tool handler contexts from the HTTP request's context,
	// so Grafana info and clients are added to the request context up front.
	contextFunc := m.httpContextFunc()
	m.HTTPServer = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The SDK rejects requests without the Accept header the spec requires,
		// but mcp-go accepted them, so keep existing clients working.
		if r.Header.Get("Accept") == "" {
			r.Header.Set("Accept", "application/json, text/event-stream")
		}
		handler.ServeHTTP(w, r.WithContext(contextFunc(r.Context(), r)))
	})
	return m, nil
}

// Close shuts down the MCP instance, closing the Live server and cleaning up resources.
func (m *MCP) Close() {
	m.LiveServer.Close()
}

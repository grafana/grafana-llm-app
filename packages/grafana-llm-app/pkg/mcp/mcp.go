package mcp

import (
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/mcp-grafana/v2/resources"
	"github.com/grafana/mcp-grafana/v2/tools"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Toolset identifies an MCP tool family that can be enabled or disabled
type Toolset string

const (
	ToolsetSearch        Toolset = "search"
	ToolsetDatasource    Toolset = "datasource"
	ToolsetIncident      Toolset = "incident"
	ToolsetPrometheus    Toolset = "prometheus"
	ToolsetLoki          Toolset = "loki"
	ToolsetElasticsearch Toolset = "elasticsearch"
	ToolsetQuickwit      Toolset = "quickwit"
	ToolsetInfluxDB      Toolset = "influxdb"
	ToolsetAlerting      Toolset = "alerting"
	ToolsetDashboard     Toolset = "dashboard"
	ToolsetFolder        Toolset = "folder"
	ToolsetOnCall        Toolset = "oncall"
	ToolsetAsserts       Toolset = "asserts"
	ToolsetAdmin         Toolset = "admin"
	ToolsetPyroscope     Toolset = "pyroscope"
	ToolsetNavigation    Toolset = "navigation"
	ToolsetTempo         Toolset = "tempo"
	ToolsetAnnotations   Toolset = "annotations"
	ToolsetRendering     Toolset = "rendering"
	ToolsetSnapshot      Toolset = "snapshot"
	ToolsetCloudWatch    Toolset = "cloudwatch"
	ToolsetCloudLogging  Toolset = "cloudlogging"
	ToolsetExamples      Toolset = "examples"
	// ToolsetSQL enables the SQL datasource tools, which cover ClickHouse,
	// Snowflake, Athena and BigQuery.
	ToolsetSQL           Toolset = "sql"
	ToolsetRunPanelQuery Toolset = "runpanelquery"
	ToolsetGraphite      Toolset = "graphite"
	ToolsetPlugin        Toolset = "plugin"
	ToolsetAPI           Toolset = "api"
	ToolsetConfig        Toolset = "config"
	ToolsetProvisioning  Toolset = "provisioning"
	ToolsetAgento11y     Toolset = "agento11y"
	ToolsetAssistant     Toolset = "assistant"
	ToolsetDocs          Toolset = "docs"
	ToolsetUser          Toolset = "user"

	// ToolsetClickHouse is an alias for ToolsetSQL, kept because the setting
	// used to be called "clickhouse".
	//
	// Deprecated: use ToolsetSQL.
	ToolsetClickHouse = ToolsetSQL
)

// AllToolsets returns every toolset this package can register.
func AllToolsets() []Toolset {
	return []Toolset{
		ToolsetSearch, ToolsetDatasource, ToolsetIncident, ToolsetPrometheus,
		ToolsetLoki, ToolsetElasticsearch, ToolsetQuickwit, ToolsetInfluxDB,
		ToolsetAlerting, ToolsetDashboard, ToolsetFolder, ToolsetOnCall,
		ToolsetAsserts, ToolsetAdmin, ToolsetPyroscope, ToolsetNavigation,
		ToolsetTempo, ToolsetAnnotations, ToolsetRendering, ToolsetSnapshot,
		ToolsetCloudWatch, ToolsetCloudLogging, ToolsetExamples, ToolsetSQL,
		ToolsetRunPanelQuery, ToolsetGraphite, ToolsetPlugin, ToolsetAPI,
		ToolsetConfig, ToolsetProvisioning, ToolsetAgento11y, ToolsetAssistant,
		ToolsetDocs, ToolsetUser,
	}
}

// defaultDisabledToolsets are the toolsets that stay off when their setting is
// unset. They either grant wide privileges (the plugin install and arbitrary
// HTTP request tools), need datasources most stacks do not run (InfluxDB,
// Quickwit, Cloud Logging, Graphite), or only matter to interactive Grafana
// Apps (Run Panel Query, Agent O11y, Assistant). Turning them on is one
// setting away.
var defaultDisabledToolsets = map[Toolset]bool{
	ToolsetQuickwit:      true,
	ToolsetInfluxDB:      true,
	ToolsetCloudLogging:  true,
	ToolsetRunPanelQuery: true,
	ToolsetGraphite:      true,
	ToolsetPlugin:        true,
	ToolsetAPI:           true,
	ToolsetAgento11y:     true,
	ToolsetAssistant:     true,
}

// IsDefaultEnabled reports whether a toolset is registered when it has no
// explicit setting. The settings layer uses it to decide the value it passes
// to Settings.IsToolsetEnabled.
func IsDefaultEnabled(toolset Toolset) bool {
	return !defaultDisabledToolsets[toolset]
}

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
	// The plugin has no read-only mode: every write-capable toolset registers
	// its write tools. AddAlertingTools takes a per-tool predicate instead of a
	// single flag, so answer yes for every tool name.
	writeEnabled := func(string) bool { return true }
	if settings.isToolsetEnabled(ToolsetSearch) {
		tools.AddSearchTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetDatasource) {
		tools.AddDatasourceTools(srv, true)
		resources.AddDatasources(srv)
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
	if settings.isToolsetEnabled(ToolsetElasticsearch) {
		tools.AddElasticsearchTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetQuickwit) {
		tools.AddQuickwitTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetInfluxDB) {
		tools.AddInfluxDBTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetAlerting) {
		tools.AddAlertingTools(srv, writeEnabled)
	}
	if settings.isToolsetEnabled(ToolsetDashboard) {
		tools.AddDashboardTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetFolder) {
		tools.AddFolderTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetOnCall) {
		tools.AddOnCallTools(srv, true)
	}
	if settings.IsGrafanaCloud && settings.isToolsetEnabled(ToolsetAsserts) {
		tools.AddAssertsTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetAdmin) {
		tools.AddAdminTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetPyroscope) {
		tools.AddPyroscopeTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetNavigation) {
		tools.AddNavigationTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetTempo) {
		tools.AddTempoTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetAnnotations) {
		tools.AddAnnotationTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetRendering) {
		tools.AddRenderingTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetSnapshot) {
		tools.AddSnapshotTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetCloudWatch) {
		tools.AddCloudWatchTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetCloudLogging) {
		tools.AddCloudLoggingTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetExamples) {
		tools.AddExamplesTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetSQL) {
		tools.AddSQLTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetRunPanelQuery) {
		tools.AddRunPanelQueryTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetGraphite) {
		tools.AddGraphiteTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetPlugin) {
		tools.AddPluginTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetAPI) {
		tools.AddAPITools(srv, true, true)
	}
	if settings.isToolsetEnabled(ToolsetConfig) {
		tools.AddConfigTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetProvisioning) {
		tools.AddProvisioningTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetAgento11y) {
		tools.AddAgento11yTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetAssistant) {
		tools.AddAssistantTools(srv, true)
	}
	if settings.isToolsetEnabled(ToolsetDocs) {
		tools.AddDocsTools(srv)
	}
	if settings.isToolsetEnabled(ToolsetUser) {
		tools.AddUserTools(srv)
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

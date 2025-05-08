package plugin

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/server"

	"github.com/grafana/grafana-llm-app/pkg/mcp"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/mcp-grafana/tools"
)

// PluginVersion is the version of the plugin, as stored in the plugin.json
// file. The `main` function will set this variable to the current version,
// which is in turn set by Mage using Go linker flags.
var PluginVersion = "development"

// Make sure App implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. Plugin should not implement all these interfaces - only those which are
// required for a particular task.
var (
	_ backend.CallResourceHandler   = (*App)(nil)
	_ instancemgmt.InstanceDisposer = (*App)(nil)
	_ backend.CheckHealthHandler    = (*App)(nil)
	_ backend.StreamHandler         = (*App)(nil)
)

// App is an example app backend plugin which can respond to data queries.
type App struct {
	backend.CallResourceHandler

	vectorService vector.Service

	healthCheckMutex  sync.Mutex
	healthLLMProvider *llmProviderHealthDetails
	healthVector      *vectorHealthDetails
	settings          *Settings
	saToken           string
	grafanaAppURL     string

	// ignoreResponsePadding is a flag to ignore padding in responses.
	// It should only ever be set in tests.
	ignoreResponsePadding bool

	mcpServer *mcp.GrafanaLiveServer
}

// NewApp creates a new example *App instance.
func NewApp(ctx context.Context, appSettings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	log.DefaultLogger.Debug("Creating new app instance")
	var app App
	var err error

	log.DefaultLogger.Debug("Loading settings")
	app.settings, err = loadSettings(appSettings)
	if err != nil {
		log.DefaultLogger.Error("Error loading settings", "err", err)
		return nil, err
	}

	if app.settings.Models == nil {
		// backwards-compat: if Model settings is nil, use the default one
		app.settings.Models = DEFAULT_MODEL_SETTINGS
	}

	// Use a httpadapter (provided by the SDK) for resource calls. This allows us
	// to use a *http.ServeMux for resource calls, so we can map multiple routes
	// to CallResource without having to implement extra logic.
	mux := http.NewServeMux()
	app.registerRoutes(mux)
	app.CallResourceHandler = httpadapter.New(mux)

	// Getting the service account token that has been shared with the plugin
	cfg := backend.GrafanaConfigFromContext(ctx)
	app.saToken, err = cfg.PluginAppClientSecret()
	if err != nil {
		log.DefaultLogger.Warn("Unable to get service account token", "err", err)
	}

	// The Grafana URL is required to request Grafana API later
	app.grafanaAppURL = strings.TrimRight(os.Getenv("GF_APP_URL"), "/")
	if app.grafanaAppURL == "" {
		// For debugging purposes only
		app.grafanaAppURL = "http://localhost:3000"
	}

	if app.settings.Vector.Enabled {
		log.DefaultLogger.Debug("Creating vector service")
		app.vectorService, err = vector.NewService(
			app.settings.Vector,
			appSettings.DecryptedSecureJSONData,
		)
		if err != nil {
			log.DefaultLogger.Error("Error creating vector service", "err", err)
			return nil, err
		}
	}

	app.healthCheckMutex = sync.Mutex{}

	if app.settings.MCP.Enabled {
		app.mcpServer = newMCPServer()
	}

	return &app, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created.
func (a *App) Dispose() {
	if a.vectorService != nil {
		a.vectorService.Cancel()
	}
	if a.mcpServer != nil {
		a.mcpServer.Close()
	}
}

func newMCPServer() *mcp.GrafanaLiveServer {
	srv := server.NewMCPServer("grafana-llm-app", PluginVersion)
	tools.AddDatasourceTools(srv)
	tools.AddSearchTools(srv)
	tools.AddIncidentTools(srv)
	tools.AddPrometheusTools(srv)
	tools.AddLokiTools(srv)
	tools.AddAlertingTools(srv)
	tools.AddOnCallTools(srv)
	tools.AddAssertsTools(srv)
	s := mcp.NewGrafanaLiveServer(srv, mcp.WithGrafanaLiveContextFunc(mcp.ContextFunc))
	return s
}

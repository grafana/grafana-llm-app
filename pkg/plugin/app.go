package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/oauthtokenretriever"
	"github.com/grafana/llm/pkg/plugin/vector/embedding"
	"github.com/grafana/llm/pkg/plugin/vector/store"
)

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

const openAIKey = "openAIKey"

type Settings struct {
	OpenAIURL            string `json:"openAIUrl"`
	OpenAIOrganizationID string `json:"openAIOrganizationId"`

	openAIKey string

	EmbeddingSettings   embedding.EmbeddingClientSettings `json:"embeddings"`
	VectorStoreSettings store.VectorStoreClientSettings   `json:"vectorStore"`
}

func loadSettings(appSettings backend.AppInstanceSettings) Settings {
	settings := Settings{
		OpenAIURL: "https://api.openai.com",
	}
	_ = json.Unmarshal(appSettings.JSONData, &settings)

	settings.openAIKey = appSettings.DecryptedSecureJSONData[openAIKey]
	return settings
}

// App is an example app backend plugin which can respond to data queries.
type App struct {
	backend.CallResourceHandler

	httpClient          *http.Client
	grafanaAppURL       string
	tokenRetriever      oauthtokenretriever.TokenRetriever
	ctx                 context.Context
	cancel              context.CancelFunc
	embeddingSettings   embedding.EmbeddingClientSettings
	vectorStoreSettings store.VectorStoreClientSettings
}

// NewApp creates a new example *App instance.
func NewApp(appSettings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	log.DefaultLogger.Info("Creating new app instance")
	var app App

	// Use a httpadapter (provided by the SDK) for resource calls. This allows us
	// to use a *http.ServeMux for resource calls, so we can map multiple routes
	// to CallResource without having to implement extra logic.
	mux := http.NewServeMux()
	app.registerRoutes(mux)
	app.CallResourceHandler = httpadapter.New(mux)

	var err error
	app.tokenRetriever, err = oauthtokenretriever.New()
	if err != nil {
		log.DefaultLogger.Warn("Error creating token retriever, vector sync will not run", "error", err)
		return &app, nil
	}

	// The Grafana URL is required to obtain tokens later on
	app.grafanaAppURL = strings.TrimRight(os.Getenv("GF_APP_URL"), "/")
	if app.grafanaAppURL == "" {
		// For debugging purposes only
		app.grafanaAppURL = "http://localhost:3000"
	}

	opts, err := appSettings.HTTPClientOptions()
	if err != nil {
		return nil, fmt.Errorf("http client options: %w", err)
	}
	app.httpClient, err = httpclient.New(opts)
	if err != nil {
		return nil, fmt.Errorf("httpclient new: %w", err)
	}

	// TODO: add embedding settings & vector store settings to app
	app.ctx, app.cancel = context.WithCancel(context.Background())
	app.startVectorSync(app.ctx)

	return &app, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created.
func (a *App) Dispose() {
	a.cancel()
}

// CheckHealth handles health checks sent from Grafana to the plugin.
func (a *App) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Info("check health")
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "ok",
	}, nil
}

func (a *App) grafanaClient(ctx context.Context) (*gapi.Client, error) {
	token, err := a.tokenRetriever.Self(ctx)
	if err != nil {
		return nil, fmt.Errorf("get OAuth token for Grafana: %w", err)
	}
	g, err := gapi.New(a.grafanaAppURL, gapi.Config{
		APIKey: token,
		Client: a.httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("create Grafana client: %w", err)
	}
	return g, nil
}

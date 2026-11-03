package plugin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/grafana/grafana-llm-app/pkg/mcp"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/embed"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
)

const (
	openAIKey                = "openAIKey"
	encodedTenantAndTokenKey = "base64EncodedAccessToken"
)

var (
	// This has to be a var so we can take its address later.
	defaultOpenAIAPIPath = "/v1"
)

type ProviderType string

const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAzure     ProviderType = "azure"
	ProviderTypeCustom    ProviderType = "custom"
	ProviderTypeGrafana   ProviderType = "grafana" // via llm-gateway
	ProviderTypeTest      ProviderType = "test"
	ProviderTypeAnthropic ProviderType = "anthropic"
)

// OpenAISettings contains the user-specified OpenAI connection details
type OpenAISettings struct {
	// The URL to the OpenAI provider
	URL string `json:"url"`

	// The API path to append to the URL.
	// If nil, the default path of /v1 is used.
	APIPath *string `json:"apiPath"`

	// The OrgID to be passed to OpenAI in requests
	OrganizationID string `json:"organizationId"`

	// What OpenAI provider the user selected. Note this can specify using the LLMGateway
	// Deprecated: Use Settings.Provider instead
	Provider ProviderType `json:"provider"`

	// Model mappings required for Azure's OpenAI
	AzureMapping [][]string `json:"azureModelMapping"`

	// Optional Azure OpenAI API version. When empty, go-openai's default is used.
	APIVersion string `json:"apiVersion,omitempty"`

	// Disabled marks if a user has explicitly disabled LLM functionality.
	// Deprecated: Use Settings.Disabled instead
	Disabled bool `json:"disabled"`

	// apiKey is the user-specified  api key needed to authenticate requests to the OpenAI
	// provider (excluding the LLMGateway). Stored securely.
	apiKey string

	// TestProvider contains the settings for the test provider.
	// Only used when Provider is ProviderTypeTest.
	TestProvider testProvider `json:"testProvider,omitempty"`
}

// AnthropicSettings contains Anthropic-specific settings
type AnthropicSettings struct {
	// The URL to the provider's API
	URL string `json:"url"`

	// apiKey is the provider-specific API key needed to authenticate requests
	// Stored securely.
	apiKey string
}

// Configured returns whether the provider has been configured
func (s *Settings) Configured() bool {
	// If disabled has been selected than the provider has been configured.
	if s.Disabled || s.OpenAI.Disabled { // Check both for backward compatibility
		return true
	}

	// For backwards compatibility, check OpenAI.Provider if Settings.Provider is empty
	provider := s.Provider
	if provider == "" {
		provider = s.OpenAI.Provider
	}

	switch provider {
	case ProviderTypeGrafana, ProviderTypeCustom, ProviderTypeTest:
		return true
	case ProviderTypeAzure:
		// Require some mappings for use with Azure.
		if len(s.OpenAI.AzureMapping) == 0 {
			return false
		}
		// Still need to check the same conditions as openAIProviderOpenAI.
		fallthrough
	case ProviderTypeOpenAI:
		return s.OpenAI.apiKey != ""
	case ProviderTypeAnthropic:
		return s.Anthropic.apiKey != ""
	}
	// Unknown or empty provider means configuration needs to be updated.
	return false
}

type ModelMapping struct {
	Model Model  `json:"model"`
	Name  string `json:"name"`
}

type ModelSettings struct {
	// Default model to use when no model is defined, or the model is not found.
	Default Model `json:"default"`

	// Mapping is mapping from our abstract model names to the provider's model names.
	Mapping map[Model]string `json:"mapping"`
}

func (c ModelSettings) getModel(model Model) string {
	// Helper function to get the name of a model.
	if name, ok := c.Mapping[model]; ok {
		return name
	}
	// If the model is not found, return the default model.
	return c.getModel(c.Default)
}

func defaultModelSettings(provider ProviderType) *ModelSettings {
	switch provider {
	case ProviderTypeAnthropic:
		return &ModelSettings{
			Default: ModelBase,
			Mapping: map[Model]string{
				ModelBase:  string(anthropic.ModelClaudeSonnet5),
				ModelLarge: string(anthropic.ModelClaudeSonnet5),
			},
		}
	default:
		return &ModelSettings{
			Default: ModelBase,
			Mapping: map[Model]string{
				ModelBase:  openai.GPT4Dot1Mini,
				ModelLarge: openai.GPT4Dot1,
			},
		}
	}
}

// LLMGatewaySettings contains the configuration for the Grafana Managed Key LLM solution.
type LLMGatewaySettings struct {
	// This is the URL of the LLM endpoint of the machine learning backend which proxies
	// the request to our llm-gateway. If empty, the gateway is disabled.
	URL string `json:"url"`
}

type MCPSettings struct {
	Disabled bool `json:"disabled"`
	// Nil (omitted) fields fall back to the plugin default for that toolset;
	// see mcp.IsDefaultEnabled. Set to false to disable.
	Toolsets MCPToolsets `json:"toolsets"`
}

// A nil pointer means the toolset uses its default (mcp.IsDefaultEnabled); a
// pointer to false disables it.
type MCPToolsets struct {
	Search        *bool `json:"search"`
	Datasource    *bool `json:"datasource"`
	Incident      *bool `json:"incident"`
	Prometheus    *bool `json:"prometheus"`
	Loki          *bool `json:"loki"`
	Elasticsearch *bool `json:"elasticsearch"`
	Quickwit      *bool `json:"quickwit"`
	InfluxDB      *bool `json:"influxdb"`
	Alerting      *bool `json:"alerting"`
	Dashboard     *bool `json:"dashboard"`
	Folder        *bool `json:"folder"`
	OnCall        *bool `json:"oncall"`
	Asserts       *bool `json:"asserts"`
	Admin         *bool `json:"admin"`
	Pyroscope     *bool `json:"pyroscope"`
	Navigation    *bool `json:"navigation"`
	Tempo         *bool `json:"tempo"`
	Annotations   *bool `json:"annotations"`
	Rendering     *bool `json:"rendering"`
	Snapshot      *bool `json:"snapshot"`
	CloudWatch    *bool `json:"cloudwatch"`
	CloudLogging  *bool `json:"cloudlogging"`
	Examples      *bool `json:"examples"`
	SQL           *bool `json:"sql"`
	RunPanelQuery *bool `json:"runpanelquery"`
	Graphite      *bool `json:"graphite"`
	Plugin        *bool `json:"plugin"`
	API           *bool `json:"api"`
	Config        *bool `json:"config"`
	Provisioning  *bool `json:"provisioning"`
	Agento11y     *bool `json:"agento11y"`
	Assistant     *bool `json:"assistant"`
	Docs          *bool `json:"docs"`
	User          *bool `json:"user"`

	// ClickHouse is the deprecated name for the SQL toolset.
	ClickHouse *bool `json:"clickhouse"`
}

// isEnabledSetting resolves one toolset setting, falling back to the toolset's
// default when the setting is unset.
func isEnabledSetting(ptr *bool, toolset mcp.Toolset) bool {
	if ptr != nil {
		return *ptr
	}
	return mcp.IsDefaultEnabled(toolset)
}

// An unknown toolset name logs a warning and returns false.
func (f MCPToolsets) IsEnabled(toolset mcp.Toolset) bool {
	switch toolset {
	case mcp.ToolsetSearch:
		return isEnabledSetting(f.Search, toolset)
	case mcp.ToolsetDatasource:
		return isEnabledSetting(f.Datasource, toolset)
	case mcp.ToolsetIncident:
		return isEnabledSetting(f.Incident, toolset)
	case mcp.ToolsetPrometheus:
		return isEnabledSetting(f.Prometheus, toolset)
	case mcp.ToolsetLoki:
		return isEnabledSetting(f.Loki, toolset)
	case mcp.ToolsetElasticsearch:
		return isEnabledSetting(f.Elasticsearch, toolset)
	case mcp.ToolsetQuickwit:
		return isEnabledSetting(f.Quickwit, toolset)
	case mcp.ToolsetInfluxDB:
		return isEnabledSetting(f.InfluxDB, toolset)
	case mcp.ToolsetAlerting:
		return isEnabledSetting(f.Alerting, toolset)
	case mcp.ToolsetDashboard:
		return isEnabledSetting(f.Dashboard, toolset)
	case mcp.ToolsetFolder:
		return isEnabledSetting(f.Folder, toolset)
	case mcp.ToolsetOnCall:
		return isEnabledSetting(f.OnCall, toolset)
	case mcp.ToolsetAsserts:
		return isEnabledSetting(f.Asserts, toolset)
	case mcp.ToolsetAdmin:
		return isEnabledSetting(f.Admin, toolset)
	case mcp.ToolsetPyroscope:
		return isEnabledSetting(f.Pyroscope, toolset)
	case mcp.ToolsetNavigation:
		return isEnabledSetting(f.Navigation, toolset)
	case mcp.ToolsetTempo:
		return isEnabledSetting(f.Tempo, toolset)
	case mcp.ToolsetAnnotations:
		return isEnabledSetting(f.Annotations, toolset)
	case mcp.ToolsetRendering:
		return isEnabledSetting(f.Rendering, toolset)
	case mcp.ToolsetSnapshot:
		return isEnabledSetting(f.Snapshot, toolset)
	case mcp.ToolsetCloudWatch:
		return isEnabledSetting(f.CloudWatch, toolset)
	case mcp.ToolsetCloudLogging:
		return isEnabledSetting(f.CloudLogging, toolset)
	case mcp.ToolsetExamples:
		return isEnabledSetting(f.Examples, toolset)
	case mcp.ToolsetSQL:
		// ClickHouse is the deprecated name for this toolset, so either
		// setting can turn it off.
		return isEnabledSetting(f.SQL, toolset) && isEnabledSetting(f.ClickHouse, toolset)
	case mcp.ToolsetRunPanelQuery:
		return isEnabledSetting(f.RunPanelQuery, toolset)
	case mcp.ToolsetGraphite:
		return isEnabledSetting(f.Graphite, toolset)
	case mcp.ToolsetPlugin:
		return isEnabledSetting(f.Plugin, toolset)
	case mcp.ToolsetAPI:
		return isEnabledSetting(f.API, toolset)
	case mcp.ToolsetConfig:
		return isEnabledSetting(f.Config, toolset)
	case mcp.ToolsetProvisioning:
		return isEnabledSetting(f.Provisioning, toolset)
	case mcp.ToolsetAgento11y:
		return isEnabledSetting(f.Agento11y, toolset)
	case mcp.ToolsetAssistant:
		return isEnabledSetting(f.Assistant, toolset)
	case mcp.ToolsetDocs:
		return isEnabledSetting(f.Docs, toolset)
	case mcp.ToolsetUser:
		return isEnabledSetting(f.User, toolset)
	default:
		log.DefaultLogger.Warn("Unknown MCP toolset", "toolset", toolset)
		return false
	}
}

// Settings contains the plugin's settings and secrets required by the plugin backend.
type Settings struct {
	// Tenant is the stack ID (Hosted Grafana ID) of the instance this plugin
	// is running on.
	Tenant string

	// GrafanaComAPIKey is a grafana.com Editor API key used to interact with the grafana.com API.
	//
	// It is created by the grafana.com API when the plugin is first provisioned for a tenant.
	//
	// It is used when persisting the plugin's settings after setup.
	GrafanaComAPIKey string

	DecryptedSecureJSONData map[string]string

	EnableGrafanaManagedLLM bool `json:"enableGrafanaManagedLLM"`

	// Provider type indicates which provider implementation to use
	Provider ProviderType `json:"provider"`

	// Disabled marks if a user has explicitly disabled LLM functionality.
	Disabled bool `json:"disabled"`

	// OpenAI related settings
	OpenAI OpenAISettings `json:"openAI"`

	// Anthropic related settings
	Anthropic AnthropicSettings `json:"anthropic"`

	// VectorDB settings. May rely on OpenAI settings.
	Vector vector.VectorSettings `json:"vector"`

	// Models contains the user-specified models.
	Models *ModelSettings `json:"models"`

	// LLMGateway provides Grafana-managed OpenAI.
	LLMGateway LLMGatewaySettings `json:"llmGateway"`

	// Allows enabling the dev sandbox on the plugin page.
	EnableDevSandbox bool `json:"enableDevSandbox"`

	// MCP settings.
	MCP MCPSettings `json:"mcp"`
}

func loadSettings(appSettings backend.AppInstanceSettings) (*Settings, error) {
	settings := Settings{OpenAI: OpenAISettings{TestProvider: defaultTestProvider()}}

	if len(appSettings.JSONData) != 0 {
		err := json.Unmarshal(appSettings.JSONData, &settings)
		if err != nil {
			log.DefaultLogger.Error(err.Error())
			return nil, err
		}
	}

	// Handle migration of Disabled field from OpenAI to top level
	if !settings.Disabled && settings.OpenAI.Disabled {
		settings.Disabled = settings.OpenAI.Disabled
	}

	// We need to handle the case where the user has customized the URL,
	// then reverted that customization so that the JSON data includes
	// an empty string.
	if settings.OpenAI.URL == "" {
		settings.OpenAI.URL = "https://api.openai.com"
	}
	// Use default API path if not overridden in settings.
	if settings.OpenAI.APIPath == nil {
		settings.OpenAI.APIPath = &defaultOpenAIAPIPath
	}
	if settings.Anthropic.URL == "" {
		settings.Anthropic.URL = "https://api.anthropic.com"
	}
	if settings.Vector.Embed.Type == embed.EmbedderOpenAI {
		settings.Vector.Embed.OpenAI.URL = settings.OpenAI.URL
		settings.Vector.Embed.OpenAI.AuthType = "openai-key-auth"
	}

	provider := settings.getEffectiveProvider()

	// Verify this is a known provider type
	knownProvider := provider == ProviderTypeOpenAI ||
		provider == ProviderTypeAzure ||
		provider == ProviderTypeCustom ||
		provider == ProviderTypeGrafana ||
		provider == ProviderTypeTest ||
		provider == ProviderTypeAnthropic

	if !knownProvider {
		log.DefaultLogger.Warn("Unknown provider", "provider", settings.Provider)
		settings.OpenAI.Provider = ""
		settings.Provider = ""
	}

	if provider == ProviderTypeGrafana && settings.LLMGateway.URL == "" {
		log.DefaultLogger.Warn("Cannot use LLM Gateway as no URL specified, disabling it")
		settings.OpenAI.Provider = ""
		settings.Provider = ""
	}

	settings.DecryptedSecureJSONData = appSettings.DecryptedSecureJSONData

	settings.OpenAI.apiKey = settings.DecryptedSecureJSONData[openAIKey]
	settings.Anthropic.apiKey = settings.DecryptedSecureJSONData["anthropicKey"]

	// TenantID and GrafanaCom token are combined as "tenantId:GComToken" and base64 encoded, the following undoes that.
	encodedTenantAndToken := settings.DecryptedSecureJSONData[encodedTenantAndTokenKey]
	if encodedTenantAndToken != "" {
		token, err := base64.StdEncoding.DecodeString(encodedTenantAndToken)
		if err != nil {
			log.DefaultLogger.Error(err.Error())
			return nil, err
		}
		tokenParts := strings.Split(strings.TrimSpace(string(token)), ":")
		if len(tokenParts) != 2 {
			return nil, errors.New("invalid access token")
		}
		settings.Tenant = strings.TrimSpace(tokenParts[0])
		if settings.Tenant == "" {
			return nil, errors.New("invalid tenant")
		}
		settings.GrafanaComAPIKey = strings.TrimSpace(tokenParts[1])
		if settings.GrafanaComAPIKey == "" {
			return nil, errors.New("invalid grafana.com API key")
		}
	}

	return &settings, nil
}

// getEffectiveProvider returns the effective provider type, handling backward compatibility
// where Provider was previously stored in OpenAI.Provider
func (s *Settings) getEffectiveProvider() ProviderType {
	if s.Provider == "" {
		return s.OpenAI.Provider
	}
	return s.Provider
}

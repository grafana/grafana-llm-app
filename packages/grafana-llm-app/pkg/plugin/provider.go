package plugin

import "errors"

func createProvider(settings *Settings) (LLMProvider, error) {
	provider := settings.getEffectiveProvider()

	switch provider {
	case ProviderTypeOpenAI, ProviderTypeCustom:
		// Handle the case when the OpenAI provider is set to Azure
		// for backwards compatibility.
		if settings.OpenAI.Provider == ProviderTypeAzure {
			return NewAzureProvider(settings.OpenAI, settings.Models.Default)
		}
		return NewOpenAIProvider(settings.OpenAI, settings.Models)
	case ProviderTypeAzure:
		return NewAzureProvider(settings.OpenAI, settings.Models.Default)
	case ProviderTypeGrafana:
		return NewGrafanaProvider(*settings)
	case ProviderTypeAnthropic:
		return NewAnthropicProvider(settings.Anthropic, settings.Models)
	case ProviderTypeTest:
		return &settings.OpenAI.TestProvider, nil
	default:
		return nil, errors.New("invalid provider configuration")
	}
}

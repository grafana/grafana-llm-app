package plugin

import "errors"

func createProvider(settings *Settings) (LLMProvider, error) {
	provider := settings.getEffectiveProvider()

	switch provider {
	case ProviderTypeOpenAI, ProviderTypeCustom:
		return NewOpenAIProvider(settings.OpenAI, settings.Models)
	case ProviderTypeAzure:
		return NewAzureProvider(settings.OpenAI, settings.Models.Default)
	case ProviderTypeGrafana:
		return NewGrafanaProvider(*settings)
	case ProviderTypeTest:
		return &settings.OpenAI.TestProvider, nil
	default:
		return nil, errors.New("Invalid Provider configuration")
	}
}

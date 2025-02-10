package plugin

import "errors"

func createProvider(settings *Settings) (LLMProvider, error) {
	switch settings.OpenAI.Provider {
	case openAIProviderOpenAI, openAIProviderCustom:
		return NewOpenAIProvider(settings.OpenAI, settings.Models)
	case openAIProviderAzure:
		return NewAzureProvider(settings.OpenAI, settings.Models.Default)
	case openAIProviderGrafana:
		return NewGrafanaProvider(*settings)
	case openAIProviderTest:
		return &settings.OpenAI.TestProvider, nil
	default:
		return nil, errors.New("Invalid OpenAI Provider supplied")
	}
}

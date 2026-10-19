import { validateProviderConfig } from './validation';
import { AppPluginSettings, Secrets, SecretsSet } from './AppConfig';

describe('validateProviderConfig', () => {
  const emptySecrets: Secrets = {};
  const emptySecretsSet: SecretsSet = {
    openAIKey: false,
    anthropicKey: false,
    vectorEmbedderBasicAuthPassword: false,
    vectorStoreBasicAuthPassword: false,
    qdrantApiKey: false,
  };

  test('returns valid when disabled', () => {
    const settings: AppPluginSettings = { disabled: true, provider: 'openai' };
    const res = validateProviderConfig(settings, emptySecrets, emptySecretsSet, false);
    expect(res.isValid).toBe(true);
    expect(res.errors).toHaveLength(0);
  });

  test('returns valid when openAI.disabled is true', () => {
    const settings: AppPluginSettings = { provider: 'openai', openAI: { disabled: true } };
    const res = validateProviderConfig(settings, emptySecrets, emptySecretsSet, false);
    expect(res.isValid).toBe(true);
    expect(res.errors).toHaveLength(0);
  });

  test('returns valid when unconfigured or test provider', () => {
    expect(validateProviderConfig({}, emptySecrets, emptySecretsSet, false).isValid).toBe(true);
    expect(validateProviderConfig({ provider: 'test' }, emptySecrets, emptySecretsSet, false).isValid).toBe(true);
  });

  describe('Grafana-managed provider', () => {
    test('requires optIn to be true', () => {
      const settings: AppPluginSettings = { provider: 'grafana' };
      const invalidRes = validateProviderConfig(settings, emptySecrets, emptySecretsSet, false);
      expect(invalidRes.isValid).toBe(false);
      expect(invalidRes.errors).toContain('You must click the "I Accept" checkbox to use OpenAI provided by Grafana');
      expect(invalidRes.fieldErrors.optIn).toBeDefined();

      const validRes = validateProviderConfig(settings, emptySecrets, emptySecretsSet, true);
      expect(validRes.isValid).toBe(true);
      expect(validRes.errors).toHaveLength(0);
    });
  });

  describe('OpenAI provider', () => {
    test('requires API key if not configured', () => {
      const settings: AppPluginSettings = { provider: 'openai' };
      const res = validateProviderConfig(settings, emptySecrets, emptySecretsSet, false);
      expect(res.isValid).toBe(false);
      expect(res.errors).toContain('OpenAI API key is required');
      expect(res.fieldErrors.apiKey).toBe('OpenAI API key is required');
    });

    test('rejects whitespace-only API key', () => {
      const settings: AppPluginSettings = { provider: 'openai' };
      const res = validateProviderConfig(settings, { openAIKey: '   ' }, emptySecretsSet, false);
      expect(res.isValid).toBe(false);
      expect(res.fieldErrors.apiKey).toBe('OpenAI API key is required');
    });

    test('accepts when new API key is provided', () => {
      const settings: AppPluginSettings = { provider: 'openai' };
      const res = validateProviderConfig(settings, { openAIKey: 'sk-test' }, emptySecretsSet, false);
      expect(res.isValid).toBe(true);
      expect(res.errors).toHaveLength(0);
    });

    test('accepts when API key is already configured on backend', () => {
      const settings: AppPluginSettings = { provider: 'openai' };
      const res = validateProviderConfig(settings, emptySecrets, { ...emptySecretsSet, openAIKey: true }, false);
      expect(res.isValid).toBe(true);
      expect(res.errors).toHaveLength(0);
    });
  });

  describe('Anthropic provider', () => {
    test('requires API key if not configured', () => {
      const settings: AppPluginSettings = { provider: 'anthropic' };
      const res = validateProviderConfig(settings, emptySecrets, emptySecretsSet, false);
      expect(res.isValid).toBe(false);
      expect(res.errors).toContain('Anthropic API key is required');
      expect(res.fieldErrors.apiKey).toBe('Anthropic API key is required');
    });

    test('accepts when anthropicKey is provided or already configured', () => {
      const settings: AppPluginSettings = { provider: 'anthropic' };
      const res1 = validateProviderConfig(settings, { anthropicKey: 'sk-ant-test' }, emptySecretsSet, false);
      expect(res1.isValid).toBe(true);

      const res2 = validateProviderConfig(settings, emptySecrets, { ...emptySecretsSet, anthropicKey: true }, false);
      expect(res2.isValid).toBe(true);
    });
  });

  describe('Custom API provider', () => {
    test('reports multiple errors when both URL and API key are missing', () => {
      const settings: AppPluginSettings = { provider: 'custom' };
      const res = validateProviderConfig(settings, emptySecrets, emptySecretsSet, false);
      expect(res.isValid).toBe(false);
      expect(res.errors).toHaveLength(2);
      expect(res.errors).toContain('Custom API URL is required');
      expect(res.errors).toContain('API key is required for Custom API');
      expect(res.fieldErrors.url).toBe('Custom API URL is required');
      expect(res.fieldErrors.apiKey).toBe('API key is required for Custom API');
    });

    test('reports missing URL when only API key is provided', () => {
      const settings: AppPluginSettings = { provider: 'custom' };
      const res = validateProviderConfig(settings, { openAIKey: 'sk-custom' }, emptySecretsSet, false);
      expect(res.isValid).toBe(false);
      expect(res.errors).toEqual(['Custom API URL is required']);
      expect(res.fieldErrors.url).toBe('Custom API URL is required');
      expect(res.fieldErrors.apiKey).toBeUndefined();
    });

    test('accepts when both URL and API key are provided', () => {
      const settings: AppPluginSettings = { provider: 'custom', openAI: { url: 'https://custom-llm.corp.local' } };
      const res = validateProviderConfig(settings, { openAIKey: 'sk-custom' }, emptySecretsSet, false);
      expect(res.isValid).toBe(true);
      expect(res.errors).toHaveLength(0);
    });
  });

  describe('Azure OpenAI provider', () => {
    test('reports multiple errors when both URL and key are missing', () => {
      const settings: AppPluginSettings = { provider: 'azure' };
      const res = validateProviderConfig(settings, emptySecrets, emptySecretsSet, false);
      expect(res.isValid).toBe(false);
      expect(res.errors).toHaveLength(2);
      expect(res.errors).toContain('Azure OpenAI Language API Endpoint is required');
      expect(res.errors).toContain('Azure OpenAI key is required');
      expect(res.fieldErrors.url).toBe('Azure OpenAI Language API Endpoint is required');
      expect(res.fieldErrors.apiKey).toBe('Azure OpenAI key is required');
    });

    test('accepts when Azure endpoint and key are configured', () => {
      const settings: AppPluginSettings = {
        provider: 'azure',
        openAI: { url: 'https://my-resource.openai.azure.com' },
      };
      const res = validateProviderConfig(settings, emptySecrets, { ...emptySecretsSet, openAIKey: true }, false);
      expect(res.isValid).toBe(true);
      expect(res.errors).toHaveLength(0);
    });
  });
});

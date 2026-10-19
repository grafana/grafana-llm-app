import { AppPluginSettings, Secrets, SecretsSet, getEffectiveProvider } from './AppConfig';

export interface FieldValidationErrors {
  apiKey?: string;
  url?: string;
  optIn?: string;
}

export interface ValidationResult {
  isValid: boolean;
  errors: string[];
  fieldErrors: FieldValidationErrors;
}

/**
 * Validates the required fields for the active LLM provider before saving or health-checking.
 */
export function validateProviderConfig(
  settings: AppPluginSettings,
  secrets: Secrets,
  secretsSet: SecretsSet,
  optIn: boolean
): ValidationResult {
  // If disabled, no validation is required
  if (settings.disabled || settings.openAI?.disabled) {
    return { isValid: true, errors: [], fieldErrors: {} };
  }

  const provider = getEffectiveProvider(settings);

  // If unconfigured or test provider, no validation is required
  if (!provider || provider === 'test') {
    return { isValid: true, errors: [], fieldErrors: {} };
  }

  const errors: string[] = [];
  const fieldErrors: FieldValidationErrors = {};

  const hasSecret = (key: keyof Secrets): boolean => {
    const newSecretValue = secrets[key];
    const isConfigured = Boolean(secretsSet[key]);
    return isConfigured || (typeof newSecretValue === 'string' && newSecretValue.trim().length > 0);
  };

  const hasUrl = (url?: string): boolean => {
    return typeof url === 'string' && url.trim().length > 0;
  };

  switch (provider) {
    case 'grafana': {
      if (!optIn) {
        const msg = 'You must click the "I Accept" checkbox to use OpenAI provided by Grafana';
        errors.push(msg);
        fieldErrors.optIn = msg;
      }
      break;
    }

    case 'openai': {
      if (!hasSecret('openAIKey')) {
        const msg = 'OpenAI API key is required';
        errors.push(msg);
        fieldErrors.apiKey = msg;
      }
      break;
    }

    case 'anthropic': {
      if (!hasSecret('anthropicKey')) {
        const msg = 'Anthropic API key is required';
        errors.push(msg);
        fieldErrors.apiKey = msg;
      }
      break;
    }

    case 'custom': {
      if (!hasUrl(settings.openAI?.url)) {
        const msg = 'Custom API URL is required';
        errors.push(msg);
        fieldErrors.url = msg;
      }
      if (!hasSecret('openAIKey')) {
        const msg = 'API key is required for Custom API';
        errors.push(msg);
        fieldErrors.apiKey = msg;
      }
      break;
    }

    case 'azure': {
      if (!hasUrl(settings.openAI?.url)) {
        const msg = 'Azure OpenAI Language API Endpoint is required';
        errors.push(msg);
        fieldErrors.url = msg;
      }
      if (!hasSecret('openAIKey')) {
        const msg = 'Azure OpenAI key is required';
        errors.push(msg);
        fieldErrors.apiKey = msg;
      }
      break;
    }
  }

  return {
    isValid: errors.length === 0,
    errors,
    fieldErrors,
  };
}

import React, { ChangeEvent } from 'react';

import { Field, FieldSet, Input, SecretInput, Select, useStyles2 } from '@grafana/ui';

import { testIds } from 'components/testIds';
import { getStyles, Secrets, SecretsSet } from './AppConfig';
import { AzureModelDeploymentConfig, AzureModelDeployments } from './AzureConfig';
import { SelectableValue } from '@grafana/data';

export type Provider = 'openai' | 'azure' | 'pulze';
export type PulzeModel = 'pulze' | 'pulze-v0' | 'openai/gpt-4';

export interface ProviderSettings {
  // The URL to reach provider.
  url?: string;
  // The organization ID for provider.
  organizationId?: string;
  // Available providers.
  provider?: Provider;
  // Available Pulze models.
  pulzeModel?: PulzeModel;
  // A mapping of OpenAI models to Azure deployment names.
  azureModelMapping?: AzureModelDeployments;
}

export function ProviderConfig({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
}: {
  settings: ProviderSettings;
  onChange: (settings: ProviderSettings) => void;
  secrets: Secrets;
  secretsSet: SecretsSet;
  onChangeSecrets: (secrets: Secrets) => void;
}) {
  const s = useStyles2(getStyles);
  // Helper to update settings using the name of the HTML event.
  const onChangeField = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({
      ...settings,
      [event.currentTarget.name]:
        event.currentTarget.type === 'checkbox' ? event.currentTarget.checked : event.currentTarget.value.trim(),
    });
  };
  return (
    <FieldSet label="LLM Settings">
      <Field label="LLM Provider">
        <Select
          data-testid={testIds.appConfig.openAIProvider}
          options={
            [
              { label: 'OpenAI', value: 'openai' },
              { label: 'Azure OpenAI', value: 'azure' },
              { label: 'Pulze', value: 'pulze' },
            ] as Array<SelectableValue<Provider>>
          }
          value={settings.provider ?? 'openai'}
          onChange={(e) => onChange({ ...settings, provider: e.value })}
          width={60}
        />
      </Field>
      <Field
        label={settings.provider === 'azure' ? 'Azure OpenAI Language API Endpoint' : 'Provider API URL'}
        className={s.marginTop}
      >
        <Input
          width={60}
          name="url"
          data-testid={testIds.appConfig.openAIUrl}
          value={settings.url}
          placeholder={
            settings.provider === 'azure'
              ? `https://<resource-name>.openai.azure.com`
              : settings.provider === 'pulze'
                ? 'https://api.pulze.ai'
                : `https://api.openai.com`
          }
          onChange={onChangeField}
        />
      </Field>

      <Field
        label={settings.provider === 'azure' ? 'Azure OpenAI Key' : 'Provider API Key'}
        description={settings.provider === 'azure' ? 'Your Azure OpenAI Key' : 'Your Provider API Key'}
      >
        <SecretInput
          width={60}
          data-testid={testIds.appConfig.providerKey}
          name="ProviderKey"
          value={secrets.providerKey}
          isConfigured={secretsSet.providerKey ?? false}
          placeholder={settings.provider === 'azure' ? '' : 'sk-...'}
          onChange={(e) => onChangeSecrets({ ...secrets, providerKey: e.currentTarget.value })}
          onReset={() => onChangeSecrets({ ...secrets, providerKey: '' })}
        />
      </Field>

      {settings.provider === 'pulze' && (
        <Field
          label="Pulze Default Model"
          description="Select the default model with which Pulze will do the request."
        >
        <Select
          data-testid={testIds.appConfig.openAIProvider}
          options={
            [
              { label: 'Pulze', value: 'pulze' },
              { label: 'Pulze-V0', value: 'pulze-v0' },
              { label: 'OpenAI/gpt-4', value: 'openai/gpt-4' },
            ] as Array<SelectableValue<PulzeModel>>
          }
          value={settings.pulzeModel ?? 'pulze'}
          onChange={(e) => onChange({ ...settings, pulzeModel: e.value })}
          width={60}
        />
        </Field>
      )}

      {settings.provider !== 'azure' && (
        <Field label="OpenAI API Organization ID" description="Your OpenAI API Organization ID">
          <Input
            width={60}
            name="organizationId"
            data-testid={testIds.appConfig.openAIOrganizationID}
            value={settings.organizationId}
            placeholder={settings.organizationId ? '' : 'org-...'}
            onChange={onChangeField}
          />
        </Field>
      )}

      {settings.provider === 'azure' && (
        <Field
          label="Azure OpenAI Model Mapping"
          description="Mapping from OpenAI model names to Azure deployment names."
        >
          <AzureModelDeploymentConfig
            modelMapping={settings.azureModelMapping ?? []}
            modelNames={['gpt-3.5-turbo', 'gpt-4']}
            onChange={(azureModelMapping) =>
              onChange({
                ...settings,
                azureModelMapping,
              })
            }
          />
        </Field>
      )}
    </FieldSet>
  );
}

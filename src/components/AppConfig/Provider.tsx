import React, { ChangeEvent } from 'react';

import { Field, FieldSet, Input, SecretInput, Select, useStyles2 } from '@grafana/ui';

import { testIds } from 'components/testIds';
import { getStyles, Secrets, SecretsSet } from './AppConfig';
import { AzureModelDeploymentConfig, AzureModelDeployments } from './AzureConfig';
import { SelectableValue } from '@grafana/data';

export type Provider = 'openai' | 'azure' | 'pulze';

export interface ProviderSettings {
  // The URL to reach provider.
  url?: string;
  // Name of the selected provider.
  name?: Provider;
  // The organization ID for OpenAI.
  organizationId?: string;
  // The default model for Pulze.
  pulzeModel?: string;
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
    <FieldSet label="Provider Settings">
      <Field label="Provider" data-testid={testIds.appConfig.openAIProvider}>
        <Select
          options={
            [
              { label: 'OpenAI', value: 'openai' },
              { label: 'Azure OpenAI', value: 'azure' },
              { label: 'Pulze', value: 'pulze' },
            ] as Array<SelectableValue<Provider>>
          }
          value={settings.name ?? 'openai'}
          onChange={(e) => onChange({ ...settings, name: e.value })}
          width={60}
        />
      </Field>
      <Field
        label={settings.name === 'azure' ? 'Azure OpenAI Language API Endpoint' : 'Provider API URL'}
        className={s.marginTop}
      >
        <Input
          width={60}
          name="url"
          data-testid={testIds.appConfig.openAIUrl}
          value={settings.url}
          placeholder={
            settings.name === 'azure'
              ? `https://<resource-name>.openai.azure.com`
              : settings.name === 'pulze'
              ? 'https://api.pulze.ai/v1'
              : `https://api.openai.com`
          }
          onChange={onChangeField}
        />
      </Field>

      <Field
        label={settings.name === 'azure' ? 'Azure OpenAI Key' : 'Provider API Key'}
        description={settings.name === 'azure' ? 'Your Azure OpenAI Key' : 'Your Provider API Key'}
      >
        <SecretInput
          width={60}
          data-testid={testIds.appConfig.openAIKey}
          name="ProviderKey"
          value={secrets.providerKey}
          isConfigured={secretsSet.providerKey ?? false}
          placeholder={settings.name === 'azure' ? '' : 'sk-...'}
          onChange={(e) => onChangeSecrets({ ...secrets, providerKey: e.currentTarget.value })}
          onReset={() => onChangeSecrets({ ...secrets, providerKey: '' })}
        />
      </Field>

      {settings.name === 'openai' && (
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

      {settings.name === 'azure' && (
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
      {settings.name === 'pulze' && (
        <Field
          label="Default Pulze Model"
          description="The default pulze model to use"
          data-testid={testIds.appConfig.pulzeModel}
        >
          <Select
            options={
              [
                { label: 'pulze', value: 'pulze' },
                { label: 'pulze-v0', value: 'pulze-v0' },
              ] as Array<SelectableValue<Provider>>
            }
            value={settings.pulzeModel ?? 'pulze'}
            onChange={(e) => onChange({ ...settings, pulzeModel: e.value })}
            width={60}
          />
        </Field>
      )}
    </FieldSet>
  );
}

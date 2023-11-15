import React, { ChangeEvent } from 'react';

import { Field, FieldSet, Input, SecretInput, Select, useStyles2 } from '@grafana/ui';

import { testIds } from 'components/testIds';
import { getStyles, Secrets, SecretsSet } from './AppConfig';
import { AzureModelDeploymentConfig, AzureModelDeployments } from './AzureConfig';
import { SelectableValue } from '@grafana/data';

export type OpenAIProvider = 'openai' | 'azure';

export interface OpenAISettings {
  // The URL to reach OpenAI.
  url?: string;
  // The organization ID for OpenAI.
  organizationId?: string;
  // Whether to use Azure OpenAI.
  provider?: OpenAIProvider;
  // A mapping of OpenAI models to Azure deployment names.
  azureModelMapping?: AzureModelDeployments;
}

export function OpenAIConfig({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
}: {
  settings: OpenAISettings;
  onChange: (settings: OpenAISettings) => void;
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
    <FieldSet label="OpenAI Settings">
      <Field label="OpenAI Provider">
        <Select
          data-testid={testIds.appConfig.openAIProvider}
          options={
            [
              { label: 'OpenAI', value: 'openai' },
              { label: 'Azure OpenAI', value: 'azure' },
              { label: 'Pulze', value: 'pulze' },
            ] as Array<SelectableValue<OpenAIProvider>>
          }
          value={settings.provider ?? 'openai'}
          onChange={(e) => onChange({ ...settings, provider: e.value })}
          width={60}
        />
      </Field>
      <Field
        label={settings.provider === 'azure' ? 'Azure OpenAI Language API Endpoint' : 'OpenAI API URL'}
        className={s.marginTop}
      >
        <Input
          width={60}
          name="url"
          data-testid={testIds.appConfig.openAIUrl}
          value={settings.url}
          placeholder={
            settings.provider === 'azure' ? `https://<resource-name>.openai.azure.com` : `https://api.openai.com`
          }
          onChange={onChangeField}
        />
      </Field>

      <Field
        label={settings.provider === 'azure' ? 'Azure OpenAI Key' : 'OpenAI API Key'}
        description={settings.provider === 'azure' ? 'Your Azure OpenAI Key' : 'Your OpenAI API Key'}
      >
        <SecretInput
          width={60}
          data-testid={testIds.appConfig.openAIKey}
          name="openAIKey"
          value={secrets.openAIKey}
          isConfigured={secretsSet.openAIKey ?? false}
          placeholder={settings.provider === 'azure' ? '' : 'sk-...'}
          onChange={(e) => onChangeSecrets({ ...secrets, openAIKey: e.currentTarget.value })}
          onReset={() => onChangeSecrets({ ...secrets, openAIKey: '' })}
        />
      </Field>

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

import React, { ChangeEvent, useEffect, useState } from 'react';

import { openai } from '@grafana/llm';
import { Checkbox, Field, FieldSet, Input, SecretInput, Select, Stack, useStyles2 } from '@grafana/ui';

import { SelectableValue } from '@grafana/data';
import { testIds } from 'components/testIds';
import { getStyles, ProviderType, Secrets, SecretsSet } from './AppConfig';
import { AzureModelDeploymentConfig, AzureModelDeployments } from './AzureConfig';

const OPENAI_API_URL = 'https://api.openai.com';
const AZURE_OPENAI_URL_TEMPLATE = 'https://<resource-name>.openai.azure.com';

export interface OpenAISettings {
  // The URL to reach OpenAI.
  url?: string;
  // The API path to append to the URL.
  // Defaults to /v1 if not provided.
  apiPath?: string;
  // The organization ID for OpenAI.
  organizationId?: string;
  // Whether to use Azure OpenAI.
  provider?: ProviderType;
  // A mapping of OpenAI models to Azure deployment names.
  azureModelMapping?: AzureModelDeployments;
  // If the LLM features have been explicitly disabled.
  disabled?: boolean;
}

export function OpenAIConfig({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
  allowCustomPath = false,
  parentProvider,
}: {
  settings: OpenAISettings;
  onChange: (settings: OpenAISettings) => void;
  secrets: Secrets;
  secretsSet: SecretsSet;
  onChangeSecrets: (secrets: Secrets) => void;
  allowCustomPath: boolean;
  parentProvider?: ProviderType;
}) {
  const s = useStyles2(getStyles);

  // Get the effective provider - use the stored provider, or default to 'openai' if undefined
  const effectiveProvider = settings.provider ?? 'openai';

  // Helper to update settings using the name of the HTML event.
  const onChangeField = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({
      ...settings,
      [event.currentTarget.name]:
        event.currentTarget.type === 'checkbox' ? event.currentTarget.checked : event.currentTarget.value.trim(),
    });
  };

  // Update settings when provider changes, set default URL for OpenAI
  const onChangeProvider = (value: ProviderType) => {
    onChange({
      ...settings,
      provider: value,
      url: value === 'openai' ? OPENAI_API_URL : '',
    });
  };

  // Use separate state to track whether the user has checked the useCustomPath checkbox.
  // This is required because we don't store the checkbox state as a bool in the settings,
  // which our onChangeField assumes we do.
  const [useCustomPath, setUseCustomPath] = useState(settings.apiPath !== undefined);
  useEffect(() => {
    // When the user checks the useCustomPath checkbox we want to immediately
    // update the apiPath field, since the empty string is a valid value;
    // the user shouldn't have to manually modify the input field to trigger
    // the onChange.
    // Similarly when they uncheck the checkbox, we want to clear the apiPath field.
    const apiPath = useCustomPath ? (settings.apiPath ?? '') : undefined;
    onChange({
      ...settings,
      apiPath,
    });
  }, [useCustomPath]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <FieldSet>
      {parentProvider !== 'custom' && (
        <Field label="Provider">
          <Select
            data-testid={testIds.appConfig.provider}
            options={
              [
                { label: 'OpenAI', value: 'openai' },
                { label: 'Azure OpenAI', value: 'azure' },
              ] as Array<SelectableValue<ProviderType>>
            }
            value={effectiveProvider}
            onChange={(e) => onChangeProvider(e.value as ProviderType)}
            width={60}
          />
        </Field>
      )}

      <Field
        label={effectiveProvider === 'azure' ? 'Azure OpenAI Language API Endpoint' : 'API URL'}
        className={s.marginTop}
      >
        <Input
          width={60}
          name="url"
          data-testid={testIds.appConfig.openAIUrl}
          value={effectiveProvider === 'openai' && parentProvider !== 'custom' ? OPENAI_API_URL : settings.url}
          placeholder={
            effectiveProvider === 'azure'
              ? AZURE_OPENAI_URL_TEMPLATE
              : effectiveProvider === 'openai'
                ? OPENAI_API_URL
                : `https://llm.domain.com`
          }
          onChange={onChangeField}
          disabled={effectiveProvider === 'openai' && parentProvider !== 'custom'}
        />
      </Field>

      {allowCustomPath && (
        <Field
          label="API Path"
          description="Customize the API path appended to the URL. Defaults to /v1 if unchecked."
          className={s.marginTop}
        >
          <Stack direction="row" gap={1} alignItems={'center'}>
            <Checkbox
              data-testid={testIds.appConfig.customizeOpenAIApiPath}
              name="Use custom API path"
              value={useCustomPath}
              onChange={(e) => setUseCustomPath(e.currentTarget.checked)}
            />

            <Input
              width={57}
              name="apiPath"
              data-testid={testIds.appConfig.openAIApiPath}
              value={settings.apiPath ?? ''}
              placeholder={useCustomPath ? '' : '/v1'}
              onChange={onChangeField}
              disabled={!useCustomPath}
            />
          </Stack>
        </Field>
      )}

      <Field label={effectiveProvider === 'azure' ? 'Azure OpenAI Key' : 'API Key'}>
        <SecretInput
          width={60}
          data-testid={testIds.appConfig.openAIKey}
          name="openAIKey"
          value={secrets.openAIKey}
          isConfigured={secretsSet.openAIKey ?? false}
          placeholder={effectiveProvider === 'azure' ? '' : 'sk-...'}
          onChange={(e) => onChangeSecrets({ ...secrets, openAIKey: e.currentTarget.value })}
          onReset={() => onChangeSecrets({ ...secrets, openAIKey: '' })}
        />
      </Field>

      {effectiveProvider === 'openai' && (
        <Field label="API Organization ID">
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

      {effectiveProvider === 'azure' && (
        <Field label="Azure OpenAI Model Mapping" description="Mapping from model name to Azure deployment name.">
          <AzureModelDeploymentConfig
            modelMapping={settings.azureModelMapping ?? []}
            modelNames={Object.values(openai.Model)}
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

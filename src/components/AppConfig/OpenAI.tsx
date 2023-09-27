import React, { ChangeEvent } from "react";

import { Field, FieldSet, Input, SecretInput, Switch, useStyles2 } from "@grafana/ui";

import { testIds } from "components/testIds";
import { getStyles, Secrets, SecretsSet } from "./AppConfig";
import { AzureModelDeploymentConfig, AzureModelDeployments } from "./AzureConfig";

export interface OpenAISettings {
  // The URL to reach OpenAI.
  url?: string;
  // The organization ID for OpenAI.
  organizationId?: string;
  // Whether to use Azure OpenAI.
  useAzure?: boolean;
  // A mapping of OpenAI models to Azure deployment names.
  azureModelMapping?: AzureModelDeployments;
}

export function OpenAIConfig({ settings, secrets, secretsSet, onChange, onChangeSecrets }: {
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
      [event.currentTarget.name]: (event.currentTarget.type === 'checkbox' ? event.currentTarget.checked : event.currentTarget.value.trim()),
    });
  };
  return (
    <FieldSet label="OpenAI Settings">
      <Field label="Use Azure OpenAI">
        <Switch
          name="useAzure"
          data-testid={testIds.appConfig.useAzureOpenAI}
          defaultChecked={settings.useAzure}
          checked={settings.useAzure}
          onChange={onChangeField}
        />
      </Field>
      <Field label="OpenAI API URL" description="" className={s.marginTop}>
        <Input
          width={60}
          name="url"
          data-testid={testIds.appConfig.openAIUrl}
          value={settings.url}
          placeholder={settings.useAzure ? `https://<resource-name>.openai.azure.com` : `https://api.openai.com`}
          onChange={onChangeField}
        />
      </Field>

      <Field label="OpenAI API Organization ID" description="Your OpenAI API Organization ID">
        <Input
          width={60}
          name="organizationId"
          data-testid={testIds.appConfig.openAIOrganizationID}
          value={settings.organizationId}
          placeholder={settings.organizationId ? '' : 'org-...'}
          onChange={onChangeField}
          disabled={settings.useAzure}
        />
      </Field>

      <Field label="OpenAI API Key" description="Your OpenAI API Key">
        <SecretInput
          width={60}
          data-testid={testIds.appConfig.openAIKey}
          name="openAIKey"
          value={secrets.openAIKey}
          isConfigured={secretsSet.openAIKey ?? false}
          placeholder={settings.useAzure ? '' : 'sk-...'}
          onChange={e => onChangeSecrets({ ...secrets, openAIKey: e.currentTarget.value })}
          onReset={() => onChangeSecrets({ ...secrets, openAIKey: '' })}
        />
      </Field>

      {settings.useAzure && (
        <Field label="Azure OpenAI Model Mapping" description="">
          <AzureModelDeploymentConfig
            modelMapping={settings.azureModelMapping ?? []}
            modelNames={["gpt-3.5-turbo", "gpt-4"]}
            onChange={
              azureModelMapping => onChange({
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

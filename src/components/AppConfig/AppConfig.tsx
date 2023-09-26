import React, { ChangeEvent, useState } from 'react';
import { lastValueFrom } from 'rxjs';
import { css } from '@emotion/css';
import { AppPluginMeta, GrafanaTheme2, PluginConfigPageProps, PluginMeta } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { Button, Field, FieldSet, Input, SecretInput, useStyles2, Switch, InlineField, IconButton, Select, InlineFieldRow } from '@grafana/ui';
import { testIds } from '../testIds';

type OpenAISettings = {
  url?: string;
  organizationId?: string;
  useAzure?: boolean;
  azureModelMapping?: ModelDeploymentMap;
}

type ModelDeploymentMap = Array<[string, string]>;

export type AppPluginSettings = {
  openAI?: OpenAISettings;
};

type State = {
  // The URL to reach our custom API.
  openAIUrl: string;
  // The Organization ID for our custom API.
  openAIOrganizationID: string;
  // Tells us if the API key secret is set.
  isOpenAIKeySet: boolean;
  // A secret key for our custom API.
  openAIKey: string;
  // A flag to tell us if we should use Azure OpenAI.
  useAzureOpenAI: boolean;
  // A mapping of Azure models to OpenAI models.
  azureModelMapping: ModelDeploymentMap;
  // A flag to tell us if state was updated
  updated: boolean;
};

export interface AppConfigProps extends PluginConfigPageProps<AppPluginMeta<AppPluginSettings>> { }

function ModelMappingConfig({ modelMapping, modelNames, onChange }: {
  modelMapping: Array<[string, string]>;
  modelNames: string[];
  onChange: (modelMapping: Array<[string, string]>) => void;
}) {
  return (
    <>
      <IconButton
        name="plus"
        aria-label="Add model mapping"
        onClick={e => {
          e.preventDefault();
          onChange([...modelMapping, ['', '']]);
        }}
      />
      {modelMapping.map(([model, deployment], i) => (
        <ModelMappingField
          key={i}
          model={model}
          deployment={deployment}
          modelNames={modelNames}
          onChange={(model, deployment) => {
            onChange([
              ...modelMapping.slice(0, i),
              [model, deployment],
              ...modelMapping.slice(i + 1),
            ]);
          }}
          onRemove={() => onChange([
            ...modelMapping.slice(0, i),
            ...modelMapping.slice(i + 1),
          ])}
        />
      )
      )}
    </>
  );
}

function ModelMappingField({ model, deployment, modelNames, onChange, onRemove }: {
  model: string;
  deployment: string;
  modelNames: string[],
  onChange: (model: string, deployment: string) => void;
  onRemove: () => void;
}): JSX.Element {
  return (
    <InlineFieldRow>
      <InlineField label="Model">
        <Select
          placeholder="model label"
          options={modelNames
            .filter(n => n !== deployment && n !== '')
            .map(value => ({ label: value, value }))
          }
          value={model}
          onChange={event => event.value !== undefined && onChange(event.value, deployment)}
        />
      </InlineField>
      <InlineField label="Deployment">
        <Input
          width={40} 
          name="AzureDeployment"
          placeholder="deployment name"
          value={deployment}
          onChange={event => event.currentTarget.value !== undefined && onChange(model, event.currentTarget.value)}
        />
      </InlineField>
      <IconButton
        name="trash-alt"
        aria-label="Remove model mapping"
        onClick={e => {
          e.preventDefault();
          onRemove()
        }}
      />
    </InlineFieldRow>
  );
}


export const AppConfig = ({ plugin }: AppConfigProps) => {
  const s = useStyles2(getStyles);
  const { enabled, pinned, jsonData, secureJsonFields } = plugin.meta;
  const [state, setState] = useState<State>({
    openAIUrl: jsonData?.openAI?.url || 'https://api.openai.com',
    openAIOrganizationID: jsonData?.openAI?.organizationId || '',
    openAIKey: '',
    isOpenAIKeySet: Boolean(secureJsonFields?.openAIKey),
    useAzureOpenAI: jsonData?.openAI?.useAzure || false,
    azureModelMapping: jsonData?.openAI?.azureModelMapping || [],
    updated: false,
  });

  const onResetApiKey = () =>
    setState({
      ...state,
      openAIUrl: '',
      openAIKey: '',
      openAIOrganizationID: '',
      isOpenAIKeySet: false,
      useAzureOpenAI: state.useAzureOpenAI,
      azureModelMapping: [],
      updated: true,
    });

  const onChange = (event: ChangeEvent<HTMLInputElement>) => {
    setState({
      ...state,
      [event.target.name]: (event.target.type === 'checkbox' ? event.target.checked : event.target.value.trim()),
      updated: true,
    });
  };

  return (
    <div data-testid={testIds.appConfig.container}>
      <FieldSet label="OpenAI Settings">
        <Field label="Use Azure OpenAI">
          <Switch
            name="useAzureOpenAI"
            data-testid={testIds.appConfig.useAzureOpenAI}
            defaultChecked={state.useAzureOpenAI}
            checked={state.useAzureOpenAI}
            onChange={onChange}
          />
        </Field>
        <Field label="OpenAI API Url" description="" className={s.marginTop}>
          <Input
            width={60}
            name="openAIUrl"
            data-testid={testIds.appConfig.openAIUrl}
            value={state.openAIUrl}
            placeholder={state.useAzureOpenAI ? `https://<resource-name>.openai.azure.com` : `https://api.openai.com`}
            onChange={onChange}
          />
        </Field>

        <Field label="OpenAI API Organization ID" description="Your OpenAI API Organization ID">
          <Input
            width={60} 
            name="openAIOrganizationID"
            data-testid={testIds.appConfig.openAIOrganizationID}
            value={state.openAIOrganizationID}
            placeholder={state.useAzureOpenAI ? '' : 'org-...'}
            onChange={onChange}
            disabled={state.useAzureOpenAI}
          />
        </Field>

        <Field label="OpenAI API Key" description="Your OpenAI API Key">
          <SecretInput
            width={60}
            data-testid={testIds.appConfig.openAIKey}
            name="openAIKey"
            value={state.openAIKey}
            isConfigured={state.isOpenAIKeySet}
            placeholder={state.useAzureOpenAI ? '' : 'sk-...'}
            onChange={onChange}
            onReset={onResetApiKey}
          />
        </Field>

        {state.useAzureOpenAI && (
          <Field label="Azure OpenAI Model Mapping" description="">
            <ModelMappingConfig
              modelMapping={state.azureModelMapping}
              modelNames={["gpt-3.5-turbo", "gpt-4"]}
              onChange={
                azureModelMapping => setState({
                  ...state,
                  azureModelMapping,
                  updated: true,
                })
              }
            />
          </Field>
        )}

        <div className={s.marginTop}>
          <Button
            type="submit"
            data-testid={testIds.appConfig.submit}
            onClick={() =>
              updatePluginAndReload(plugin.meta.id, {
                enabled,
                pinned,
                jsonData: {
                  openAI: {
                    url: state.openAIUrl,
                    organizationId: state.openAIOrganizationID,
                    useAzure: state.useAzureOpenAI,
                    azureModelMapping: state.azureModelMapping,
                  },
                },
                // This cannot be queried later by the frontend.
                // We don't want to override it in case it was set previously and left untouched now.
                secureJsonData: state.isOpenAIKeySet
                  ? undefined
                  : {
                    openAIKey: state.openAIKey,
                  },
              })
            }
            disabled={!state.updated}
          >
            Save API settings
          </Button>
        </div>
      </FieldSet>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  colorWeak: css`
    color: ${theme.colors.text.secondary};
  `,
  marginTop: css`
    margin-top: ${theme.spacing(3)};
  `,
});

const updatePluginAndReload = async (pluginId: string, data: Partial<PluginMeta<AppPluginSettings>>) => {
  try {
    await updatePlugin(pluginId, data);

    // Reloading the page as the changes made here wouldn't be propagated to the actual plugin otherwise.
    // This is not ideal, however unfortunately currently there is no supported way for updating the plugin state.
    window.location.reload();
  } catch (e) {
    console.error('Error while updating the plugin', e);
  }
};

export const updatePlugin = async (pluginId: string, data: Partial<PluginMeta>) => {
  const response = getBackendSrv().fetch({
    url: `/api/plugins/${pluginId}/settings`,
    method: 'POST',
    data,
  });

  return lastValueFrom(response);
};

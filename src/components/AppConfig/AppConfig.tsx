import React, { ChangeEvent, useState } from 'react';
import { lastValueFrom } from 'rxjs';
import { css } from '@emotion/css';
import { AppPluginMeta, GrafanaTheme2, PluginConfigPageProps, PluginMeta } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { Button, Field, FieldSet, Input, SecretInput, useStyles2 } from '@grafana/ui';
import { testIds } from '../testIds';

export type AppPluginSettings = {
  openAIUrl?: string;
};

type State = {
  // The URL to reach our custom API.
  openAIUrl: string;
  // Tells us if the API key secret is set.
  isOpenAIKeySet: boolean;
  // A secret key for our custom API.
  openAIKey: string;
};

export interface AppConfigProps extends PluginConfigPageProps<AppPluginMeta<AppPluginSettings>> {}

export const AppConfig = ({ plugin }: AppConfigProps) => {
  const s = useStyles2(getStyles);
  const { enabled, pinned, jsonData, secureJsonFields } = plugin.meta;
  const [state, setState] = useState<State>({
    openAIUrl: jsonData?.openAIUrl || 'https://api.openai.com',
    openAIKey: '',
    isOpenAIKeySet: Boolean(secureJsonFields?.openAIKey),
  });

  const onResetApiKey = () =>
    setState({
      ...state,
      openAIKey: '',
      isOpenAIKeySet: false,
    });

  const onChange = (event: ChangeEvent<HTMLInputElement>) => {
    setState({
      ...state,
      [event.target.name]: event.target.value.trim(),
    });
  };

  return (
    <div data-testid={testIds.appConfig.container}>
      <FieldSet label="OpenAI Settings">
        <Field label="OpenAI API Url" description="" className={s.marginTop}>
          <Input
            width={60}
            name="openAIUrl"
            data-testid={testIds.appConfig.openAIUrl}
            value={state.openAIUrl}
            placeholder={`https://api.openai.com`}
            onChange={onChange}
          />
        </Field>

        <Field label="OpenAI API Key" description="Your OpenAI API Key">
          <SecretInput
            width={60}
            data-testid={testIds.appConfig.openAIKey}
            name="openAIKey"
            value={state.openAIKey}
            isConfigured={state.isOpenAIKeySet}
            placeholder={'sk-...'}
            onChange={onChange}
            onReset={onResetApiKey}
          />
        </Field>

        <div className={s.marginTop}>
          <Button
            type="submit"
            data-testid={testIds.appConfig.submit}
            onClick={() =>
              updatePluginAndReload(plugin.meta.id, {
                enabled,
                pinned,
                jsonData: {
                  openAIUrl: state.openAIUrl,
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
            disabled={Boolean(!state.openAIUrl || (!state.isOpenAIKeySet && !state.openAIKey))}
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

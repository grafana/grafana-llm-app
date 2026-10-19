import { PluginType } from '@grafana/data';
import { openai } from '@grafana/llm';
import { getBackendSrv } from '@grafana/runtime';
import { render, screen } from '@testing-library/react';
import { testIds } from 'components/testIds';
import React from 'react';
import { of } from 'rxjs';
import { AppConfig, AppConfigProps, updateAndSavePluginSettings } from './AppConfig';

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: jest.fn(),
}));

describe('Components/AppConfig', () => {
  let props: AppConfigProps;

  beforeEach(() => {
    jest.resetAllMocks();

    props = {
      plugin: {
        meta: {
          id: 'sample-app',
          name: 'Sample App',
          type: PluginType.app,
          enabled: true,
          jsonData: {
            displayVectorStoreOptions: true,
            vector: {
              enabled: true,
              store: {
                type: 'qdrant',
              },
            },
          },
        },
      },
      query: {},
    } as unknown as AppConfigProps;
  });

  test('renders OpenAI configuration when provider is OpenAI', () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false, jsonData: { provider: 'openai' } } };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.queryByText('Use OpenAI-compatible API')).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.provider)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /save & test/i })).toBeInTheDocument();
  });

  test('renders Anthropic configuration when provider is Anthropic', () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false, jsonData: { provider: 'anthropic' } } };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.queryByText('Use Anthropic API')).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.anthropicUrl)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.anthropicKey)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /save & test/i })).toBeInTheDocument();
  });

  test('sends only allowlisted fields to Grafana.com and the full settings to Grafana', async () => {
    const fetch = jest.fn().mockReturnValue(of({ ok: true, data: {} }));
    jest.mocked(getBackendSrv).mockReturnValue({ fetch } as unknown as ReturnType<typeof getBackendSrv>);

    const settings = {
      enabled: true,
      pinned: true,
      jsonData: {
        provider: 'openai' as const,
        disabled: false,
        models: { default: openai.Model.BASE, mapping: { [openai.Model.BASE]: 'gpt-4.1-mini' } },
        openAI: { url: 'https://api.openai.com' },
        enableGrafanaManagedLLM: true,
      },
      secureJsonData: {
        openAIKey: 'sk-new',
        anthropicKey: 'anthropic-secret',
      },
    };

    await updateAndSavePluginSettings('grafana-llm-app', true, settings);

    expect(fetch).toHaveBeenCalledTimes(2);
    expect(fetch).toHaveBeenNthCalledWith(1, {
      url: '/api/plugins/grafana-llm-app/resources/save-plugin-settings',
      method: 'POST',
      data: {
        jsonData: {
          provider: 'openai',
          disabled: false,
          models: { default: openai.Model.BASE, mapping: { [openai.Model.BASE]: 'gpt-4.1-mini' } },
        },
        secureJsonData: { openAIKey: 'sk-new' },
      },
    });
    expect(fetch).toHaveBeenNthCalledWith(2, {
      url: '/api/plugins/grafana-llm-app/settings',
      method: 'POST',
      data: settings,
    });
  });
});

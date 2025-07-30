import { PluginType } from '@grafana/data';
import { render, screen } from '@testing-library/react';
import { testIds } from 'components/testIds';
import React from 'react';
import { AppConfig, AppConfigProps } from './AppConfig';

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
});

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

  test('displays validation error and disables save button when OpenAI API key is missing', () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false, jsonData: { provider: 'openai' } } };

    // @ts-ignore
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.getAllByText('OpenAI API key is required').length).toBeGreaterThanOrEqual(1);
    const saveButton = screen.getByRole('button', { name: /save & test/i });
    expect(saveButton).toBeDisabled();
  });

  test('displays multiple validation errors when Custom API URL and key are missing', () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false, jsonData: { provider: 'custom' } } };

    // @ts-ignore
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.getByText('Please resolve the following configuration issues:')).toBeInTheDocument();
    expect(screen.getAllByText('Custom API URL is required').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('API key is required for Custom API').length).toBeGreaterThanOrEqual(1);
    const saveButton = screen.getByRole('button', { name: /save & test/i });
    expect(saveButton).toBeDisabled();
  });

  test('displays validation error when Anthropic API key is missing', () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false, jsonData: { provider: 'anthropic' } } };

    // @ts-ignore
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.getAllByText('Anthropic API key is required').length).toBeGreaterThanOrEqual(1);
    const saveButton = screen.getByRole('button', { name: /save & test/i });
    expect(saveButton).toBeDisabled();
  });

  test('does not display validation error when key is already configured on backend', () => {
    const plugin = {
      meta: {
        ...props.plugin.meta,
        enabled: false,
        jsonData: { provider: 'openai' },
        secureJsonFields: { openAIKey: true },
      },
    };

    // @ts-ignore
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.queryByText('OpenAI API key is required')).not.toBeInTheDocument();
  });
});

import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';

import { testIds } from '../testIds';
import { OpenAIConfig, OpenAISettings } from './OpenAI';

describe('OpenAIConfig Azure API version', () => {
  it('shows and updates the optional version for Azure', () => {
    const onChange = jest.fn<void, [OpenAISettings]>();
    render(
      <OpenAIConfig
        settings={{ provider: 'azure', apiVersion: '2024-08-01-preview' }}
        secrets={{}}
        secretsSet={{}}
        onChange={onChange}
        onChangeSecrets={() => {}}
        allowCustomPath={false}
      />
    );

    const input = screen.getByTestId(testIds.appConfig.azureOpenAIApiVersion);
    expect((input as HTMLInputElement).value).toBe('2024-08-01-preview');

    fireEvent.change(input, { target: { value: ' 2024-10-21 ' } });
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ apiVersion: '2024-10-21' }));
  });

  it('does not show the Azure version for OpenAI', () => {
    render(
      <OpenAIConfig
        settings={{ provider: 'openai' }}
        secrets={{}}
        secretsSet={{}}
        onChange={() => {}}
        onChangeSecrets={() => {}}
        allowCustomPath={false}
      />
    );

    expect(screen.queryByTestId(testIds.appConfig.azureOpenAIApiVersion)).toBeNull();
  });
});

import React from 'react';
import { render } from '@testing-library/react';
import { LLMConfig } from './LLMConfig';

test('renders LLMConfig without invalid DOM structure', () => {
  render(
    <LLMConfig
      settings={{}}
      onChange={() => {}}
      secrets={{}}
      secretsSet={{ openAIKey: false }}
      optIn={false}
      setOptIn={() => {}}
      onChangeSecrets={() => {}}
    />
  );
});

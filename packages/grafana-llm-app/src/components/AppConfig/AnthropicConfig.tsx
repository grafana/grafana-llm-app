import React, { ChangeEvent } from 'react';

import { Field, FieldSet, Input, SecretInput, useStyles2 } from '@grafana/ui';

import { testIds } from 'components/testIds';
import { getStyles, Secrets, SecretsSet } from './AppConfig';

const ANTHROPIC_API_URL = 'https://api.anthropic.com';

export interface AnthropicSettings {
  // The URL to reach Anthropic.
  url?: string;
  // If the LLM features have been explicitly disabled.
  disabled?: boolean;
}

export function AnthropicConfig({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
}: {
  settings: AnthropicSettings;
  onChange: (settings: AnthropicSettings) => void;
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
    <FieldSet>
      <Field label="API URL" className={s.marginTop}>
        <Input
          width={60}
          name="url"
          data-testid={testIds.appConfig.anthropicUrl}
          value={ANTHROPIC_API_URL}
          placeholder={ANTHROPIC_API_URL}
          onChange={onChangeField}
          disabled={true}
        />
      </Field>

      <Field label="API Key">
        <SecretInput
          width={60}
          data-testid={testIds.appConfig.anthropicKey}
          name="anthropicKey"
          value={secrets.anthropicKey}
          isConfigured={secretsSet.anthropicKey ?? false}
          placeholder={secretsSet.anthropicKey ? 'sk-ant-...' : 'not configured'}
          onChange={(e) => onChangeSecrets({ ...secrets, anthropicKey: e.currentTarget.value })}
          onReset={() => onChangeSecrets({ ...secrets, anthropicKey: '' })}
        />
      </Field>
    </FieldSet>
  );
}

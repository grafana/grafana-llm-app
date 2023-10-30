import React from 'react';

import { InlineField, Input, SecretInput, useStyles2 } from '@grafana/ui';

import { testIds } from 'components/testIds';
import { Secrets, SecretsSet, getStyles } from '../AppConfig';

import { AuthSettings } from '../Vector';

export const BasicAuthConfig = ({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
  secretKey,
}: {
  settings?: AuthSettings;
  secrets: Secrets;
  secretsSet: SecretsSet;
  onChange: (settings: AuthSettings) => void;
  onChangeSecrets: (secrets: Secrets) => void;
  secretKey: 'vectorEmbedderBasicAuthPassword' | 'vectorStoreBasicAuthPassword';
}) => {
  const s = useStyles2(getStyles);

  const onPasswordReset = () => {
    onChangeSecrets({
      ...secrets,
      [secretKey]: '',
    });
  };

  const onPasswordChange = (event: React.SyntheticEvent<HTMLInputElement>) => {
    onChangeSecrets({
      ...secrets,
      [secretKey]: event.currentTarget.value,
    });
  };

  return (
    <>
      <InlineField label="User" labelWidth={s.inlineFieldWidth}>
        <Input
          width={s.inlineFieldInputWidth}
          placeholder="user"
          data-testid={testIds.appConfig.basicAuthUsername}
          value={settings?.basicAuthUser}
          onChange={(e) => onChange({ ...settings, basicAuthUser: e.currentTarget.value })}
        />
      </InlineField>
      <InlineField label="Password" labelWidth={s.inlineFieldWidth}>
        <SecretInput
          width={s.inlineFieldInputWidth}
          placeholder="password"
          data-testid={testIds.appConfig.basicAuthPassword}
          value={secrets[secretKey]}
          isConfigured={secretsSet[secretKey] ?? false}
          onReset={onPasswordReset}
          onChange={onPasswordChange}
        />
      </InlineField>
    </>
  );
};

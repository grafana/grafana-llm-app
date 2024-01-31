import React from 'react';

import { HealthCheckResult } from '@grafana/runtime';
import { Alert, AlertVariant, VerticalGroup } from '@grafana/ui';

interface HealthCheckDetails {
  provider: ProviderHealthDetails | boolean;
  vector: VectorHealthDetails | boolean;
  version: string;
}

interface ProviderHealthDetails {
  // Whether the minimum required provider settings have been provided.
  configured: boolean;
  // Whether we can call the provider API with the provided settings.
  ok: boolean;
  // If set, the error returned when trying to call the provider API.
  // Will be undefined if ok is true.
  error?: string;
  // A map of model names to their health details.
  // The health check attempts to call the provider API with each
  // of a few models and records the result of each call here.
  models: Record<string, ProviderModelHealthDetails>;
}

interface ProviderModelHealthDetails {
  // Whether we can use this model in calls to provider.
  ok: boolean;
  // If set, the error returned when trying to call the provider API.
  // Will be undefined if ok is true.
  error?: string;
}

interface VectorHealthDetails {
  // Whether the vector service has been enabled.
  enabled: boolean;
  // Whether we can use the vector service with the provided settings.
  ok: boolean;
  // If set, the error returned when trying to call the vector service.
  // Will be undefined if ok is true.
  error?: string;
}

const isHealthCheckDetails = (obj: unknown): obj is HealthCheckDetails => {
  return typeof obj === 'object' && obj !== null && 'provider' in obj && 'vector' in obj && 'version' in obj;
};

const alertVariants = new Set<AlertVariant>(['success', 'info', 'warning', 'error']);
const isAlertVariant = (str: string): str is AlertVariant => alertVariants.has(str as AlertVariant);
const getAlertVariant = (status: string): AlertVariant => {
  if (status.toLowerCase() === 'ok') {
    return 'success';
  }
  return isAlertVariant(status) ? status : 'info';
};
const getAlertSeverity = (status: string, details: HealthCheckDetails): AlertVariant => {
  const severity = getAlertVariant(status);
  if (severity !== 'success') {
    return severity;
  }
  if (!isHealthCheckDetails(details)) {
    return 'success';
  }
  if (typeof details.provider === 'object' && typeof details.vector === 'object') {
    const vectorOk = !details.vector.enabled || details.vector.ok;
    return details.provider.ok && vectorOk ? 'success' : 'warning';
  }
  return severity;
};

export function ShowHealthCheckResult(props: HealthCheckResult) {
  let severity = getAlertVariant(props.status ?? 'error');
  if (!isHealthCheckDetails(props.details)) {
    return (
      <div className="gf-form-group p-t-2">
        <Alert severity={severity} title={props.message} />
      </div>
    );
  }

  severity = getAlertSeverity(props.status ?? 'error', props.details);
  const showProvider =
    typeof props.details.provider === 'boolean' ||
    (typeof props.details.provider === 'object' && (props.details.provider.configured || !props.details.provider.ok));
  const showVector =
    typeof props.details.vector === 'boolean' ||
    (typeof props.details.vector === 'object' && props.details.vector.enabled);
  return (
    <VerticalGroup>
      {showProvider && <ShowProviderHealth provider={props.details.provider} />}
      {showVector && <ShowVectorHealth vector={props.details.vector} />}
    </VerticalGroup>
  );
}

function ShowProviderHealth({ provider }: { provider: ProviderHealthDetails | boolean }) {
  if (typeof provider === 'boolean') {
    const severity = provider ? 'success' : 'error';
    const message = provider ? 'Provider health check succeeded!' : 'Provider health check failed.';
    return <Alert title={message} severity={severity} />;
  }
  const { message, severity }: { message: string; severity: AlertVariant } = provider.ok
    ? { message: 'Provider health check succeeded!', severity: 'success' }
    : { message: 'Provider health check failed!', severity: 'error' };
  return (
    <Alert severity={severity} title={message}>
      <b>Models</b>
      <div>
        {Object.entries(provider.models).map(([model, details], i) => (
          <li key={i}>
            {model}: {details.ok ? 'OK' : `Error: ${details.error}`}
          </li>
        ))}
      </div>
    </Alert>
  );
}

function ShowVectorHealth({ vector }: { vector: VectorHealthDetails | boolean }) {
  if (typeof vector === 'boolean') {
    const severity = vector ? 'success' : 'error';
    const message = vector ? 'Vector service health check succeeded!' : 'Vector service health check failed.';
    return <Alert title={message} severity={severity} />;
  }
  const severity = vector.ok ? 'success' : 'error';
  const message = vector.ok ? 'Vector service health check succeeded!' : 'Vector service health check failed.';
  return (
    <Alert title={message} severity={severity}>
      {vector.error && <div>Error: {vector.error}</div>}
    </Alert>
  );
}

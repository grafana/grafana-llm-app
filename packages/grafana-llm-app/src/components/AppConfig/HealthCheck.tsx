import React from 'react';

import { HealthCheckResult, config } from '@grafana/runtime';
import { Alert, AlertVariant, VerticalGroup } from '@grafana/ui';

interface HealthCheckDetails {
  openAI: OpenAIHealthDetails | boolean;
  vector: VectorHealthDetails | boolean;
  version: string;
}

interface OpenAIHealthDetails {
  // Whether the minimum required OpenAI settings have been provided.
  configured: boolean;
  // Whether we can call the OpenAI API with the provided settings.
  ok: boolean;
  // If set, the error returned when trying to call the OpenAI API.
  // Will be undefined if ok is true.
  error?: string;
  // A map of model names to their health details.
  // The health check attempts to call the OpenAI API with each
  // of a few models and records the result of each call here.
  models: Record<string, OpenAIModelHealthDetails>;
  assistant: OpenAIAssistantHealthDetails;
}

interface OpenAIModelHealthDetails {
  // Whether we can use this model in calls to OpenAI.
  ok: boolean;
  // If set, the error returned when trying to call the OpenAI API.
  // Will be undefined if ok is true.
  error?: string;
}

interface OpenAIAssistantHealthDetails {
  // Whether we can use the assistant API with the provided settings.
  ok: boolean;
  // If set, the error returned when trying to call the assistant API.
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
  return typeof obj === 'object' && obj !== null && 'openAI' in obj && 'vector' in obj && 'version' in obj;
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
  if (typeof details.openAI === 'object' && typeof details.vector === 'object') {
    const vectorOk = !details.vector.enabled || details.vector.ok;
    return details.openAI.ok && vectorOk ? 'success' : 'warning';
  }
  return severity;
};

export function ShowHealthCheckResult(props: HealthCheckResult) {
  let severity = getAlertVariant(props.status ?? 'error');
  if (!isHealthCheckDetails(props.details)) {
    return <Alert severity={severity} title={props.message} />;
  }

  severity = getAlertSeverity(props.status ?? 'error', props.details);
  const showOpenAI =
    typeof props.details.openAI === 'boolean' ||
    (typeof props.details.openAI === 'object' && props.details.openAI.configured);
  const showVector =
    typeof props.details.vector === 'boolean' ||
    (typeof props.details.vector === 'object' && props.details.vector.enabled);
  return (
    <VerticalGroup>
      <ShowGrafanaHealth />
      {showOpenAI && <ShowOpenAIHealth openAI={props.details.openAI} />}
      {showVector && <ShowVectorHealth vector={props.details.vector} />}
    </VerticalGroup>
  );
}

function ShowGrafanaHealth() {
  if (config.liveEnabled) {
    return null;
  }
  return (
    <Alert
      title="Grafana Live is disabled"
      severity="warning"
    >
      <div>
        Grafana Live is disabled. This plugin requires Grafana Live to be enabled in order to function correctly.
      </div>
      <div>
        Set the <a href="https://grafana.com/docs/grafana/latest/setup-grafana/configure-grafana/#max_connections"><code>max_connections</code></a> setting to a non-zero value
        in the Grafana configuration file to enable Grafana Live.
      </div>
    </Alert>
  )
}

function ShowOpenAIHealth({ openAI }: { openAI: OpenAIHealthDetails | boolean }) {
  if (typeof openAI === 'boolean') {
    const severity = openAI ? 'success' : 'error';
    const message = openAI ? 'OpenAI health check succeeded!' : 'OpenAI health check failed.';
    return <Alert title={message} severity={severity} />;
  }
  const message = openAI.ok ? 'OpenAI health check succeeded!' : 'OpenAI health check failed.';
  const severity = openAI.ok ? 'success' : 'error';
  return (
    <Alert severity={severity} title={message}>
      <b>Models</b>
      <div>
        {Object.entries(openAI.models).map(([model, details], i) => (
          <li key={i}>
            {model}: {details.ok ? 'OK' : `Error: ${details.error}`}
          </li>
        ))}
      </div>
      <b>Assistant: </b>
      {openAI.assistant.ok ? 'OK' : `Error: ${openAI.assistant.error}. The configured OpenAI provider may not offer assistants APIs.`}
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

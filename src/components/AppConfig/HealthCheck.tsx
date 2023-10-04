import React from 'react';

import { HealthCheckResult } from "@grafana/runtime";
import { Alert, AlertVariant } from "@grafana/ui";

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
}

interface OpenAIModelHealthDetails {
  // Whether we can use this model in calls to OpenAI.
  ok: boolean;
  // If set, the error returned when trying to call the OpenAI API.
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
}

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
}

export function ShowHealthCheckResult(props: HealthCheckResult) {
  let severity = getAlertVariant(props.status ?? 'error');
  if (!isHealthCheckDetails(props.details)) {
    return (
      <div className="gf-form-group p-t-2">
        <Alert severity={severity} title={props.message}>
        </Alert>
      </div>
    );
  }

  severity = getAlertSeverity(props.status ?? 'error', props.details);
  const message = severity === 'success' ? 'Health check succeeded!' : props.message;

  return (
    <div className="gf-form-group p-t-2">
      <Alert severity={severity} title={message}>
        <ShowOpenAIHealth openAI={props.details.openAI} />
        <ShowVectorHealth vector={props.details.vector} />
      </Alert>
    </div>
  );
}

function ShowOpenAIHealth({ openAI }: { openAI: OpenAIHealthDetails | boolean }) {
  if (typeof openAI === 'boolean') {
    return <div>OpenAI: {openAI ? 'Enabled' : 'Disabled'}</div>;
  }
  return (
    <div>
      <h5>OpenAI</h5>
      <div>{openAI.ok ? 'OK' : `Error: ${openAI.error}`}</div>
      <b>Models</b>
      <table>
        <thead>
        </thead>
        <tbody>
          {Object.entries(openAI.models).map(([model, details], i) => (
            <tr key={i}>
              <td>{model}: </td>
              <td>{details.ok ? 'OK' : `Error: ${details.error}`}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ShowVectorHealth({ vector }: { vector: VectorHealthDetails | boolean }) {
  if (typeof vector === 'boolean') {
    return <div>Vector: {vector ? 'Enabled' : 'Disabled'}</div>;
  }
  return (
    <div>
      <h5>Vector service</h5>
      <div>{vector.enabled ? 'Enabled' : 'Disabled'}</div>
      {vector.enabled && (
        <div>{vector.ok ? 'OK' : `Error: ${vector.error}`}</div>
      )}
    </div>
  )
}

export interface HealthCheckResponse {
  status: 'ok' | 'error';
  details?: HealthCheckDetails;
}

export interface HealthCheckDetails {
  openAI: OpenAIHealthDetails | boolean;
  vector: VectorHealthDetails | boolean;
  version: string;
}

export interface OpenAIHealthDetails {
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
  models?: Record<string, OpenAIModelHealthDetails>;
}

export interface OpenAIModelHealthDetails {
  // Whether we can use this model in calls to OpenAI.
  ok: boolean;
  // If set, the error returned when trying to call the OpenAI API.
  // Will be undefined if ok is true.
  error?: string;
}

export interface VectorHealthDetails {
  // Whether the vector service has been enabled.
  enabled: boolean;
  // Whether we can use the vector service with the provided settings.
  ok: boolean;
  // If set, the error returned when trying to call the vector service.
  // Will be undefined if ok is true.
  error?: string;
}

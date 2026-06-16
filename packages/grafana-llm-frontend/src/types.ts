export interface HealthCheckResponse {
  status: "ok" | "error";
  details?: HealthCheckDetails;
}

export interface HealthCheckDetails {
  llmProvider: LLMProviderHealthDetails | boolean;
  vector: VectorHealthDetails | boolean;
  version: string;
}

export interface LLMProviderHealthDetails {
  // Whether the minimum required LLM provider settings have been provided.
  configured: boolean;
  // Whether we can call the LLM provider API with the provided settings.
  ok: boolean;
  // If set, the error returned when trying to call the LLM provider API.
  // Will be undefined if ok is true.
  error?: string;
  // A map of model names to their health details.
  // The health check attempts to call the provider API with each
  // of the configured models and records the result of each call here.
  models?: Record<string, ModelHealthDetails>;
  // Health details for the provider's assistant APIs.
  assistant?: ModelHealthDetails;
}

export interface ModelHealthDetails {
  // Whether we can use this model in calls to the provider.
  ok: boolean;
  // If set, the error returned when trying to call the provider API.
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

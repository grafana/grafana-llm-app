# Changelog

## Unreleased

## 0.7.0

* Refactors repo into monorepo together with frontend dependencies
* Creates developer sandbox for developing frontend dependencies
* Switches CI/CD to github actions

## 0.6.3

- Fix additional UI bugs
- Fix issue where health check returned true even if LLM was disabled

## 0.6.2

- Fix UI issues around OpenAI provider introduced in 0.6.1

## 0.6.1

- Store Grafana-managed OpenAI opt-in in ML cloud backend DB (Grafana Cloud only)
- Updated Grafana-managed OpenAI opt-in messaging (Grafana Cloud only)
- UI update for LLM provider selection

## 0.6.0

- Add Grafana-managed OpenAI as a provider option (Grafana Cloud only)

## 0.5.2

- Allow Qdrant API key to be configured in config UI, not just when provisioning

## 0.5.1

- Fix issue where temporary errors were cached, causing /health to fail permanently.

## 0.5.0

- Add basic auth to VectorAPI

## 0.4.0

- Add 'Enabled' switch for vector services to configuration UI
- Added instructions for developing with example app
- Improve health check to return more granular details
- Add support for filtered vector search
- Improve vector service health check

## 0.3.0

- Add Go package providing an OpenAI client to use the LLM app from backend Go code
- Add support for Azure OpenAI. The plugin must be configured to use OpenAI and provide a link between OpenAI model names and Azure deployment names
- Return streaming errors as part of the stream, with objects like `{"error": "<error message>"}`

## 0.2.1

- Improve health check endpoint to include status of various features
- Change path handling for chat completions streams to put separate requests into separate streams. Requests can pass a UUID as the suffix of the path now, but is backwards compatible with an older version of the frontend code.

## 0.2.0

- Expose vector search API to perform semantic search against a vector database using a configurable embeddings source

## 0.1.0

- Support proxying LLM requests from Grafana to OpenAI

# Changelog

## Unreleased

## 0.10.9

- Update wording around OpenAI and open models
- Update LLM setup instructions

## 0.10.8

- Bump various dependencies to fix security issues (e.g. #446)

## 0.10.7

- Use non-crypto UUID generation for stream request #409

## 0.10.6

- Fix bug where it was impossible to fix a saved invalid OpenAI url (#405)

## 0.10.1

- Settings: differentiate between disabled and not configured (#350)

## 0.10.0

- Breaking: use `base` and `large` model names instead of `small`/`medium`/`large` (#334)
- Breaking: remove function calling arguments from `@grafana/llm` package (#343)
- Allow customisation of mapping between abstract model and provider model, and default model (#337, #338, #340)
- Make the `model` field optional for chat completions & chat completion stream endpoints (#341)
- Don't preload the plugin to avoid slowing down Grafana load times (#339)

## 0.9.1

- Fix handling of streaming requests made via resource endpoints (#326)

## 0.9.0

- Initial backend support for abstracted models (#315)

## 0.8.6

- Fix panic with stream EOF (#308)

## 0.8.5

- Added a `displayVectorStoreOptions` flag to optionally display the vector store configs

## 0.8.1

- Add mitigation for side channel attacks

## 0.7.0

- Refactors repo into monorepo together with frontend dependencies
- Creates developer sandbox for developing frontend dependencies
- Switches CI/CD to github actions

## 0.6.4

- Fix bug where resource calls to OpenAI would fail for Grafana managed LLMs

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

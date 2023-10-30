# Changelog

## Unreleased

* Add basic auth to VectorAPI 

## 0.4.0

* Add 'Enabled' switch for vector services to configuration UI
* Added instructions for developing with example app
* Improve health check to return more granular details
* Add support for filtered vector search
* Improve vector service health check

## 0.3.0

* Add Go package providing an OpenAI client to use the LLM app from backend Go code
* Add support for Azure OpenAI. The plugin must be configured to use OpenAI and provide a link between OpenAI model names and Azure deployment names
* Return streaming errors as part of the stream, with objects like `{"error": "<error message>"}`

## 0.2.1

* Improve health check endpoint to include status of various features
* Change path handling for chat completions streams to put separate requests into separate streams. Requests can pass a UUID as the suffix of the path now, but is backwards compatible with an older version of the frontend code.

## 0.2.0

* Expose vector search API to perform semantic search against a vector database using a configurable embeddings source

## 0.1.0

* Support proxying LLM requests from Grafana to OpenAI

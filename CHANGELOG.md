# Changelog

## 0.2.1

* Change path handling for chat completions streams to put separate requests into separate streams. Requests can pass a UUID as the suffix of the path now, but is backwards compatible with an older version of the frontend code.

## 0.2.0

* Expose vector search API to perform semantic search against a vector database using a configurable embeddings source

## 0.1.0

* Support proxying LLM requests from Grafana to OpenAI

# Grafana LLM app

This is a Grafana application plugin which centralizes access to LLMs across Grafana.

It is responsible for:

- storing API keys for LLM providers
- proxying requests to LLMs with auth, so that other Grafana components need not store API keys
- providing Grafana Live streams of streaming responses from LLM providers (namely OpenAI)
- providing LLM based extensions to Grafana's extension points (e.g. 'explain this panel')

Future functionality will include:

- support for multiple LLM providers, including the ability to choose your own at runtime
- rate limiting of requests to LLMs, for cost control
- token and cost estimation
- RBAC to only allow certain users to use LLM functionality

## For users

Install and configure this plugin to enable various LLM-related functionality across Grafana.
This will include new functionality inside Grafana itself, such as explaining panels, or
in plugins, such as natural language query editors.

All LLM requests will be routed via this plugin, which ensures the correct API key is being
used and rate limited appropriately.

## For plugin developers

This plugin is not designed to be directly interacted with; instead, we will provide convenience
functions in the [`@grafana/experimental`][experimental] package which will communicate with this
plugin, if installed. See the [`grafana-llm-examples`] repository for examples of how this works.

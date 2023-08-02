# Grafana LLM examples

This is a Grafana plugin designed to showcase using the LLM functionality available in `@grafana/experimental`. Under the hood this uses the [`grafana-llm-app`] to proxy requests to the LLM provider.

## Getting started

To get started you'll need a few things:

- an OpenAI key - see the 'Getting access to LLMs' section of the [Getting Started with LLMs at Grafana doc][getting-started-doc]
  - this should be made available as the `OPENAI_API_KEY` environment variable
- Docker

Then run the following:

    docker-compose up

You should then be able to access Grafana on http://localhost:3000.

Next you'll need to build this plugin:

    npm install
    npm run dev

Head to the [LLM Examples](http://localhost:3000/a/grafana-llmexamples-app) plugin page to see some use of the LLMs in action!

## Explanation

The Grafana container in docker-compose is provisioned with the `grafana-llm-app` plugin installed (using `GF_INSTALL_PLUGINS`) and configured with your OpenAI key (using `provisioning/plugins/apps.yaml`).

This plugin makes use of the `@grafana/experimental` package to make requests to OpenAI via the `grafana-llm-app` plugin, which provides an authenticating proxy and handles streaming responses using Grafana Live.

Take a look at `src/pages/ExamplePage.tsx` to see how to make requests and use responses.

You can also toggle the value of `disabled` for the `grafana-llm-app` plugin in `provisioning/plugins/apps.yaml` to see what happens when the LLM plugin is unavailable.

## Adapting your own plugin

To add LLM functionality to your own plugin you'll need to do the following:

- add a dependency on `@grafana/experimental>=1.7.0` and make use of the `llms` module
- ensure the `grafana-llm-app` plugin is installed and configured in the Grafana instance
  - install by setting the `GF_INSTALL_PLUGINS` environment variable at startup time - see `docker-compose.yaml` for an example
  - configure either:
    - using provisioning (see provisioning/plugins/apps.yaml), or
    - within the app settings screen (e.g. http://localhost:3000/plugins/grafana-llm-app)

[getting-started-doc]: https://docs.google.com/document/d/1H9bo0QOrVbmjioTleqFsknpGszZ-py75YX2aWRcCNGE/edit#heading=h.180bjy5a5l0k
[`grafana-llm-app`]: https://github.com/grafana/grafana-llm-app

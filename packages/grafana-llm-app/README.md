# Grafana LLM App

A Grafana plugin designed to centralize access to LLMs, providing authentication, proxying, streaming, and custom extensions.
Installing this plugin will enable various pieces of LLM-based functionality throughout Grafana.

## Install the plugin on Grafana Cloud

Prerequisite:
- Any Grafana Cloud environment (including the free tier)

Steps:
1. In Grafana Cloud, click **Administration** > **Plugins and data** > **Plugins** in the side navigation menu.
1. Browse or search for the **LLM** plugin and click to open it.
1. On the **Configuration** tab, select "Enable OpenAI access via Grafana".
1. Click to permit us to share limited data with OpenAI's API (not for training, and only to provide these features).
1. Click **Save settings**.

If you prefer, you may configure your own API authentication from supported LLM providers, including OpenAI and Azure. With this option, the LLM app securely stores API keys for you.

## Install the plugin directly

To install this plugin, use the `GF_INSTALL_PLUGINS` environment variable when running Grafana:

```sh
GF_INSTALL_PLUGINS=grafana-llm-app
```

or alternatively install using the Grafana CLI:

```sh
grafana cli plugins install grafana-llm-app
```

The plugin can then be configured either in the UI or using provisioning, as shown below.

## Provisioning this plugin

To provision this plugin you should set the following environment variable when running Grafana:

```sh
OPENAI_API_KEY=sk-...
```

and use the following provisioning file (e.g. in `/etc/grafana/provisioning/plugins/grafana-llm-app`, when running in Docker):

```yaml
apiVersion: 1

apps:
  - type: 'grafana-llm-app'
    disabled: false
    jsonData:
      openAI:
        url: https://api.openai.com
    secureJsonData:
      openAIKey: $OPENAI_API_KEY
```

### Using Azure OpenAI

To provision the plugin to use Azure OpenAI, use settings similar to this:

```yaml
apiVersion: 1

apps:
  - type: 'grafana-llm-app'
    disabled: false
    jsonData:
      provider: azure
      openAI:
        url: https://<resource>.openai.azure.com
        azureModelMapping:
          - ["base", "gpt-35-turbo"]
          - ["large", "gpt-4-turbo"]
    secureJsonData:
      openAIKey: $OPENAI_API_KEY
```

where:

- `<resource>` is your Azure OpenAI resource name
- the `azureModelMapping` field contains `[model, deployment]` pairs so that features know
  which Azure deployment to use in place of each model you wish to be used.

### Using Anthropic

To provision the plugin to use Anthropic, use settings similar to this:

```yaml
apiVersion: 1

apps:
  - type: 'grafana-llm-app'
    disabled: false
    jsonData:
      provider: anthropic
    secureJsonData:
      anthropicKey: $ANTHROPIC_API_KEY
```

### Provisioning vector services

The vector services of the plugin allow some AI-based features (initially, the PromQL query advisor) to use semantic search to send better context to LLMs (and improve responses). Configuration is in roughly three parts:

- 'global' vector settings:
  - `enabled` - whether to enable or disable vector services overall
  - `model` - the name of the model to use to calculate embeddings for searches. This must match the model used when storing the data, or the embeddings will be meaningless.
- 'embedding' vector settings (`embed`):
  - `type` - the type of embedding service, either `openai` or `grafana/vectorapi` to use [Grafana's own vector API](https://github.com/grafana/vectorapi) (recommended if you're just starting out).
  - `grafanaVectorAPI`, if `type` is `grafana/vectorapi`, with keys:
    - `url` - the URL of the Grafana VectorAPI instance.
    - `authType` - the type of authentication to use, either `no-auth` or `basic-auth`.
    - `basicAuthUser` - the username to use if `authType` is `basic-auth`.
- 'store' vector settings (`store`):
  - `type` - the type of vector store to connect to. We recommend starting out with `grafana/vectorapi` to use [Grafana's own vector API](https://github.com/grafana/vectorapi) for a quick start. We also support `qdrant` for [Qdrant](https://qdrant.tech).
  - `grafanaVectorAPI`, if `type` is `grafana/vectorapi`, with keys:
    - `url` - the URL of the Grafana VectorAPI instance.
    - `authType` - the type of authentication to use, either `no-auth` or `basic-auth`.
    - `basicAuthUser` - the username to use if `authType` is `basic-auth`.
  - `qdrant`, if `type` is `qdrant`, with keys:
    - `address` - the address of the Qdrant server. Note that this uses a gRPC connection.
    - `secure` - boolean, whether to use a secure connection. If you're using a secure connection you can set the `qdrantApiKey` field in `secureJsonData` to provide an API key with each request.

#### Note
- Currently Azure OpenAI is not supported as an embedder.
- Grafana Vector API used in `embedding` and `store` can be optionally different.
- If you want to enable the PromQL Query Advisor, set up the [Grafana vector API](https://github.com/grafana/vectorapi) - we'll walk you through loading the data you need for that feature. If you're interested in building your own vector-based features on the Grafana platform, we do also support OpenAI embeddings and Qdrant.


**Grafana VectorAPI Store + Grafana VectorAPI Embedder example**

```yaml
apps:
  - type: grafana-llm-app
    jsonData:
      provider: openai
      openAI:
        url: https://api.openai.com
        organizationId: $OPENAI_ORGANIZATION_ID
      # provider: azure
      # openAI:
      #   url: https://<resource>.openai.azure.com
      #   azureModelMapping:
      #     - ["gpt-3.5-turbo", "gpt-35-turbo"]
      vector:
        enabled: true
        model: BAAI/bge-small-en-v1.5
        embed:
          type: grafana/vectorapi
          grafanaVectorAPI:
            url: <vectorapi-url> # e.g. http://localhost:8889
            authType: no-auth
            # authType: basic-auth
            # basicAuthUser: <user>
        store:
          type: grafana/vectorapi
          grafanaVectorAPI:
            url: <vectorapi-url> # e.g. http://localhost:8889
            authType: no-auth
            # authType: basic-auth
            # basicAuthUser: <user>

    secureJsonData:
      openAIKey: $OPENAI_API_KEY
      # openAIKey: $AZURE_OPENAI_API_KEY
      # vectorEmbedderBasicAuthPassword: $VECTOR_EMBEDDER_BASIC_AUTH_PASSWORD
      # vectorStoreBasicAuthPassword: $VECTOR_STORE_BASIC_AUTH_PASSWORD
```

**OpenAI Embedder + Qdrant Store example**

```yaml
apiVersion: 1

apps:
  - type: grafana-llm-app
    jsonData:
      openAI:
        provider: openai
        url: https://api.openai.com
        organizationId: $OPENAI_ORGANIZATION_ID
      vector:
        enabled: true
        model: text-embedding-ada-002
        embed:
          type: openai
        store:
          type: qdrant
          qdrant:
            address: <qdrant-grpc-address> # e.g. localhost:6334

    secureJsonData:
      openAIKey: $OPENAI_API_KEY
```

- Note: openai embed type uses the setting from `openAI` automatically

## Adding LLM features to your plugin or Grafana core

To make use of this plugin when adding LLM-based features, you can use the helper functions in the `@grafana/llm` package.

First, add the correct version of `@grafana/llm` to your dependencies in package.json:

```json
{
  "dependencies": {
    "@grafana/llm": "0.8.0"
  }
}
```

Then in your components you can use the `llm` object from `@grafana/llm` like so:

```typescript
import React, { useState } from 'react';
import { useAsync } from 'react-use';

import { llms } from '@grafana/llm';
import { PluginPage } from '@grafana/runtime';

import { Button, Input, Spinner } from '@grafana/ui';

const MyComponent = (): JSX.Element => {
  const [input, setInput] = React.useState('');
  const [message, setMessage] = React.useState('');
  const [reply, setReply] = useState('');

  const { loading, error } = useAsync(async () => {
    const enabled = await llms.openai.enabled();
    if (!enabled) {
      return false;
    }
    if (message === '') {
      return;
    }
    // Stream the completions. Each element is the next stream chunk.
    const stream = llms.openai
      .streamChatCompletions({
        model: llms.openai.Model.BASE,
        messages: [
          { role: 'system', content: 'You are a helpful assistant with deep knowledge of the Grafana, Prometheus and general observability ecosystem.' },
          { role: 'user', content: message },
        ],
      })
      .pipe(llms.openai.accumulateContent());
    // Subscribe to the stream and update the state for each returned value.
    return stream.subscribe(setReply);
  }, [message]);

  if (error) {
    // TODO: handle errors.
    return null;
  }

  return (
    <div>
      <Input value={input} onChange={(e) => setInput(e.currentTarget.value)} placeholder="Enter a message" />
      <br />
      <Button type="submit" onClick={() => setMessage(input)}>
        Submit
      </Button>
      <br />
      <div>{loading ? <Spinner /> : reply}</div>
    </div>
  );
};
```

The `messages` parameter is the same as OpenAI's concept of [`messages`](https://platform.openai.com/docs/guides/text-generation/chat-completions-api).

The `.subscribe` method can take [a few different forms](https://github.com/ReactiveX/rxjs/blob/e47129bd77a9b6f897550d3fcffb9d53e98b03a9/packages/rxjs/src/internal/Observable.ts#L23). The "callback form" shown here is the more concise form. Another form allows more specific callbacks based on conditions, e.g. `error` or `complete` which can be useful if you want to do specific UI actions like showing a loading indicator.

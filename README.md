# Grafana LLM App (Public Preview)

A Grafana plugin designed to centralize access to LLMs, providing authentication, proxying, streaming, and custom extensions.
Installing this plugin will enable various pieces of LLM-based functionality throughout Grafana.

Note: The Grafana LLM App plugin is currently in [Public preview](https://grafana.com/docs/release-life-cycle/). Grafana Labs offers support on a best-effort basis, and there might be breaking changes before the feature is generally available.

## Install the plugin on Grafana Cloud

Prerequisites:
- Any Grafana Cloud environment (including Free)
- API connection details from an account with [OpenAI](https://platform.openai.com) or [Azure OpenAI](https://oai.azure.com/)

Steps:
1. In your Grafana instance, open Administration → Plugins
1. Select "All" instead of "Installed" and search for "LLM"
1. Click "Install via grafana.com"
1. On the [LLM's plugin page](https://grafana.com/grafana/plugins/grafana-llm-app/), you should see your instance listed; click "Install plugin"
1. Return to Grafana, and search installed plugins, reloading until the LLM plugin is listed (this may take a minute or two)
1. Configuration: choose your provider (OpenAI or Azure) and fill in the fields needed
1. Save settings, then click "Enable" (upper right) to enable the plugin


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
      openAI:
        provider: azure
        url: https://<resource>.openai.azure.com
        azureModelMapping:
          - ["gpt-3.5-turbo", "gpt-35-turbo"]
    secureJsonData:
      openAIKey: $OPENAI_API_KEY
```

where:

- `<resource>` is your Azure OpenAI resource name
- the `azureModelMapping` field contains `[model, deployment]` pairs so that features know
  which Azure deployment to use in place of each model you wish to be used.


### Provisioning vector services

The vector services of the plugin allow certain AI-based features to perform semantic search to improve the responses given by LLMs. Configuration is in roughly three parts:

- 'global' vector settings:
  - `enabled` - whether to enable or disable vector services overall
  - `model` - the name of the model to use to calculate embeddings for searches. This must match the model used when storing the data, or the embeddings will be meaningless.
- 'embedding' vector settings (`embed`):
  - `type` - the type of embedding service, either `openai` or `grafana/vectorapi` to use Grafana's own vector API.
  - `openai`, if `type` is `openai`, with keys:
    - `url` - the URL to the OpenAI instance.
  - `grafanaVectorAPI` - custom settings for OpenAI embeddings, with keys:
    - `url` - the URL to the OpenAI instance.
    - `authType` - the type of authentication to use, either `no-auth` or `basic-auth`.
    - `basicAuthUser` - the username to use if `authType` is `basic-auth`.
- 'store' vector settings (`store`):
  - `type` - the type of vector store to connect to, either `qdrant` to use [Qdrant](https://qdrant.tech) or `grafana/vectorapi` to use Grafana's own vector API.
  - `qdrant`, if `type` is `qdrant`, with keys:
    - `address` - the address of the Qdrant server. Note that this uses a gRPC connection.
    - `secure` - boolean, whether to use a secure connection. If you're using a secure connection you can set the `qdrantApiKey` field in `secureJsonData` to provide an API key with each request.
  - `grafanaVectorAPI`, if `type` is `grafana/vectorapi`, with keys:
    - `url` - the URL of the Grafana VectorAPI instance.
    - `authType` - the type of authentication to use, either `no-auth` or `basic-auth`.
    - `basicAuthUser` - the username to use if `authType` is `basic-auth`.

**Qdrant example**

```yaml
apiVersion: 1

apps:
  - type: 'grafana-llm-app'
    disabled: false
    jsonData:
      vector: 
        enabled: true
        model: text-embedding-ada-002
        embed:
          type: openai
        store:
          type: qdrant
          qdrant:
            address: localhost:6334
            secure: true
    secureJsonData:
      openAIKey: $OPENAI_API_KEY
      vectorStoreBasicAuthPassword: $STORE_PASSWORD
      vectorEmbedderBasicAuthPassword: $EMBEDDER_PASSWORD 
```

## Adding LLM features to your plugin or Grafana core

To make use of this plugin when adding LLM-based features, you can use the helper functions in the `@grafana/experimental` package.

First, add the correct version of `@grafana/experimental` to your dependencies in package.json:

```json
{
  "dependencies": {
    "@grafana/experimental": "1.7.0"
  }
}
```

Then in your components you can use the `llm` object from `@grafana/experimental` like so:

```typescript
import React, { useState } from 'react';
import { useAsync } from 'react-use';

import { llms } from '@grafana/experimental';
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
        model: 'gpt-3.5-turbo',
        messages: [
          { role: 'system', content: 'You are a cynical assistant.' },
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

## Developing this plugin

### Backend

1. Update [Grafana plugin SDK for Go](https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/) dependency to the latest minor version:

   ```bash
   go get -u github.com/grafana/grafana-plugin-sdk-go
   go mod tidy
   ```

2. Build backend plugin binaries for Linux, Windows and Darwin:

   ```bash
   mage -v
   ```

3. List all available Mage targets for additional commands:

   ```bash
   mage -l
   ```

### Frontend

1. Install dependencies

   ```bash
   npm run install
   ```

2. Build plugin in development mode and run in watch mode

   ```bash
   npm run dev
   ```

3. Build plugin in production mode

   ```bash
   npm run build
   ```

4. Run the tests (using Jest)

   ```bash
   # Runs the tests and watches for changes, requires git init first
   npm run test

   # Exits after running all the tests
   npm run test:ci
   ```

5. Spin up a Grafana instance and run the plugin inside it (using Docker)

   ```bash
   npm run server
   ```

6. Run the E2E tests (using Cypress)

   ```bash
   # Spins up a Grafana instance first that we tests against
   npm run server

   # Starts the tests
   npm run e2e
   ```

7. Run the linter

   ```bash
   npm run lint

   # or

   npm run lint:fix
   ```

### Developing with the Example App
The LLM example app can be a quick way to test out changes to the LLM plugin.

To use the example app in conjunction with the LLM plugin:
1. Clone the [llm example app](https://github.com/grafana/grafana-llmexamples-app/tree/main)
2. Update the following fields in `docker-compose.yaml` in the llm example app
- comment out # GF_INSTALL_PLUGINS: grafana-llm-app
- Add the following volume:
```
<some-parent-path>/grafana-llm-app/dist:/var/lib/grafana/plugins/grafana-llm-app
```
3. Follow the instructions in the llm example app to run the app

## Release process

### Plugin Release
- Bump version in package.json (e.g., 0.2.0 to 0.2.1)
- Add notes to changelog describing changes since last release
- Merge PR for a branch containing those changes into main
- Go to drone [here](https://drone.grafana.net/grafana/grafana-llm-app) and identify the build corresponding to the merge into main
- Promote to target 'publish'

### llmclient Release
- Push a new tag to the repo (e.g., `git tag -a llmclient/v0.X.X -m "llmclient v0.X.X release"`)

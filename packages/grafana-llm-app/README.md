# Grafana LLM App (Public Preview)

A Grafana plugin designed to centralize access to LLMs, providing authentication, proxying, streaming, and custom extensions.
Installing this plugin will enable various pieces of LLM-based functionality throughout Grafana.

Note: The Grafana LLM App plugin is currently in [Public preview](https://grafana.com/docs/release-life-cycle/). Grafana Labs offers support on a best-effort basis, and there might be breaking changes before the feature is generally available.

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

### Adding Support for Other LLM Vendors

The Grafana LLM App is designed to be extensible, allowing you to add support for additional LLM providers beyond the built-in ones (OpenAI, Azure OpenAI, Anthropic). This section provides guidance on how to implement a new provider.

> **Tip:** For a complete example of adding a new provider, see [PR #566](https://github.com/grafana/grafana-llm-app/pull/566) which added Anthropic support. This PR demonstrates all the necessary changes for both backend and frontend components.

#### Backend Implementation Steps

1. **Create Provider Settings**:
   - Add a new settings struct in `packages/grafana-llm-app/pkg/plugin/settings.go` for your provider
   - Example for a new provider called "MyLLM":
     ```go
     // MyLLMSettings contains MyLLM-specific settings
     type MyLLMSettings struct {
         // The URL to the provider's API
         URL string `json:"url"`
         
         // apiKey is the provider-specific API key needed to authenticate requests
         // Stored securely.
         apiKey string
     }
     ```
   - Update the `Settings` struct to include your new provider settings
   - Add a new provider type constant in the `ProviderType` enum

2. **Create Provider Implementation**:
   - Create a new file `packages/grafana-llm-app/pkg/plugin/myllm_provider.go`
   - Implement the `LLMProvider` interface defined in `llm_provider.go`
   - Required methods include:
     - `Models()` - Returns available models
     - `ChatCompletion()` - Handles chat completion requests
     - `ChatCompletionStream()` - Handles streaming chat completion requests
     - `ListAssistants()` - Handles assistant-related functionality (can return an error if not supported)


3. **Register the Provider**:
   - Update the `createProvider` function in `packages/grafana-llm-app/pkg/plugin/provider.go` to include your new provider
   - Add a case for your provider type that calls your provider's constructor

4. **Update Configuration Logic**:
   - Update the `Configured()` method in `settings.go` to handle your provider type
   - Ensure the settings validation logic works correctly for your provider

5. **Add Model Mapping Support**:
   - Implement a `toMyLLM()` method for the `Model` type in `llm_provider.go` to map abstract models to your provider's models

#### Frontend Implementation Steps

1. **Update TypeScript Types**:
   - Add your provider to the `ProviderType` enum in `packages/grafana-llm-app/src/components/AppConfig/AppConfig.tsx`:
     ```typescript
     export type ProviderType = 'openai' | 'azure' | 'grafana' | 'test' | 'custom' | 'anthropic' | 'myllm';
     ```
   - Add your provider to the `LLMOptions` type in `packages/grafana-llm-app/src/components/AppConfig/LLMConfig.tsx`:
     ```typescript
     export type LLMOptions = 'grafana-provided' | 'openai' | 'test' | 'disabled' | 'unconfigured' | 'custom' | 'anthropic' | 'myllm';
     ```
   - Add your provider settings interface in a new file or in `AppConfig.tsx`:
     ```typescript
     export interface MyLLMSettings {
       // The URL to reach your provider's API
       url?: string;
       // If the LLM features have been explicitly disabled
       disabled?: boolean;
     }
     ```
   - Update the `AppPluginSettings` interface to include your provider settings:
     ```typescript
     export interface AppPluginSettings {
       // existing fields...
       myllm?: MyLLMSettings;
     }
     ```
   - Update the `Secrets` type to include your provider's API key:
     ```typescript
     export type Secrets = {
       // existing fields...
       myllmKey?: string;
     };
     ```

2. **Create Provider Configuration Component**:
   - Create a new file `packages/grafana-llm-app/src/components/AppConfig/MyLLMConfig.tsx` for your provider's configuration UI:
     ```typescript
     import React, { ChangeEvent } from 'react';
     import { Field, FieldSet, Input, SecretInput, useStyles2 } from '@grafana/ui';
     import { testIds } from 'components/testIds';
     import { getStyles, Secrets, SecretsSet } from './AppConfig';

     const MYLLM_API_URL = 'https://api.myllm.com';

     export interface MyLLMSettings {
       // The URL to reach your provider's API
       url?: string;
       // If the LLM features have been explicitly disabled
       disabled?: boolean;
     }

     export function MyLLMConfig({
       settings,
       secrets,
       secretsSet,
       onChange,
       onChangeSecrets,
     }: {
       settings: MyLLMSettings;
       onChange: (settings: MyLLMSettings) => void;
       secrets: Secrets;
       secretsSet: SecretsSet;
       onChangeSecrets: (secrets: Secrets) => void;
     }) {
       const s = useStyles2(getStyles);
       
       const onChangeField = (event: ChangeEvent<HTMLInputElement>) => {
         onChange({
           ...settings,
           [event.currentTarget.name]:
             event.currentTarget.type === 'checkbox' ? event.currentTarget.checked : event.currentTarget.value.trim(),
         });
       };

       return (
         <FieldSet>
           <Field label="API URL" className={s.marginTop}>
             <Input
               width={60}
               name="url"
               data-testid={testIds.appConfig.myllmUrl}
               value={MYLLM_API_URL}
               placeholder={MYLLM_API_URL}
               onChange={onChangeField}
               disabled={true}
             />
           </Field>

           <Field label="API Key">
             <SecretInput
               width={60}
               data-testid={testIds.appConfig.myllmKey}
               name="myllmKey"
               value={secrets.myllmKey}
               isConfigured={secretsSet.myllmKey ?? false}
               placeholder="your-api-key-format"
               onChange={(e) => onChangeSecrets({ ...secrets, myllmKey: e.currentTarget.value })}
               onReset={() => onChangeSecrets({ ...secrets, myllmKey: '' })}
             />
           </Field>
         </FieldSet>
       );
     }
     ```

3. **Create Provider Logo Component** (optional):
   - Create a new file `packages/grafana-llm-app/src/components/AppConfig/MyLLMLogo.tsx` for your provider's logo:
     ```typescript
     import React from 'react';

     export function MyLLMLogo({ width, height }: { width: number; height: number }) {
       return (
         <svg width={width} height={height} viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
           {/* Your provider's SVG logo */}
         </svg>
       );
     }
     ```

4. **Update Test IDs**:
   - Add test IDs for your provider in `packages/grafana-llm-app/src/components/testIds.ts`:
     ```typescript
     export const testIds = {
       appConfig: {
         // existing fields...
         myllmKey: 'data-testid ac-myllm-api-key',
         myllmUrl: 'data-testid ac-myllm-api-url',
       },
     };
     ```

5. **Update LLMConfig Component**:
   - Import your provider components in `LLMConfig.tsx`:
     ```typescript
     import { MyLLMConfig } from './MyLLMConfig';
     import { MyLLMLogo } from './MyLLMLogo';
     ```
   - Update the `getLLMOptionFromSettings` function to handle your provider:
     ```typescript
     function getLLMOptionFromSettings(settings: AppPluginSettings): LLMOptions {
       // existing code...
       switch (provider) {
         // existing cases...
         case 'myllm':
           return 'myllm';
         default:
           return 'unconfigured';
       }
     }
     ```
   - Add a selection handler for your provider:
     ```typescript
     const selectMyLLMProvider = () => {
       if (llmOption !== 'myllm') {
         onChange({ ...settings, provider: 'myllm', disabled: false });
       }
     };
     ```
   - Add a UI card for your provider:
     ```tsx
     <div onClick={selectMyLLMProvider}>
       <Card isSelected={llmOption === 'myllm'} className={s.cardWithoutBottomMargin}>
         <Card.Heading>Use MyLLM API</Card.Heading>
         <Card.Description>
           Enable LLM features in Grafana using MyLLM's models
           {llmOption === 'myllm' && (
             <MyLLMConfig
               settings={settings.myllm ?? {}}
               onChange={(myllm) => onChange({ ...settings, myllm })}
               secrets={secrets}
               secretsSet={secretsSet}
               onChangeSecrets={onChangeSecrets}
             />
           )}
         </Card.Description>
         <Card.Figure>
           <MyLLMLogo width={20} height={20} />
         </Card.Figure>
       </Card>
     </div>
     ```

6. **Update Tests**:
   - Add tests for your provider in `packages/grafana-llm-app/src/components/AppConfig/AppConfig.test.tsx`:
     ```typescript
     test('renders MyLLM configuration when provider is MyLLM', () => {
       const plugin = { meta: { ...props.plugin.meta, enabled: false, jsonData: { provider: 'myllm' } } };
       render(<AppConfig {...props} plugin={plugin as any} />);
       
       expect(screen.queryByTestId(testIds.appConfig.myllmUrl)).toBeInTheDocument();
       expect(screen.queryByTestId(testIds.appConfig.myllmKey)).toBeInTheDocument();
     });
     ```

#### Example Implementation Pattern

Here's a simplified example based on the Anthropic provider implementation:

```go
// myllm_provider.go
package plugin

import (
    "context"
    // Import your provider's SDK or HTTP client libraries
)

type myLLMProvider struct {
    settings MyLLMSettings
    models   *ModelSettings
    client   *YourProviderClient
}

func NewMyLLMProvider(settings MyLLMSettings, models *ModelSettings) (LLMProvider, error) {
    // Initialize your provider's client with the settings
    client := initializeYourProviderClient(settings)
    
    // Define default model mappings if none are provided
    defaultModels := &ModelSettings{
        Default: ModelBase,
        Mapping: map[Model]string{
            ModelBase:  "your-base-model",
            ModelLarge: "your-large-model",
        },
    }
    
    if models == nil {
        models = defaultModels
    }
    
    return &myLLMProvider{
        settings: settings,
        models:   models,
        client:   client,
    }, nil
}

// Implement all required interface methods

func (p *myLLMProvider) Models(ctx context.Context) (ModelResponse, error) {
    // Return available models
}

func (p *myLLMProvider) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
    // Convert from the standard format to your provider's format
    // Make the API call
    // Convert the response back to the standard format
}

func (p *myLLMProvider) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
    // Implement streaming support
}

func (p *myLLMProvider) ListAssistants(ctx context.Context, limit *int, order *string, after *string, before *string) (openai.AssistantsList, error) {
    // Implement or return an error if not supported
    return openai.AssistantsList{}, fmt.Errorf("myllm does not support assistants")
}
```

#### Testing Your Provider

1. Implement unit tests for your provider in a `myllm_provider_test.go` file
2. Test the provider with real API calls using the developer sandbox
3. Ensure all existing functionality works with your new provider

#### Provisioning Your Provider

Once implemented, users can provision your provider using configuration like:

```yaml
apiVersion: 1

apps:
  - type: 'grafana-llm-app'
    disabled: false
    jsonData:
      provider: myllm
      myllm:
        url: https://api.myllm.com
    secureJsonData:
      myllmKey: $MYLLM_API_KEY
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

# Contributing to Grafana LLM App

## Adding Support for Other LLM Vendors

The Grafana LLM App is designed to be extensible, allowing you to add support for additional LLM providers beyond the built-in ones (OpenAI, Azure OpenAI, Anthropic). This section provides guidance on how to implement a new provider.

> **Tip:** For a complete example of adding a new provider, see [PR #566](https://github.com/grafana/grafana-llm-app/pull/566) which added Anthropic support. This PR demonstrates all the necessary changes for both backend and frontend components.

### Backend Implementation Steps

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

### Frontend Implementation Steps

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

### Example Implementation Pattern

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

### Testing Your Provider

1. Implement unit tests for your provider in a `myllm_provider_test.go` file
2. Test the provider with real API calls using the developer sandbox
3. Ensure all existing functionality works with your new provider

### Provisioning Your Provider

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
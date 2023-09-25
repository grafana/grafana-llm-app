# Grafana LLM App (Experimental)

A Grafana plugin designed to centralize access to LLMs, providing authentication, rate limiting, and more.
Installing this plugin will enable various pieces of LLM-based functionality throughout Grafana.

Note: This plugin is **experimental**, and may change significantly between
versions, or be deprecated completely in favor of a different approach based on
user feedback.

## Installing this plugin

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
import { scan } from 'rxjs/operators';

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
    const stream = llms.openai.streamChatCompletions({
      model: 'gpt-3.5-turbo',
      messages: [
        { role: 'system', content: 'You are a cynical assistant.' },
        { role: 'user', content: message },
      ],
    }).pipe(
      // Accumulate the stream chunks into a single string.
      scan((acc, delta) => acc + delta, '')
    );
    // Subscribe to the stream and update the state for each returned value.
    return stream.subscribe(setReply);
  }, [message]);

  if (error) {
    // TODO: handle errors.
    return null;
  }

  return (
    <div>
      <Input
        value={input}
        onChange={(e) => setInput(e.currentTarget.value)}
        placeholder="Enter a message"
      />
      <br />
      <Button type="submit" onClick={() => setMessage(input)}>Submit</Button>
      <br />
      <div>{loading ? <Spinner /> : reply}</div>
    </div>
  );
}
```

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

## Release process
- Bump version in package.json (e.g., 0.2.0 to 0.2.1)
- Add notes to changelog describing changes since last r elease
- Merge PR for a branch containing those changes into main
- Go to drone [here](https://drone.grafana.net/grafana/grafana-llm-app) and identify the build corresponding to the merge into main
- Promote to target 'publish'
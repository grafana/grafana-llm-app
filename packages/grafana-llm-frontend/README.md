# Frontend functions for LLM interaction

This is a collection of convenience functions and components to make interacting with LLM functionality in Grafana easier.

First, add the latest version of `@grafana/llm` to your dependencies in package.json:

```json
{
  "dependencies": {
    "@grafana/llm": "0.22.0"
  }
}
```

**Note:** If you're writing tests that import from `@grafana/llm`, you may need to configure Jest to handle ES modules. See the Jest Configuration section below for details.

## Jest Configuration

### ESM errors with Jest

When writing tests that import from `@grafana/llm`, you may encounter Jest errors like `SyntaxError: Cannot use import statement outside a module`. This happens because `@grafana/llm` uses ES module dependencies that Jest needs to transform.

If you're using Grafana's plugin scaffolding, extend your Jest configuration to include the additional ES modules. For convenience, `@grafana/llm` exports the required module list:

```javascript
// jest.config.js
const { grafanaESModules, nodeModulesToTransform } = require('./.config/jest/utils');
const { grafanaLLMESModules } = require('@grafana/llm/jest');

module.exports = {
  // Jest configuration provided by Grafana scaffolding
  ...require('./.config/jest.config'),
  transformIgnorePatterns: [nodeModulesToTransform([...grafanaESModules, ...grafanaLLMESModules])],
};
```

### MCP Functionality

If you're testing code that uses MCP (Model Context Protocol) features, add the `TransformStream` polyfill to your `jest-setup.js`:

```javascript
// jest-setup.js
// Jest setup provided by Grafana scaffolding
import './.config/jest-setup';

// Add this import and global for MCP functionality
import { TransformStream } from 'node:stream/web';
import { TextEncoder } from 'util';

// TextEncoder may already be present in your setup
global.TextEncoder = TextEncoder;
// Add this line for MCP TransformStream support
global.TransformStream = TransformStream;
```

## Usage

### Handle provider configuration states

Before rendering LLM-powered features, use `llm.health()` to distinguish between an available provider, an unconfigured provider, and a provider that an administrator has configured but that is currently unavailable:

```typescript
import { llm } from '@grafana/llm';

const health = await llm.health();

if (health.ok) {
  // The provider is configured and healthy. Show LLM-powered features.
} else if (!health.configured) {
  // No provider has been configured. Prompt an administrator to configure one.
} else {
  // The provider was configured, but is disabled or unhealthy. Hide LLM-powered features.
}
```

Use `llm.enabled()` when you only need a boolean availability check. It returns `true` when the provider is both configured and healthy, but it does not distinguish an unconfigured provider from one that is disabled or unhealthy.

Then in your components you can use the `llm` object from `@grafana/llm` like so:

```typescript
import React, { useState } from 'react';
import { useAsync } from 'react-use';
import { scan } from 'rxjs/operators';

import { openai } from '@grafana/llm';
import { PluginPage } from '@grafana/runtime';

import { Button, Input, Spinner } from '@grafana/ui';

const MyComponent = (): JSX.Element => {
  const [input, setInput] = React.useState('');
  const [message, setMessage] = React.useState('');
  const [reply, setReply] = useState('');

  const { loading, error } = useAsync(async () => {
    const enabled = await openai.enabled();
    if (!enabled) {
      return false;
    }
    if (message === '') {
      return;
    }
    // Stream the completions. Each element is the next stream chunk.
    const stream = openai
      .streamChatCompletions({
        // model: openai.Model.LARGE, // defaults to BASE, use larger model for longer context and complex tasks
        messages: [
          { role: 'system', content: 'You are a helpful assistant with deep knowledge of the Grafana, Prometheus and general observability ecosystem.' },
          { role: 'user', content: message },
        ],
      })
      .pipe(
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

# Grafana LLM app (public preview)

This Grafana application plugin centralizes access to LLMs across Grafana.

It is responsible for:

- storing API keys for LLM providers
- proxying requests to LLMs with auth, so that other Grafana components need not store API keys
- providing Grafana Live streams of streaming responses from LLM providers (namely OpenAI)
- providing LLM based extensions to Grafana's extension points (e.g. 'explain this panel')

Future functionality will include:

- support for additional LLM providers, including the ability to choose your own at runtime
- rate limiting of requests to LLMs, for cost control
- token and cost estimation
- RBAC to only allow certain users to use LLM functionality

Note: The Grafana LLM App plugin is currently in [Public preview](https://grafana.com/docs/release-life-cycle/). Grafana Labs offers support on a best-effort basis, and there might be breaking changes before the feature is generally available.

## For users

Install and configure this plugin to enable various LLM-related functionality across Grafana.
This includes new functionality inside Grafana itself, such as explaining panels, or in plugins,
such as natural language query editors.

All LLM requests will be routed via this plugin, which ensures the correct API key is being
used and requests are routed appropriately.

## For plugin developers

This plugin is not designed to be directly interacted with; instead, use the convenience functions
in the [`@grafana/experimental`](https://www.npmjs.com/package/@grafana/experimental)
package which will communicate with this plugin, if installed.

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

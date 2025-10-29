# Grafana LLM plugin

This Grafana application plugin centralizes access to LLMs across Grafana.

It is responsible for:

- storing API keys for LLM providers
- proxying requests to LLMs with auth, so that other Grafana components need not store API keys
- providing Grafana Live streams of streaming responses from LLM providers (namely OpenAI)
- providing LLM based extensions to Grafana's extension points (e.g. 'explain this panel')
  <br/><br/>

Future functionality will include:

- support for more LLM providers, including the ability to choose your own at runtime
- rate limiting of requests to LLMs, for cost control
- token and cost estimation
- RBAC to only allow certain users to use LLM functionality
  <br/><br/>

## For users

Grafana Cloud: the LLM app plugin is installed for everyone, but LLM features are disabled by default. To enable LLM features, select "Enable OpenAI access via Grafana" in plugin configuration.

OSS or Enterprise: install and configure this plugin with your OpenAI-compatible API key to enable LLM-powered features across Grafana.

For more detailed setup instructions see [LLM plugin](https://grafana.com/docs/grafana-cloud/machine-learning/llm/) section under machine learning.

This includes new functionality inside Grafana itself, such as explaining panels, or in plugins,
such as automated incident summaries, AI assistants for flame graphs and Sift error logs, and more.

All LLM requests will be routed via this plugin, which ensures the correct API key is being
used and requests are routed appropriately.

## For plugin developers

This plugin is not designed to be directly interacted with; instead, use the convenience functions
in the [`@grafana/llm`](https://www.npmjs.com/package/@grafana/llm)
package which will communicate with this plugin, if installed.

Working examples can be found in the ['@grafana/llm README'](https://github.com/grafana/grafana-llm-app/tree/main/packages/grafana-llm-frontend/README.md) and in the DevSandbox.tsx class.

First, add the latest version of `@grafana/llm` to your dependencies in package.json:

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
import { scan } from 'rxjs/operators';

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
        model: llms.openai.Model.BASE
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

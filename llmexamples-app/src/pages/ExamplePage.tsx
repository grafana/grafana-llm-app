import React from 'react';
import { testIds } from '../components/testIds';
import { PluginPage } from '@grafana/runtime';

import { useLLM } from 'hooks/useLLM';
import LLMChat from 'components/LLMChat';

export function ExamplePage() {

  const { llm, isLoading } = useLLM();
  const [lastMessage, setLastMessage] = React.useState('');

  if (isLoading) {
    return <div>Loading...</div>;
  }
  if (!llm) {
    return <div>LLMs are not available.</div>;
  }

  return (
    <PluginPage>
      <div data-testid={testIds.pageTwo.container}>
        {llm.models.filter((_, i) => i < 5).map((model) => (
          <p key={model.id}>{model.id}</p>
        ))}
      </div>

      <LLMChat
        modelId="gpt-3.5-turbo"
        systemPrompt="You are a cynical assistant."
        callback={(text) => {
          console.log(text);
          setLastMessage(text);
        }}
      />
      <div>
        {lastMessage}
      </div>
    </PluginPage>
  );
}


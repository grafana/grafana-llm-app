import React, { useState, useEffect } from 'react';

import { openai } from '@sandersaarond/llm';
import { PluginPage } from '@grafana/runtime';
import { Button, Input } from '@grafana/ui';

export function ExamplePage() {

  const { setMessages, reply, value, error, streamStatus } = openai.useOpenAIStream('gpt-3.5-turbo', 1);
  // The current input value.
  const [input, setInput] = useState('');
  // The final message to send to the LLM, updated when the button is clicked.

  const sendMessages = () => {
    setMessages([{
      role: "system",
      content: "You are an assistant, but you aren't happy about it."
    }, {
      role: "user",
      content: input,
    }])
  }

  useEffect(() => {
    console.log("Error");
    console.log(error);
  }, [error])

  return (
    <PluginPage>
      {value?.enabled ? (
        <>
          <Input
            value={input}
            onChange={(e) => setInput(e.currentTarget.value)}
            placeholder="Enter a message"
          />
          <br />
          <Button type="submit" onClick={sendMessages}>Submit Stream</Button>
          <br />
          <div>{reply}</div>
          <div>{streamStatus === openai.StreamStatus.GENERATING ? "Response is started" : "Response is not started"}</div>
          <div>{streamStatus === openai.StreamStatus.COMPLETED ? "Response is finished" : "Response is not finished"}</div>
        </>
      ) : (
        <div>LLM plugin not enabled.</div>
      )}
    </PluginPage>
  );
}

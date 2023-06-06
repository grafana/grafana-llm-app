import React from "react";

import { Button, Field, Form, Input } from "@grafana/ui";

import { useLLM } from "hooks/useLLM";

interface Props {
  modelId: string;
  systemPrompt: string;
  callback: (text: string) => void;
}

// A simple chat component that takes a modelId, systemPrompt, and callback function.
const LLMChat = ({ modelId, systemPrompt, callback }: Props) => {
  const { llm, isLoading } = useLLM();

  if (isLoading) {
    return <div>Loading LLMs...</div>;
  }
  if (!llm) {
    return <div>LLMs are not available.</div>;
  }
  const session = llm.beginSession(modelId, systemPrompt);

  return (
    <div>
      <h1>LLM chat!</h1>
      <div>
        <Form
          onSubmit={async ({ message }) => {
            const returnedMessage = await session.sendMessage(message);
            callback(returnedMessage);
          }}
        >
          {({ register }) => {
            return (
              <>
                <Field>
                  <Input {...register("message")} />
                </Field>
                <Button type="submit">Submit</Button>
              </>
            )
          }}
        </Form>
      </div>
    </div>
  )
}

export default LLMChat;

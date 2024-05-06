import React, { useState } from "react";
import { Button, FieldSet, Input, Modal, Spinner } from "@grafana/ui";
import { useAsync } from "react-use";
import { finalize } from "rxjs";
import { openai } from "@grafana/llm";

const BasicChatTest = () => {
  // The current input value.
  const [input, setInput] = useState('');
  // The final message to send to the LLM, updated when the button is clicked.
  const [message, setMessage] = useState('');
  // The latest reply from the LLM.
  const [reply, setReply] = useState('');

  const [useStream, setUseStream] = useState(false);

  const [started, setStarted] = useState(false);
  const [finished, setFinished] = useState(true);

  const { loading, error, value } = useAsync(async () => {
    // Check if the LLM plugin is enabled and configured.
    // If not, we won't be able to make requests, so return early.
    console.log("Logging works");
    const openAIHealthDetails = await openai.enabled();
    console.log("openAIHealthDetails: ", openAIHealthDetails);
    const enabled = openAIHealthDetails;
    console.log("enabled: ", enabled);
    if (!enabled) {
      return { enabled };
    }
    if (message === '') {
      return { enabled };
    }

    setStarted(true);
    setFinished(false);
    if (!useStream) {
      // Make a single request to the LLM.
      const response = await openai.chatCompletions({
        model: openai.Model.SMALL,
        messages: [
          { role: 'system', content: 'You are a cynical assistant.' },
          { role: 'user', content: message },
        ],
      });
      setReply(response.choices[0].message.content);
      setStarted(false);
      setFinished(true);
      return { enabled, response };
    } else {
      // Stream the completions. Each element is the next stream chunk.
      const stream = openai.streamChatCompletions({
        model: openai.Model.SMALL,
        messages: [
          { role: 'system', content: 'You are a cynical assistant.' },
          { role: 'user', content: message },
        ],
      }).pipe(
        // Accumulate the stream content into a stream of strings, where each
        // element contains the accumulated message so far.
        openai.accumulateContent(),
        // The stream is just a regular Observable, so we can use standard rxjs
        // functionality to update state, e.g. recording when the stream
        // has completed.
        // The operator decision tree on the rxjs website is a useful resource:
        // https://rxjs.dev/operator-decision-tree.
        finalize(() => {
          setStarted(false);
          setFinished(true);
        })
      );
      // Subscribe to the stream and update the state for each returned value.
      return {
        enabled,
        stream: stream.subscribe(setReply),
      };
    }
  }, [message]);

  if (error) {
    // TODO: handle errors.
    return <div>error</div>;
  }

  return (
    <div>
      {value?.enabled ? (
        <>
          <Input
            value={input}
            onChange={(e) => setInput(e.currentTarget.value)}
            placeholder="Enter a message"
          />
          <br />
          <Button type="submit" onClick={() => {setMessage(input); setUseStream(true);}}>Submit Stream</Button>
          <Button type="submit" onClick={() => {setMessage(input); setUseStream(false);}}>Submit Request</Button>
          <br />
          <div>{loading ? <Spinner /> : reply}</div>
          <div>{started ? "Response is started" : "Response is not started"}</div>
          <div>{finished ? "Response is finished" : "Response is not finished"}</div>
        </>
      ) : (
        <div>LLM plugin not enabled.</div>
      )}
    </div>
  );
};


export const DevSandbox = () => {
    const [modalIsOpen, setModalIsOpen] = useState(false);
    const closeModal = () => {
      setModalIsOpen(false);
    }

    return (
        <FieldSet label="Development Sandbox">
            <Button onClick={() => setModalIsOpen(true)}>Open development sandbox</Button>
                <Modal title="Development Sandbox" isOpen={modalIsOpen} onDismiss={closeModal}>
                  <BasicChatTest />
                    <Button variant="primary" onClick={closeModal}>
                    Close
                    </Button>
                </Modal>
        </FieldSet>
    );
};

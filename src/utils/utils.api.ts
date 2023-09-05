import { isLiveChannelMessageEvent, LiveChannelAddress, LiveChannelMessageEvent, LiveChannelScope } from "@grafana/data";
import { getBackendSrv, getGrafanaLiveSrv } from "@grafana/runtime";

import { Observable } from "rxjs";
import { filter, map, takeWhile } from "rxjs/operators";

import pluginJson from '../plugin.json';

export interface Message {
  role: string,
  content: string,
}

export interface ChatCompletionsProps {
  model: string;
  messages: Message[];
}

interface Choice {
  message: {
    content: string;
  }
}

interface ChatCompletionsResponse<T = Choice> {
  choices: T[];
}

export const chatCompletions = async ({ model, messages }: ChatCompletionsProps): Promise<string> => {
  const response = await getBackendSrv().post<ChatCompletionsResponse>('/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions', {
    model,
    messages,
  }, { headers: { 'Content-Type': 'application/json' } });
  return response.choices[0].message.content;
}

interface ContentMessage {
  content: string;
}

interface RoleMessage {
  role: string;
}

interface DoneMessage {
  done: boolean;
}

type ChatCompletionsDelta = ContentMessage | RoleMessage | DoneMessage;

interface ChatCompletionsChunk {
  delta: ChatCompletionsDelta;
}

const isContentMessage = (message: any): message is ContentMessage => {
  return message.content !== undefined;
}

const isDoneMessage = (message: any): message is DoneMessage => {
  return message.done !== undefined;
}

export const streamChatCompletions = ({ model, messages }: ChatCompletionsProps): Observable<string> => {
  const channel: LiveChannelAddress = {
    scope: LiveChannelScope.Plugin,
    namespace: pluginJson.id,
    path: `/openai/v1/chat/completions`,
    data: {
      model,
      messages,
    },
  };
  const responses = getGrafanaLiveSrv()
    .getStream(channel)
    .pipe(filter((event) => isLiveChannelMessageEvent(event))) as Observable<LiveChannelMessageEvent<ChatCompletionsResponse<ChatCompletionsChunk>>>
  return responses.pipe(
    takeWhile((event) => !isDoneMessage(event.message.choices[0].delta)),
    map((event) => event.message.choices[0].delta),
    filter((delta) => isContentMessage(delta)),
    map((delta) => (delta as ContentMessage).content),
  );
}

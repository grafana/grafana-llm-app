import { isLiveChannelMessageEvent, LiveChannelAddress, LiveChannelMessageEvent, LiveChannelScope } from "@grafana/data";
import { getBackendSrv, getGrafanaLiveSrv } from "@grafana/runtime";

import { Observable } from "rxjs";
import { filter, map, takeWhile } from "rxjs/operators";

import pluginJson from '../plugin.json';

export interface ChatCompletionsProps {
  model: string;
  systemPrompt: string;
  userPrompt: string;
}

interface Choice {
  message: {
    content: string;
  }
}

interface ChatCompletionsResponse<T = Choice> {
  choices: T[];
}

export const chatCompletions = async ({ model, systemPrompt, userPrompt }: ChatCompletionsProps): Promise<string> => {
  const response = await getBackendSrv().post<ChatCompletionsResponse>('/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions', {
    model,
    messages: [
      { role: 'system', content: systemPrompt },
      { role: 'user', content: userPrompt },
    ],
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

export const streamChatCompletions = ({ model, systemPrompt, userPrompt }: ChatCompletionsProps): Observable<string> => {
  const channel: LiveChannelAddress = {
    scope: LiveChannelScope.Plugin,
    namespace: pluginJson.id,
    path: `/v1/chat/completions`,
    data: {
      model,
      messages: [
        { role: 'system', content: systemPrompt },
        { role: 'user', content: userPrompt },
      ],
    },
  };
  const messages = getGrafanaLiveSrv()
    .getStream(channel)
    .pipe(filter((event) => isLiveChannelMessageEvent(event))) as Observable<LiveChannelMessageEvent<ChatCompletionsResponse<ChatCompletionsChunk>>>
  return messages.pipe(
    takeWhile((event) => !isDoneMessage(event.message.choices[0].delta)),
    map((event) => event.message.choices[0].delta),
    filter((delta) => isContentMessage(delta)),
    map((delta) => (delta as ContentMessage).content),
  );
}

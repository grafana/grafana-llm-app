import { getBackendSrv } from "@grafana/runtime";

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

interface ChatCompletionsResponse {
  choices: Choice[];
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

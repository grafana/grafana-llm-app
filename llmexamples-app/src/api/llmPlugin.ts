import { getBackendSrv } from '@grafana/runtime';

type SystemMessage = 'system';
type AssistantMessage = 'assistant';
type UserMessage = 'user';

export interface Message {
  role: SystemMessage | AssistantMessage | UserMessage;
  content: string;
}

// A stateful chat session with the LLM API.
//
// Really this just stores all the chat history so far,
// and each call to the chat completions API includes all
// previous messages.
export interface Session {
  modelId: string;
  systemPrompt: string;
  messages: Message[];
  sendMessage(message: string): Promise<string>;
}

class SessionImpl implements Session {
  modelId: string;
  systemPrompt: string;
  messages: Message[] = [];

  llm: LLMSrv;

  constructor(modelId: string, systemPrompt: string, llm: LLMSrv) {
    this.llm = llm;
    this.modelId = modelId;
    this.systemPrompt = systemPrompt;
    this.messages = [
      { role: 'system', content: systemPrompt },
    ];
  }

  async sendMessage(message: string): Promise<string> {
    this.messages.push({ role: 'user', content: message });
    const response = await this.llm.chat(this.modelId, this.messages);
    this.messages.push({ role: 'system', content: response });
    return response;
  }
}

interface Response<T> {
  data: T;
}

interface Model {
  id: string;
}

// The LLM API.
//
// Plugins can use this to interact with the LLM Grafana plugin.
export interface LLMSrv {
  models: Model[];
  getModels(): Promise<Model[]>;
  beginSession(modelId: string, systemPrompt: string): Session;
  chat(model: string, messages: Message[]): Promise<string>;
}

// The LLM API implementation that uses the `grafana-llm-app`
// plugin to route all requests to an LLM.
class LLMPluginImpl implements LLMSrv {
  models: Model[];

  constructor(private plugin: Plugin) {
    this.plugin = plugin;
    this.models = [];
  }

  async init(): Promise<void> {
    this.models = await this.getModels();
  }

  async getModels(): Promise<Model[]> {
    const response: Response<Model[]> = await getBackendSrv().get('/api/plugins/grafana-llm-app/resources/openai/v1/models');
    return response.data;
  }

  async chat(model: string, messages: Message[]): Promise<string> {
    const response = await getBackendSrv().post('/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions', {
      model,
      messages,
    }, { headers: { 'Content-Type': 'application/json' } });
    return response.choices[0].message.content;
  }

  beginSession(modelId: string, systemPrompt: string): Session {
    return new SessionImpl(modelId, systemPrompt, this);
  }

  enabled(): boolean {
    return this.plugin.enabled;
  }
}

interface Plugin {
  id: string;
  enabled: boolean;
}

let LLM_SERVER: LLMSrv | undefined = undefined;

export const getLLMSrv = async (): Promise<LLMSrv | undefined> => {
  if (LLM_SERVER) {
    return LLM_SERVER;
  }
  const plugins: Plugin[] = await getBackendSrv().get('/api/plugins');
  const plugin = plugins.find((p) => p.id === 'grafana-llm-app');
  if (!plugin || !plugin.enabled) {
    return undefined;
  }
  const srv = new LLMPluginImpl(plugin);
  await srv.init();
  LLM_SERVER = srv;
  return LLM_SERVER;
}

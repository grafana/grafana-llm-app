export type LLMAppHealthCheck = {
  details: {
    openAIEnabled?: boolean;
    vectorEnabled?: boolean;
    version?: string;
  };
};

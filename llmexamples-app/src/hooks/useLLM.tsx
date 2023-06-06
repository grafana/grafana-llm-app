import React, { createContext, memo, useContext, useEffect, useState } from 'react';

import { getLLMSrv, LLMSrv } from 'api/llmPlugin';

type LLMContextType = React.PropsWithChildren<{}> & {
  llm: LLMSrv | undefined;
  isLoading: boolean;
}

const LLMContext = createContext<LLMContextType | undefined>(undefined);

export const LLMProvider = memo<React.PropsWithChildren<{}>>(({ children }) => {
  const [llmPlugin, setLLMPlugin] = useState<LLMSrv | undefined>(undefined);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  useEffect(() => {
    const setPlugin = async () => {
      const plugin = await getLLMSrv();
      setIsLoading(false);
      setLLMPlugin(plugin);
    };
    setPlugin();
  }, []);

  return (
    <LLMContext.Provider value={{ isLoading, llm: llmPlugin }}>
      {children}
    </LLMContext.Provider>
  );
});

LLMProvider.displayName = "LLMProvider";

// A hook to load the LLM provider from context.
export const useLLM = (): LLMContextType => {
  const context = useContext(LLMContext);

  if (context === undefined) {
    throw new Error('You can only use `useLLM` in a component wrapped in a `LLMProvider`.');
  }

  return context;
};


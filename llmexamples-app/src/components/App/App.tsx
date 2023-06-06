import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
import { ExamplePage } from '../../pages';
import { LLMProvider } from 'hooks/useLLM';

export function App(props: AppRootProps) {
  return (
    <LLMProvider>
      <Switch>
        <Route component={ExamplePage} />
      </Switch>
    </LLMProvider>
  );
}

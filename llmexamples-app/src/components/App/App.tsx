import React from 'react';
import { Route, Switch } from 'react-router-dom';

import { AppRootProps } from '@grafana/data';

import { ExamplePage, VectorSearch } from '../../pages';

export function App(props: AppRootProps) {
  return (
    <Switch>
      <Route exact component={VectorSearch} path="/a/grafana-llmexamples-app/vector-search" />
      {/* Default page */}
      <Route component={ExamplePage} />
    </Switch>
  );
}

import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
import { ExamplePage } from '../../pages';

export function App(props: AppRootProps) {
  return (
    <Switch>
      <Route component={ExamplePage} />
    </Switch>
  );
}

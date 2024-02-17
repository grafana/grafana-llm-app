import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
import { Models } from '../../pages';

export function App(props: AppRootProps) {
  return (
      <Switch>
        {/* Default page */}
        <Route component={Models} />
      </Switch>
  );
}

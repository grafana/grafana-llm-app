import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
import { Models } from '../../pages';

export function App(props: AppRootProps) {
  return (
    <Routes>
      {/* Default page */}
      <Route element={<Models />} />
    </Routes>
  );
}

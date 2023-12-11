import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
import { Models } from '../../pages';

export function App(props: AppRootProps) {
  return (
    <BrowserRouter>
      <Routes>
        {/* Default page */}
        <Route Component={Models} />
      </Routes>
    </BrowserRouter>
  );
}

import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { AppRootProps, PluginType } from '@grafana/data';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from './App';
import { BackendSrv, getBackendSrv, setBackendSrv } from '@grafana/runtime';

describe('Components/App', () => {
  let props: AppRootProps;
  let origBackendSrv: BackendSrv;

  beforeEach(() => {
    jest.resetAllMocks();
    origBackendSrv = getBackendSrv();

    props = {
      basename: 'a/sample-app',
      meta: {
        id: 'sample-app',
        name: 'Sample App',
        type: PluginType.app,
        enabled: true,
        jsonData: {},
      },
      query: {},
      path: '',
      onNavChanged: jest.fn(),
    } as unknown as AppRootProps;
  });

  afterEach(() => {
    setBackendSrv(origBackendSrv);
  });

  test('renders without an error"', async () => {
    const getMock = jest.fn().mockResolvedValue({ data: "models response" });
    setBackendSrv({ ...origBackendSrv, get: getMock });
    render(
      <BrowserRouter>
        <App {...props} />
      </BrowserRouter>
    );

    await waitFor(() => {
        expect(screen.queryByText(/models response/i)).toBeInTheDocument();
        });
    });
});

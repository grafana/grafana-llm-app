import React from 'react';
import { testIds } from '../components/testIds';
import { useAsync } from 'react-use';
import { getBackendSrv } from '@grafana/runtime';

export function Models() {
  const { error, loading, value } = useAsync(() => {
    const backendSrv = getBackendSrv();

    return backendSrv.get(`api/plugins/grafana-llm-app/resources/openai/v1/models`);
  });

  if (loading) {
    return (
      <div data-testid={testIds.models.container}>
        <span>Loading...</span>
      </div>
    );
  }

  if (error || !value) {
    return (
      <div data-testid={testIds.models.container}>
        <span>Error loading models: {error?.message}</span>
      </div>
    );
  }

  return (
    <div data-testid={testIds.models.container}>
      <h1>Available Models</h1>
      <pre>{JSON.stringify(value, null, 2)}</pre>
    </div>
  );
}

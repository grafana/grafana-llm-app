import React, { useState } from "react";
import { useAsync } from 'react-use';

import { llms } from '@grafana/experimental';
import { PluginPage } from "@grafana/runtime";
import { Button, Input, Spinner } from "@grafana/ui";

const dashboardsCollection = 'grafana.core.dashboards';

interface DashboardSearchResult {
  title: string | null;
  description: string | null;
  panels: Array<{ title: string | null; description: string | null; }>;
}

export function VectorSearch(): JSX.Element {
  // The current input value.
  const [input, setInput] = useState('');
  // The final search term to send to the vector service, updated when the button is clicked.
  const [searchTerm, setSearchTerm] = useState('');
  // The results from the vector service.
  const [searchResults, setSearchResults] = useState<DashboardSearchResult[] | undefined>(undefined);

  const { loading, error, value } = useAsync(async () => {
    const value = {
      enabled: await llms.vector.enabled(),
    };
    if (!value.enabled) {
      return value;
    }
    if (searchTerm === '') {
      return value;
    }

    const results = await llms.vector.search<DashboardSearchResult>({
      query: searchTerm,
      collection: dashboardsCollection,
      topK: 5,
    });
    setSearchResults(results.map((result) => result.payload));
    return value;
  }, [searchTerm]);

  return (
    <PluginPage>
      {value?.enabled ? (
        <>
          <h3>Semantic search for dashboards</h3>
          <Input
            value={input}
            onChange={(e) => setInput(e.currentTarget.value)}
            placeholder="Enter a search term"
          />
          <br />
          <Button type="submit" onClick={() => setSearchTerm(input)}>Submit</Button>
          <br />
          <div>{loading ? (
            <Spinner />
          ) : (error ? (
            <div>error: {error}</div>
          ) : (searchResults === undefined ? (
            <></>
          ) : (
            <>
              <h4>Results</h4>
              <table>
                <thead>
                  <tr>
                    <th>Title</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  {searchResults?.map((result, i) => (
                    <tr key={i}>
                      <td>{result.title}</td>
                      <td>{result.description}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </>
          )))

          }</div>
        </>
      ) : (
        <div>Vector search not enabled.</div>
      )}
    </PluginPage>
  )
}

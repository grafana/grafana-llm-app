import React from 'react';

import { PluginExtensionLinkConfig, PluginExtensionPanelContext, PluginExtensionPoints, PluginExtensionTypes } from "@grafana/data"
import { getBackendSrv } from '@grafana/runtime';
import { useAsync } from 'react-use';
import { scan } from 'rxjs/operators';
import { Spinner } from '@grafana/ui';
import { Dashboard } from '@grafana/schema';
import { streamChatCompletions } from 'utils/utils.api';

interface ExplainPanelModalProps {
  context: PluginExtensionPanelContext,
}

const panelExplainerPrompt = 'Given the following JSON representation of a Grafana panel, explain what it shows. Be fairly brief in your summary, and use the present tense.';

const ExplainPanelModal = ({ context }: ExplainPanelModalProps) => {
  const backendSrv = getBackendSrv();
  const [streamState, setStreamState] = React.useState<string>('');
  const state = useAsync(async () => {
    // Load the current dashboard JSON and find the relevant panel.
    const dashboardJSON = await backendSrv.get(`/api/dashboards/uid/${context.dashboard.uid}`) as { dashboard: Dashboard };
    const panelJSON = dashboardJSON.dashboard.panels?.find(
      // @ts-ignore: some panels don't have IDs, that's fine because they just won't match.
      (panel) => panel.id === context.id
    );
    // Use the panel JSON as the user prompt.
    // TODO: include the data? or in future some kind of screenshot
    // of the data, somehow.
    const userPrompt = JSON.stringify(panelJSON, null, 2);
    // Stream the completions. Each element is the next stream chunk.
    const stream = streamChatCompletions({
      model: 'gpt-3.5-turbo',
      systemPrompt: panelExplainerPrompt,
      userPrompt,
    }).pipe(
      // Accumulate the stream chunks into a single string.
      scan((acc, delta) => acc + delta, '')
    );
    // Subscribe to the stream and update the state for each returned value.
    return stream.subscribe(setStreamState);
  });
  if (state.loading || streamState === '') {
    return <Spinner />
  }
  if (state.error) {
    // TODO: handle errors.
    return null;
  }
  return (
    <div>{streamState}</div>
  )
}

export const panelExplainer: PluginExtensionLinkConfig<PluginExtensionPanelContext> = {
  title: 'Explain this panel',
  description: 'Explain this panel',
  type: PluginExtensionTypes.link,
  extensionPointId: PluginExtensionPoints.DashboardPanelMenu,
  onClick: (event, { context, openModal }) => {
    if (event !== undefined) {
      event.preventDefault();
    }
    if (context === undefined) {
      return;
    }
    openModal({
      title: 'Panel explanation',
      body: () => <ExplainPanelModal context={context} />,
    });
  },
};

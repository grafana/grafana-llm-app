import React from 'react';

import { PluginExtensionLinkConfig, PluginExtensionPanelContext, PluginExtensionPoints, PluginExtensionTypes } from "@grafana/data"
import { getBackendSrv } from '@grafana/runtime';
import { useAsync } from 'react-use';
import { Spinner } from '@grafana/ui';
import { Dashboard } from '@grafana/schema';
import { llms } from '@grafana/experimental';

interface ExplainPanelModalProps {
  context: PluginExtensionPanelContext,
}

const panelExplainerPrompt = 'Given the following JSON representation of a Grafana panel, explain what it shows. Be fairly brief in your summary, and use the present tense.';

const ExplainPanelModal = ({ context }: ExplainPanelModalProps) => {
  const backendSrv = getBackendSrv();
  const [streamState, setStreamState] = React.useState<string>('');
  const state = useAsync(async () => {

    // Check if the LLM plugin is enabled and configured.
    // If not, we won't be able to make requests, so return early.
    const enabled = await llms.openai.enabled();
    if (!enabled) {
      return { enabled };
    }

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
    llms.openai.streamChatCompletions({
      model: 'gpt-3.5-turbo',
      messages: [
        { role: 'system', content: panelExplainerPrompt },
        { role: 'user', content: userPrompt },
      ]
    })
      // Accumulate the stream chunks into a single string.
      .pipe(llms.openai.accumulateContent())
      // Subscribe to the stream and update the state for each returned value.
      .subscribe(setStreamState);
    return { enabled: true };
  });
  if (state.loading || streamState === '') {
    return <Spinner />
  }
  if (state.error) {
    // TODO: handle errors.
    return null;
  }
  if (!(state.value?.enabled ?? false)) {
    return <div>LLM plugin not enabled.</div>
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

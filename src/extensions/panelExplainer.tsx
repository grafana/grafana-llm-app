import React from 'react';

import { PluginExtensionLinkConfig, PluginExtensionPanelContext, PluginExtensionPoints, PluginExtensionTypes } from "@grafana/data"
import { getBackendSrv } from '@grafana/runtime';
import { useAsync } from 'react-use';
import { Spinner } from '@grafana/ui';
import { Dashboard } from '@grafana/schema';
import { chatCompletions } from 'utils/utils.api';

interface ExplainPanelModalProps {
  context: PluginExtensionPanelContext,
}

const panelExplainerPrompt = 'Given the following JSON representation of a Grafana panel, explain what it shows. Be fairly brief in your summary, and use the present tense.';

const ExplainPanelModal = ({ context }: ExplainPanelModalProps) => {
  const backendSrv = getBackendSrv();
  const state = useAsync(async () => {
    const dashboardJSON = await backendSrv.get(`/api/dashboards/uid/${context.dashboard.uid}`) as { dashboard: Dashboard };
    const panelJSON = dashboardJSON.dashboard.panels?.find(
      // @ts-ignore
      (panel) => panel.id === context.id
    );
    const userPrompt = JSON.stringify(panelJSON, null, 2);
    const response = await chatCompletions({
      model: 'gpt-3.5-turbo',
      systemPrompt: panelExplainerPrompt,
      userPrompt,
    });
    return response;
  });
  if (state.loading) {
    return <Spinner />
  }
  if (state.error || state.value === undefined) {
    return null;
  }
  return (
    <div>
      {state.value}
    </div>
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


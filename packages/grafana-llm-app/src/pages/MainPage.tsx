import React from 'react';
import { css } from '@emotion/css';
import { PluginPage } from '@grafana/runtime';
import { useTheme2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';

import { Models } from './Models';
import { MCPToolsWithProvider } from './MCPTools';
import { testIds } from 'components/testIds';

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(4)};
    padding: ${theme.spacing(2)};
    max-width: 1200px;
    margin: 0 auto;
  `,
  header: css`
    text-align: center;
    margin-bottom: ${theme.spacing(3)};
  `,
  title: css`
    font-size: ${theme.typography.h1.fontSize};
    font-weight: ${theme.typography.h1.fontWeight};
    color: ${theme.colors.text.primary};
    margin: 0 0 ${theme.spacing(1)} 0;
  `,
  subtitle: css`
    font-size: ${theme.typography.h4.fontSize};
    color: ${theme.colors.text.secondary};
    margin: 0;
    font-weight: normal;
  `,
  sectionsContainer: css`
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: ${theme.spacing(4)};

    @media (max-width: 768px) {
      grid-template-columns: 1fr;
    }
  `,
  section: css`
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.radius.default};
    border: 1px solid ${theme.colors.border.medium};
    padding: ${theme.spacing(3)};
    overflow: hidden;
  `,
  sectionTitle: css`
    font-size: ${theme.typography.h3.fontSize};
    font-weight: ${theme.typography.h3.fontWeight};
    color: ${theme.colors.text.primary};
    margin: 0 0 ${theme.spacing(2)} 0;
    padding-bottom: ${theme.spacing(1)};
    border-bottom: 1px solid ${theme.colors.border.weak};
  `,
  sectionContent: css`
    // Remove default margins from child components
    h1,
    h2 {
      margin-top: 0;
      font-size: ${theme.typography.h4.fontSize};
    }

    pre {
      max-height: 300px;
      overflow-y: auto;
      background: ${theme.colors.background.primary};
      border: 1px solid ${theme.colors.border.weak};
      border-radius: ${theme.shape.radius.default};
      padding: ${theme.spacing(2)};
      font-size: ${theme.typography.bodySmall.fontSize};
    }

    ul {
      margin: 0;
      padding-left: ${theme.spacing(3)};
    }

    li {
      margin-bottom: ${theme.spacing(0.5)};
      line-height: 1.5;
    }
  `,
  fullWidthSection: css`
    grid-column: 1 / -1;
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.radius.default};
    border: 1px solid ${theme.colors.border.medium};
    padding: ${theme.spacing(3)};
    margin-top: ${theme.spacing(2)};
  `,
  infoBox: css`
    background: ${theme.colors.info.transparent};
    border: 1px solid ${theme.colors.info.border};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(2)};
    margin-bottom: ${theme.spacing(3)};
    color: ${theme.colors.info.text};
    font-size: ${theme.typography.bodySmall.fontSize};
    line-height: 1.4;
  `,
});

export function MainPage() {
  const theme = useTheme2();
  const styles = getStyles(theme);

  return (
    <PluginPage>
      <div data-testid={testIds.mainPage.container} className={styles.container}>
        <header className={styles.header}>
          <h1 className={styles.title}>Grafana LLM</h1>
          <p className={styles.subtitle}>Large Language Model integration with Model Context Protocol support</p>
        </header>

        <div className={styles.infoBox}>
          <strong>About this plugin:</strong> This plugin provides LLM capabilities to Grafana through OpenAI-compatible
          APIs and extends functionality with Model Context Protocol (MCP) tools for enhanced AI workflows.
        </div>

        <div className={styles.sectionsContainer}>
          <section className={styles.section}>
            <h2 className={styles.sectionTitle}>Language Models</h2>
            <div className={styles.sectionContent}>
              <Models />
            </div>
          </section>

          <section className={styles.section}>
            <h2 className={styles.sectionTitle}>MCP Tools</h2>
            <div className={styles.sectionContent}>
              <MCPToolsWithProvider />
            </div>
          </section>
        </div>

        <section className={styles.fullWidthSection}>
          <h2 className={styles.sectionTitle}>Getting Started</h2>
          <div className={styles.sectionContent}>
            <p>
              <strong>Models:</strong> View and manage available language models configured for this Grafana instance.
              Models are provided through the configured LLM provider (OpenAI, Anthropic, or custom).
            </p>
            <p>
              <strong>MCP Tools:</strong> Model Context Protocol tools extend LLM capabilities with access to Grafana
              data sources, dashboards, alerts, and other contextual information. These tools can be used by AI
              assistants to provide more relevant and data-driven responses.
            </p>
          </div>
        </section>
      </div>
    </PluginPage>
  );
}

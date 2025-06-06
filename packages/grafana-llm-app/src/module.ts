import { AppPlugin } from '@grafana/data';

import { MainPage } from './pages';
import { AppConfig } from './components/AppConfig';

export const plugin = new AppPlugin<{}>().setRootPage(MainPage).addConfigPage({
  title: 'Configuration',
  icon: 'cog',
  body: AppConfig,
  id: 'configuration',
});

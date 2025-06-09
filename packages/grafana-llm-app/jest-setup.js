// Jest setup provided by Grafana scaffolding
import './.config/jest-setup';

import { TextEncoder } from 'util';
import './src/test/mocks/streams';

global.TextEncoder = TextEncoder;

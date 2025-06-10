// Jest setup provided by Grafana scaffolding
import './.config/jest-setup';

import { TransformStream } from 'node:stream/web';
import { TextEncoder } from 'util';

global.TextEncoder = TextEncoder;
global.TransformStream = TransformStream;

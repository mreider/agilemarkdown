// Bootstrap for the laboratory web build. Mounts the same <App/>
// the VS Code extension does, webpack's resolve.alias maps the
// `../webview/vscode-api` import to `./web-api` so the React tree
// runs against the HTTP + WebSocket backend instead of postMessage.

import * as React from 'react';
import { createRoot } from 'react-dom/client';
import '../webview/styles.css';
import './lab.css';

// Pull in the web shim so its WebSockets boot once.
import './web-api';

import { LabLayout } from './LabLayout';

const container = document.getElementById('root');
if (!container) {
  throw new Error('laboratory: #root not found');
}
createRoot(container).render(<LabLayout />);

// Kick the snapshot + log WS connections on next tick.
setTimeout(() => { window.__amWebBoot?.(); }, 0);

// LabLayout splits the screen into the existing <App/> on top and a
// fixed BottomPanel underneath. The grid sizes the bottom pane to a
// fixed 280px so the screenshot framing stays consistent across runs.

import * as React from 'react';
import { App } from '../webview/App';
import { BottomPanel } from './BottomPanel';

export function LabLayout() {
  return (
    <div className="lab-root">
      <div className="lab-top">
        <App />
      </div>
      <BottomPanel />
    </div>
  );
}

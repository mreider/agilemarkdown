// BottomPanel is the laboratory's terminal-area pane. One tab visible
// at a time (per LABORATORY.md decision #4). Phase 2 ships CommandLog;
// Phase 3 adds MCP Console and Terminal as additional tabs.
//
// Layout: a fixed-height tray under the board. Tabs at the very top.
// Active tab's content fills the rest. The Board above auto-shrinks
// because the lab-root grid is `1fr <panel-height>`.

import * as React from 'react';
import { useState } from 'react';
import { CommandLog } from './CommandLog';
import { McpConsole } from './McpConsole';
import { Terminal } from './Terminal';

type Tab = 'log' | 'mcp' | 'term';

export function BottomPanel() {
  const [tab, setTab] = useState<Tab>('log');
  // Mount-and-keep state per tab: switching tabs preserves the
  // CommandLog scrollback, the MCP form contents, and the Terminal's
  // PTY session. We toggle visibility with CSS instead of unmounting.
  return (
    <div className="lab-bottom">
      <div className="lab-bottom-tabs" role="tablist">
        <TabBtn label="CommandLog"  active={tab === 'log'}  onClick={() => setTab('log')}  />
        <TabBtn label="MCP Console" active={tab === 'mcp'}  onClick={() => setTab('mcp')}  />
        <TabBtn label="Terminal"    active={tab === 'term'} onClick={() => setTab('term')} />
      </div>
      <div className="lab-bottom-body">
        <div hidden={tab !== 'log'}  style={{ height: '100%' }}><CommandLog /></div>
        <div hidden={tab !== 'mcp'}  style={{ height: '100%' }}><McpConsole /></div>
        <div hidden={tab !== 'term'} style={{ height: '100%' }}><Terminal /></div>
      </div>
    </div>
  );
}

function TabBtn({ label, active, onClick }: { label: string; active: boolean; onClick(): void }) {
  return (
    <button
      role="tab"
      aria-selected={active}
      className={`lab-tab ${active ? 'active' : ''}`}
      onClick={onClick}
    >
      {label}
    </button>
  );
}

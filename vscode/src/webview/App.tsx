import * as React from 'react';
import { useEffect, useMemo, useState } from 'react';
import {
  PriorityListResult,
  IceboxListResult,
  DashboardResult,
  VelocityHistoryResult,
  BurnupResult,
  CFDResult,
  TypeMixResult,
  EpicProgressResult,
  GetItemResult,
  GetCommentsResult,
  ListTasksResult,
  HistoryResult,
  OrderRow,
} from '../shared/types';
import { Sidebar } from './components/Sidebar';
import { Board } from './components/Board';
import { DetailPanel } from './components/DetailPanel';
import { Analytics } from './components/Analytics';
import { SprintPlan } from './components/SprintPlan';
import { Icon } from './components/icons';
import { SearchBox } from './components/SearchBox';
import { vscode } from './vscode-api';

export interface Snapshot {
  backlog: string;
  backlogs: string[];
  priority: PriorityListResult;
  icebox: IceboxListResult;
  dashboard: DashboardResult | null;
  velocity: VelocityHistoryResult;
  typeMix: TypeMixResult;
  burnup: BurnupResult | null;
  cfd: CFDResult | null;
  epics: EpicProgressResult[];
}

export interface Detail {
  item: GetItemResult;
  comments: GetCommentsResult;
  tasks: ListTasksResult;
  history?: HistoryResult;
}

export type SidebarView = 'all' | 'my-work' | 'current' | 'backlog' | 'icebox' | 'done';
export interface FilterState {
  view: SidebarView;
  label?: string;
  sort: 'rank' | 'points' | 'age';
}

export function App() {
  const [snap, setSnap] = useState<Snapshot | null>(null);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [tab, setTab] = useState<'stories' | 'plan' | 'analytics'>('stories');
  const [error, setError] = useState<string | null>(null);
  const [sprintPlanData, setSprintPlanData] = useState<any | null>(null);
  const [filter, setFilter] = useState<FilterState>({ view: 'all', sort: 'rank' });
  const [me, setMe] = useState<{ name: string; email: string } | null>(null);

  useEffect(() => {
    function onMsg(e: MessageEvent) {
      const m = e.data as { type: string; payload?: any };
      switch (m.type) {
        case 'snapshot':
          setSnap(m.payload);
          break;
        case 'detail':
          setDetail(m.payload);
          break;
        case 'sprint-plan':
          setSprintPlanData(m.payload);
          break;
        case 'whoami':
          setMe(m.payload);
          break;
        case 'refresh':
          vscode?.postMessage({ type: 'load' });
          break;
        case 'error':
          setError(m.payload?.message || 'unknown error');
          break;
      }
    }
    window.addEventListener('message', onMsg);
    vscode?.postMessage({ type: 'load' });
    vscode?.postMessage({ type: 'whoami' });
    return () => window.removeEventListener('message', onMsg);
  }, []);

  const counts = useMemo(() => {
    if (!snap) return { current: 0, backlog: 0, icebox: 0, accepted: 0, my: 0 };
    const acc = snap.dashboard?.stories_accepted_total ?? 0;
    const myMatch = (r: OrderRow) => me && (r.assignees || []).some(a =>
      a === me.name || a === me.email || a === me.email.split('@')[0]);
    const myWork = snap.priority.items.filter(r => me && myMatch(r)).length;
    return {
      current: snap.priority.items.filter(r => isCurrent(r)).length,
      backlog: snap.priority.items.filter(r => !isCurrent(r)).length,
      icebox: snap.icebox.count,
      accepted: acc,
      my: myWork,
    };
  }, [snap, me]);

  const labels = useMemo(() => {
    if (!snap) return [] as { name: string; count: number }[];
    const map = new Map<string, number>();
    for (const r of [...snap.priority.items, ...snap.icebox.items]) {
      for (const t of r.tags || []) {
        map.set(t, (map.get(t) || 0) + 1);
      }
    }
    return Array.from(map.entries()).map(([name, count]) => ({ name, count })).sort((a, b) => b.count - a.count);
  }, [snap]);

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand"><span className="brand-dot" /> Agile Markdown</div>
        <div className="topbar-spacer" />
        <button className="btn" onClick={() => vscode?.postMessage({ type: 'load' })}>Refresh</button>
      </header>

      <nav className="subbar">
        <button className={`tab ${tab === 'stories' ? 'active' : ''}`} onClick={() => setTab('stories')}>
          <Icon.Stories /> Stories
        </button>
        <button
          className={`tab ${tab === 'plan' ? 'active' : ''}`}
          onClick={() => {
            setTab('plan');
            if (snap?.backlog) vscode?.postMessage({ type: 'load-sprint-plan', payload: { backlog: snap.backlog } });
          }}
        >
          <Icon.Calendar /> Plan
        </button>
        <button className={`tab ${tab === 'analytics' ? 'active' : ''}`} onClick={() => setTab('analytics')}>
          <Icon.Analytics /> Analytics
        </button>
        <div className="subbar-spacer" />
        <SearchBox onSelect={path => vscode?.postMessage({ type: 'open-detail', payload: { path } })} />
        <button
          className="icon-btn subbar-help"
          title="Open agilemarkdown documentation"
          onClick={() => vscode?.postMessage({ type: 'open-docs' })}
        >
          <Icon.Help />
        </button>
        {snap?.backlog && (
          <button
            className="btn-primary subbar-add"
            onClick={() => vscode?.postMessage({ type: 'new-story-prompt', payload: { backlog: snap.backlog } })}
          >
            <Icon.Plus /> Add story
          </button>
        )}
      </nav>

      {error && <div className="error-banner">{error}</div>}

      {!snap ? (
        <div className="empty">Loading…</div>
      ) : !snap.backlog ? (
        <div className="empty-state">
          <h2>No backlogs in this project yet</h2>
          <p>A backlog is a folder of <code>.md</code> stories plus a matching <code>&lt;name&gt;.md</code> overview file. Create one to start ranking work.</p>
          <button className="cta" onClick={() => vscode?.postMessage({ type: 'create-backlog' })}>Create a backlog…</button>
        </div>
      ) : tab === 'stories' ? (
        <div className="board">
          <Sidebar counts={counts} velocity={snap.velocity} backlog={snap.backlog} filter={filter} setFilter={setFilter} labels={labels} />
          <Board snap={snap} filter={filter} me={me} onSelect={path => vscode?.postMessage({ type: 'open-detail', payload: { path } })} selectedPath={detail?.item.path} />
        </div>
      ) : tab === 'plan' ? (
        <SprintPlan data={sprintPlanData} onRefresh={() => vscode?.postMessage({ type: 'load-sprint-plan', payload: { backlog: snap.backlog } })} />
      ) : (
        <Analytics snap={snap} />
      )}

      {detail && (
        <DetailPanel
          detail={detail}
          onClose={() => setDetail(null)}
        />
      )}
    </div>
  );
}

function isCurrent(row: OrderRow): boolean {
  // Counts as "current iteration" for the Sidebar tally: anything in
  // flight or still in the queue. Accepted/rejected are terminal and
  // belong to the Done view.
  return row.status !== 'accepted' && row.status !== 'rejected';
}

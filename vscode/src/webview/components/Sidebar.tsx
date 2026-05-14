import * as React from 'react';
import { VelocityHistoryResult } from '../../shared/types';
import { Sparkline } from './Sparkline';
import { Icon } from './icons';
import { vscode } from '../vscode-api';
import { FilterState, SidebarView } from '../App';

export interface SidebarCounts {
  current: number;
  backlog: number;
  icebox: number;
  accepted: number;
  my: number;
}

export function Sidebar({ counts, velocity, backlog, filter, setFilter, labels }: {
  counts: SidebarCounts;
  velocity: VelocityHistoryResult;
  backlog: string;
  filter: FilterState;
  setFilter(f: FilterState): void;
  labels: { name: string; count: number }[];
}) {
  const items: Array<{ key: SidebarView; label: string; count: number; icon: React.ReactNode }> = [
    { key: 'my-work',  label: 'My work',           count: counts.my,        icon: <Icon.Stories /> },
    { key: 'current',  label: 'Current iteration', count: counts.current,   icon: <Icon.Layers /> },
    { key: 'backlog',  label: 'Backlog',           count: counts.backlog,   icon: <Icon.Inbox /> },
    { key: 'icebox',   label: 'Icebox',            count: counts.icebox,    icon: <Icon.Snowflake /> },
    { key: 'done',     label: 'Done',              count: counts.accepted,  icon: <Icon.Check /> },
  ];
  const accepted = velocity.rows.map(r => r.accepted);
  const latest = accepted.length > 0 ? accepted[accepted.length - 1] : 0;

  function setView(view: SidebarView) {
    // Clicking the active row clears the filter, so a second click
    // returns to "All". Less surprising than a permanent-toggle that
    // sticks until you reload.
    if (filter.view === view) setFilter({ ...filter, view: 'all' });
    else setFilter({ ...filter, view });
  }
  function toggleLabel(name: string) {
    setFilter({ ...filter, label: filter.label === name ? undefined : name });
  }

  return (
    <aside className="side">
      <button
        className="side-add btn-primary"
        onClick={() => vscode?.postMessage({ type: 'new-story-prompt', payload: { backlog } })}
        title={`New story in ${backlog}`}
      >
        <Icon.Plus /> Add story
      </button>

      <div className="side-section-title">Views</div>
      <div
        className={`nav-item ${filter.view === 'all' ? 'active' : ''}`}
        onClick={() => setView('all')}
      >
        <span className="nav-item-icon"><Icon.Stories /></span>
        <span className="nav-item-label">All stories</span>
      </div>
      {items.map(it => (
        <div
          key={it.key}
          className={`nav-item ${filter.view === it.key ? 'active' : ''}`}
          onClick={() => setView(it.key)}
        >
          <span className="nav-item-icon">{it.icon}</span>
          <span className="nav-item-label">{it.label}</span>
          <span className="nav-count">{it.count}</span>
        </div>
      ))}

      {labels.length > 0 && (
        <>
          <div className="side-section-title">Labels</div>
          {labels.slice(0, 12).map(l => (
            <div
              key={l.name}
              className={`nav-item ${filter.label === l.name ? 'active' : ''}`}
              onClick={() => toggleLabel(l.name)}
            >
              <span className="nav-item-icon"><Icon.Tag /></span>
              <span className="nav-item-label">{l.name}</span>
              <span className="nav-count">{l.count}</span>
            </div>
          ))}
        </>
      )}

      <div className="velocity-card">
        <div className="velocity-label">Velocity</div>
        <div className="velocity-num">{Math.round(latest)} <small>pts/iter</small></div>
        <Sparkline data={accepted} />
        <div className="velocity-foot">last {accepted.length} iter</div>
      </div>
    </aside>
  );
}

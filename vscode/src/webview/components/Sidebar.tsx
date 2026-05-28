import * as React from 'react';
import { VelocityHistoryResult } from '../../shared/types';
import { Sparkline } from './Sparkline';
import { Icon } from './icons';
import { FilterState, SidebarView } from '../App';

export interface SidebarCounts {
  backlog: number;
  current: number;
  icebox: number;
  accepted: number;
  my: number;
}

export function Sidebar({ counts, velocity, filter, setFilter, labels }: {
  counts: SidebarCounts;
  velocity: VelocityHistoryResult;
  backlog: string;
  filter: FilterState;
  setFilter(f: FilterState): void;
  labels: { name: string; count: number }[];
}) {
  // Backlog is the default landing view, the whole project's stories
  // in both columns. The other rows are filters: a smaller slice of
  // the same data. The dedicated "Add story" button at the top of the
  // sidebar is gone; new stories are created via the + buttons on the
  // Current and Icebox column headers.
  const items: Array<{ key: SidebarView; label: string; count: number; icon: React.ReactNode }> = [
    { key: 'backlog', label: 'Backlog',           count: counts.backlog,  icon: <Icon.Stories /> },
    { key: 'my-work', label: 'My work',           count: counts.my,       icon: <Icon.Stories /> },
    { key: 'current', label: 'Current iteration', count: counts.current,  icon: <Icon.Layers /> },
    { key: 'icebox',  label: 'Icebox',            count: counts.icebox,   icon: <Icon.Snowflake /> },
    { key: 'done',    label: 'Done',              count: counts.accepted, icon: <Icon.Check /> },
  ];
  const accepted = velocity.rows.map(r => r.accepted);
  const latest = accepted.length > 0 ? accepted[accepted.length - 1] : 0;

  function setView(view: SidebarView) {
    // Views read as nav links: clicking the active row is a no-op.
    if (filter.view === view) return;
    setFilter({ ...filter, view });
  }
  function toggleLabel(name: string) {
    setFilter({ ...filter, label: filter.label === name ? undefined : name });
  }

  return (
    <aside className="side">
      <div className="side-section-title">Views</div>
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

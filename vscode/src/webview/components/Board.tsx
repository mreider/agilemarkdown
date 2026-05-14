import * as React from 'react';
import { useMemo, useState } from 'react';
import { Snapshot } from '../App';
import { OrderRow, EpicProgressResult } from '../../shared/types';
import { StoryCard, epicColor } from './StoryCard';
import { Icon } from './icons';
import { vscode } from '../vscode-api';
import { FilterState } from '../App';

export function Board({ snap, filter, me, onSelect, selectedPath }: {
  snap: Snapshot;
  filter: FilterState;
  me: { name: string; email: string } | null;
  onSelect(path: string): void;
  selectedPath?: string;
}) {
  const velocity = snap.priority.velocity || 0;
  // Apply view + label filter before bucketing so iteration bands
  // reflect only what's visible. Counts in the sidebar stay total.
  const filteredPriority = useMemo(
    () => applyFilter(snap.priority.items, filter, me),
    [snap.priority.items, filter, me],
  );
  const filteredIcebox = useMemo(
    () => applyFilter(snap.icebox.items, filter, me),
    [snap.icebox.items, filter, me],
  );
  const bands = useMemo(() => bucketIntoIterations(filteredPriority, velocity), [filteredPriority, velocity]);
  // Hide the priority/icebox columns when the active view doesn't
  // include them, so the layout always tells the truth about what's
  // being shown.
  const showPriority = filter.view !== 'icebox' && filter.view !== 'done';
  const showIcebox = filter.view === 'all' || filter.view === 'icebox';
  const [showAdd, setShowAdd] = useState(false);
  const [draft, setDraft] = useState('');

  function postStatus(path: string, status: string) {
    vscode?.postMessage({ type: 'set-status', payload: { path, status } });
  }

  function dragData(row: OrderRow) {
    return JSON.stringify({ path: row.path, status: row.status, source: 'priority' });
  }

  function onDrop(e: React.DragEvent, target: 'priority' | 'icebox', anchor?: { after?: string }) {
    const raw = e.dataTransfer.getData('text/am-row');
    if (!raw) return;
    const data = JSON.parse(raw);
    if (target === 'icebox' && data.source === 'priority') {
      vscode?.postMessage({ type: 'move-to-icebox', payload: { backlog: snap.backlog, item_path: data.path } });
    } else if (target === 'priority' && data.source === 'icebox') {
      vscode?.postMessage({ type: 'move-to-priority', payload: { backlog: snap.backlog, item_paths: [data.path], after: anchor?.after } });
    } else if (target === 'priority' && data.source === 'priority' && anchor?.after) {
      vscode?.postMessage({ type: 'rank-item', payload: { backlog: snap.backlog, item_path: data.path, after: anchor.after } });
    }
  }

  return (
    <>
      {/* Current */}
      {showPriority && <>
      <div className="col">
        <div className="col-head">
          <span className="col-head-icon"><Icon.Layers /></span>
          <span className="col-title">Current</span>
          <span className="col-meta">{currentBandMeta(bands)}</span>
          <div className="col-actions">
            <button className="icon-btn" title="New story" onClick={() => setShowAdd(s => !s)}><Icon.Plus /></button>
          </div>
        </div>
        <div className="col-body" onDragOver={e => e.preventDefault()} onDrop={e => onDrop(e, 'priority')}>
          {showAdd && (
            <div className="story add-row">
              <input
                value={draft}
                onChange={e => setDraft(e.target.value)}
                placeholder="New story title"
                onKeyDown={e => {
                  if (e.key === 'Enter' && draft.trim()) {
                    vscode?.postMessage({ type: 'create-item', payload: { backlog: snap.backlog, title: draft.trim() } });
                    setDraft('');
                    setShowAdd(false);
                  }
                  if (e.key === 'Escape') { setShowAdd(false); setDraft(''); }
                }}
                autoFocus
              />
            </div>
          )}
          {bands.length === 0 && (
            <div className="empty">Priority is empty.</div>
          )}
          {bands.map((band, i) => (
            <div key={i}>
              <IterMarker num={band.iter} points={band.points} accepted={band.accepted} isCurrent={i === 0} />
              {band.items.map(row => (
                <div
                  key={row.path}
                  draggable
                  onDragStart={e => { e.dataTransfer.setData('text/am-row', dragData(row)); }}
                  onDragOver={e => e.preventDefault()}
                  onDrop={e => onDrop(e, 'priority', { after: basename(row.path) })}
                >
                  <StoryCard
                    row={row}
                    selected={selectedPath === row.path}
                    onClick={() => onSelect(row.path)}
                    onAction={status => postStatus(row.path, status)}
                  />
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>

      </>}
      {/* Icebox */}
      {showIcebox && <>
      <div className="col">
        <div className="col-head">
          <span className="col-head-icon"><Icon.Snowflake /></span>
          <span className="col-title">Icebox</span>
          <span className="col-meta">{filteredIcebox.length} {filteredIcebox.length === 1 ? 'story' : 'stories'}</span>
        </div>
        <div className="col-body" onDragOver={e => e.preventDefault()} onDrop={e => onDrop(e, 'icebox')}>
          {filteredIcebox.length === 0 && <div className="empty">Icebox is empty.</div>}
          {filteredIcebox.map(row => (
            <div
              key={row.path}
              draggable
              onDragStart={e => {
                e.dataTransfer.setData('text/am-row', JSON.stringify({ path: row.path, status: row.status, source: 'icebox' }));
              }}
            >
              <StoryCard
                row={row}
                selected={selectedPath === row.path}
                onClick={() => onSelect(row.path)}
                onAction={status => postStatus(row.path, status)}
              />
            </div>
          ))}
        </div>
      </div>

      </>}
      {/* Epics */}
      <div className="col">
        <div className="col-head">
          <span className="col-head-icon"><Icon.Layers /></span>
          <span className="col-title">Epics</span>
          <span className="col-meta">{snap.epics.length}</span>
        </div>
        <div className="epic-list">
          {snap.epics.map(e => <EpicCard key={e.slug} e={e} />)}
          {snap.epics.length === 0 && <div className="empty">No epics yet.</div>}
        </div>
      </div>
    </>
  );
}

// applyFilter narrows a row list to the active filter. Order is
// preserved: rank stays meaningful inside the result.
function applyFilter(
  rows: OrderRow[],
  filter: FilterState,
  me: { name: string; email: string } | null,
): OrderRow[] {
  const myKeys = me ? [me.name, me.email, me.email.split('@')[0]].filter(Boolean) : [];
  return rows.filter(r => {
    switch (filter.view) {
      case 'current':
        if (r.status === 'accepted' || r.status === 'rejected') return false;
        break;
      case 'backlog':
        // Below-the-line items handled inside iteration bands; for
        // now, treat 'backlog' as everything in priority that hasn't
        // started yet.
        if (r.status !== 'unstarted') return false;
        break;
      case 'done':
        if (r.status !== 'accepted') return false;
        break;
      case 'my-work':
        if (!me || !(r.assignees || []).some(a => myKeys.includes(a))) return false;
        break;
      // 'all' and 'icebox' don't constrain per-row state here; the
      // Board hides the wrong columns to express scope.
    }
    if (filter.label && !(r.tags || []).includes(filter.label)) return false;
    return true;
  });
}

function basename(p: string): string {
  const i = p.lastIndexOf('/');
  return i >= 0 ? p.slice(i + 1) : p;
}

// Header meta line for the Current column. Mirrors the prototype:
// "7/18 pts · 39%". Falls back to "<points> pts" before any stories
// have been accepted in the current iteration band.
function currentBandMeta(bands: Band[]): string {
  if (bands.length === 0) return '0 pts';
  const b = bands[0];
  if (b.accepted > 0) {
    const pct = b.points > 0 ? Math.round((b.accepted / b.points) * 100) : 0;
    return `${b.accepted}/${b.points} pts · ${pct}%`;
  }
  return `${b.points} pts`;
}

function IterMarker({ num, points, accepted, isCurrent }: { num: number; points: number; accepted?: number; isCurrent?: boolean }) {
  const pct = points > 0 ? Math.min(100, Math.round(((accepted ?? 0) / points) * 100)) : 0;
  const range = iterationRange(num);
  return (
    <div className="iter-marker">
      <div className="iter-num">{num}</div>
      <div className="iter-info">
        <div className="iter-range">{range}{isCurrent && ' · Current'}</div>
        <div className="iter-sub">{accepted != null ? `${accepted}/${points}` : `${points}`} pts</div>
      </div>
      <div className="iter-progress">
        <div className="progress-track">
          <div className="progress-fill" style={{ width: `${pct}%` }} />
        </div>
      </div>
    </div>
  );
}

// iterationRange renders "May 4 — May 10" for an iteration band by
// offset from the current week's Monday. num=1 is the current
// iteration (offset 0), num=2 is next, and so on. Length defaults to
// one week; future work can plumb cfg.iteration.length_weeks through.
function iterationRange(num: number): string {
  const today = new Date();
  const dow = (today.getDay() + 6) % 7; // 0 = Monday
  const monday = new Date(today);
  monday.setDate(today.getDate() - dow);
  monday.setHours(0, 0, 0, 0);
  const start = new Date(monday);
  start.setDate(monday.getDate() + 7 * (num - 1));
  const end = new Date(start);
  end.setDate(start.getDate() + 6);
  return `${fmtMonDay(start)} — ${fmtMonDay(end)}`;
}

function fmtMonDay(d: Date): string {
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

function EpicCard({ e }: { e: EpicProgressResult }) {
  const pct = Math.round(e.percent_done);
  const color = epicColor(e.slug) || 'var(--ink-4)';
  // Distribution segments for the stacked bar. Without per-state
  // counts on EpicProgressResult we approximate: accepted (filled),
  // remaining (track). When the Go side starts emitting in-flight /
  // unstarted breakdowns this becomes a three-segment bar.
  const accepted = e.accepted_points || 0;
  const remaining = Math.max(0, (e.total_points || 0) - accepted);
  const total = accepted + remaining || 1;
  return (
    <div className="epic-card">
      <div className="epic-card-head">
        <span className="epic-color" style={{ background: color }} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div className="epic-name">{e.slug}</div>
          <div className="epic-id">{e.accepted_stories}/{e.total_stories} stories</div>
        </div>
      </div>
      <div className="epic-progress">
        <div className="epic-progress-track"><div className="epic-progress-fill" style={{ width: `${pct}%`, background: color }} /></div>
        <div className="epic-progress-pct">{pct}%</div>
      </div>
      <div className="epic-stack">
        {accepted > 0 && <div className="epic-stack-bar" style={{ flex: accepted / total, background: color }} />}
        {remaining > 0 && <div className="epic-stack-bar" style={{ flex: remaining / total, background: 'var(--line)' }} />}
      </div>
      <div className="epic-meta">
        <div><strong>{e.accepted_points}</strong>/{e.total_points} pts</div>
      </div>
    </div>
  );
}

interface Band {
  iter: number;
  points: number;
  accepted: number;
  items: OrderRow[];
}

// bucketIntoIterations splits the priority list into iteration bands using
// the project velocity. Items that already accepted in an earlier window
// stay attached to the first band; we have no historical iteration data on
// the client. Velocity 0 -> single band of everything.
function bucketIntoIterations(items: OrderRow[], velocity: number): Band[] {
  if (items.length === 0) return [];
  const cap = velocity > 0 ? velocity : Number.MAX_SAFE_INTEGER;
  const bands: Band[] = [];
  let band: Band = { iter: 1, points: 0, accepted: 0, items: [] };
  for (const it of items) {
    const p = parseFloat(it.estimate || '0') || 0;
    if (band.points + p > cap && band.items.length > 0) {
      bands.push(band);
      band = { iter: bands.length + 1, points: 0, accepted: 0, items: [] };
    }
    band.items.push(it);
    band.points += p;
    if (it.status === 'accepted') band.accepted += p;
  }
  bands.push(band);
  return bands;
}

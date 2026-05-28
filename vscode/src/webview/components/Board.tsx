import * as React from 'react';
import { useMemo, useState } from 'react';
import { Snapshot, BoardColumn, FilterColumn } from '../App';
import { OrderRow, EpicProgressResult } from '../../shared/types';
import { StoryCard, epicColor } from './StoryCard';
import { Icon } from './icons';
import { vscode } from '../vscode-api';

export function Board({
  snap, me, onSelect, selectedPath, onAddStory,
  searchQuery, onSearchQueryChange, candidates,
  visibleColumns, allColumns, columnLabels, onColumnToggle,
  filterColumns, onOpenFilter, onCloseFilter,
}: {
  snap: Snapshot;
  me: { name: string; email: string } | null;
  onSelect(path: string): void;
  selectedPath?: string;
  onAddStory(target: 'priority' | 'icebox'): void;
  searchQuery: string;
  onSearchQueryChange(q: string): void;
  candidates: { owners: string[]; tags: string[]; epics: string[] };
  visibleColumns: Set<BoardColumn>;
  allColumns: BoardColumn[];
  columnLabels: Record<BoardColumn, string>;
  onColumnToggle(col: BoardColumn, show: boolean): void;
  filterColumns: FilterColumn[];
  onOpenFilter(kind: 'epic' | 'tag', value: string): void;
  onCloseFilter(id: string): void;
}) {
  const velocity = snap.priority.velocity || 0;
  const priorityItems = snap.priority.items;
  const iceboxItems = snap.icebox.items;
  const doneItems = snap.done?.items || [];
  const allItems = useMemo(() => [...priorityItems, ...iceboxItems, ...doneItems], [priorityItems, iceboxItems, doneItems]);
  const bands = useMemo(() => bucketIntoIterations(priorityItems, velocity), [priorityItems, velocity]);
  // Multi-field search state. Lifted into Board because the Search
  // column lives inside the board layout.
  const [searchOwner, setSearchOwner] = useState('');
  const [searchTag, setSearchTag] = useState('');
  const [searchEpic, setSearchEpic] = useState('');
  const [searchStatuses, setSearchStatuses] = useState<Set<string>>(new Set());
  const toggleSearchStatus = (s: string) => {
    setSearchStatuses(prev => {
      const n = new Set(prev);
      if (n.has(s)) n.delete(s); else n.add(s);
      return n;
    });
  };
  // Search column: substring match over title + path, priority items
  // first then icebox, preserving the order they appear in their
  // home column. Lowercased; trimmed query. Default to empty when
  // the prop is unset (some tests render Board without it).
  const trimmedQuery = (searchQuery || '').trim().toLowerCase();
  const searchResults = useMemo(() => {
    const hasAnyFilter = trimmedQuery || searchOwner || searchTag || searchEpic || searchStatuses.size > 0;
    if (!hasAnyFilter) return [] as Array<{ row: OrderRow; source: 'priority' | 'icebox' | 'done' }>;
    const out: Array<{ row: OrderRow; source: 'priority' | 'icebox' | 'done' }> = [];
    const matches = (r: OrderRow) => {
      if (trimmedQuery) {
        const hay = `${r.title || ''} ${r.path || ''}`.toLowerCase();
        if (!hay.includes(trimmedQuery)) return false;
      }
      if (searchOwner && !(r.assignees || []).some(a => a.toLowerCase() === searchOwner.toLowerCase())) return false;
      if (searchTag && !(r.tags || []).some(t => t.toLowerCase() === searchTag.toLowerCase())) return false;
      if (searchEpic && (r.epic || '').toLowerCase() !== searchEpic.toLowerCase()) return false;
      if (searchStatuses.size > 0 && !searchStatuses.has(r.status || '')) return false;
      return true;
    };
    for (const r of priorityItems) if (matches(r)) out.push({ row: r, source: 'priority' });
    for (const r of iceboxItems) if (matches(r)) out.push({ row: r, source: 'icebox' });
    for (const r of doneItems) if (matches(r)) out.push({ row: r, source: 'done' });
    return out;
  }, [trimmedQuery, searchOwner, searchTag, searchEpic, searchStatuses, priorityItems, iceboxItems, doneItems]);
  const hasSearchInputs = !!(trimmedQuery || searchOwner || searchTag || searchEpic || searchStatuses.size > 0);

  // Filter-column row predicates (epic / tag).
  function rowsForFilter(f: FilterColumn): OrderRow[] {
    return allItems.filter(r => {
      if (f.kind === 'epic') return (r.epic || '').toLowerCase() === f.value.toLowerCase();
      return (r.tags || []).some(t => t.toLowerCase() === f.value.toLowerCase());
    });
  }
  // Silence unused-variable warnings for the snapshot-side params we
  // no longer read directly (legacy filter param removed).
  void me;

  // The `am` CLI resolves ITEM_PATH with filepath.Abs from cwd, so a
  // bare basename ("feature-1.md") posted from the board would land at
  // <repo>/<basename> instead of <repo>/<backlog>/<basename> and the
  // command silently fails. Backlog-cd'd verbs (rank, ice, unice) are
  // unaffected because the extension cd's into the backlog before
  // running them; everything else (set-status, open-detail, comments,
  // tasks, ...) runs from the repo root and needs the prefix. The
  // sidebar already qualifies paths the same way before posting.
  function qpath(row: OrderRow): string {
    return `${snap.backlog}/${row.path}`;
  }

  function postStatus(row: OrderRow, status: string) {
    vscode?.postMessage({ type: 'set-status', payload: { path: qpath(row), status } });
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

  // Total of visible content columns + dynamic filter columns.
  const totalVisible = visibleColumns.size + filterColumns.length;

  return (
    <>
      {/* Empty state: no content columns AND no search. A single
          CTA invites the user to add a column back. */}
      {totalVisible === 0 && (
        <div className="board-empty">
          <div className="board-empty-card">
            <div className="board-empty-title">No columns</div>
            <div className="board-empty-sub">Pick a column to start working.</div>
            <div className="board-empty-actions">
              {allColumns.map(c => (
                <button key={c} className="btn-mini primary" onClick={() => onColumnToggle(c, true)}>
                  Add {columnLabels[c]}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Search column: multi-field form. Backed by the same
          predicate as the old top-bar search but with owner / tag /
          epic / status filters too. */}
      {visibleColumns.has('search') && (
      <div className="col col-search">
        <div className="col-head">
          <span className="col-head-icon"><Icon.Search /></span>
          <span className="col-title">Search</span>
          <span className="col-meta">
            {hasSearchInputs ? `${searchResults.length} ${searchResults.length === 1 ? 'result' : 'results'}` : ''}
          </span>
          <div className="col-actions">
            <button className="col-close" aria-label="Hide Search column" title="Hide Search column"
              onClick={() => onColumnToggle('search', false)}><Icon.X /></button>
          </div>
        </div>
        <div className="col-body">
          <div className="search-form">
            <input
              className="search-text"
              placeholder="Text in title or path…"
              value={searchQuery}
              onChange={e => onSearchQueryChange(e.target.value)}
              autoFocus
            />
            <label className="search-label">Owner</label>
            <select className="search-select" value={searchOwner} onChange={e => setSearchOwner(e.target.value)}>
              <option value="">any</option>
              {candidates.owners.map(o => <option key={o} value={o}>{o}</option>)}
            </select>
            <label className="search-label">Tag</label>
            <select className="search-select" value={searchTag} onChange={e => setSearchTag(e.target.value)}>
              <option value="">any</option>
              {candidates.tags.map(t => <option key={t} value={t}>{t}</option>)}
            </select>
            <label className="search-label">Epic</label>
            <select className="search-select" value={searchEpic} onChange={e => setSearchEpic(e.target.value)}>
              <option value="">any</option>
              {candidates.epics.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
            <label className="search-label">Status</label>
            <div className="search-status-row">
              {['unstarted','started','finished','delivered','accepted','rejected'].map(s => (
                <label key={s} className={`search-status-chip ${searchStatuses.has(s) ? 'on' : ''}`}>
                  <input type="checkbox" checked={searchStatuses.has(s)} onChange={() => toggleSearchStatus(s)} />
                  {s}
                </label>
              ))}
            </div>
            <button className="btn-mini search-clear" onClick={() => {
              onSearchQueryChange('');
              setSearchOwner(''); setSearchTag(''); setSearchEpic('');
              setSearchStatuses(new Set());
            }}>Clear</button>
          </div>
          {!hasSearchInputs && (
            <div className="empty">Add a filter to see matching stories.</div>
          )}
          {hasSearchInputs && searchResults.length === 0 && (
            <div className="empty">No matches.</div>
          )}
          {searchResults.map(({ row, source }) => (
            <StoryCard
              key={`${source}/${row.path}`}
              row={row}
              selected={selectedPath === qpath(row)}
              inIcebox={source === 'icebox'}
              onClick={() => onSelect(qpath(row))}
              onTagDoubleClick={t => onOpenFilter('tag', t)}
                onEpicClick={slug => onOpenFilter('epic', slug)}
              onAction={status => {
                if (source === 'icebox' && status === 'started') {
                  vscode?.postMessage({ type: 'pull-from-icebox', payload: { backlog: snap.backlog, item_path: row.path } });
                } else {
                  postStatus(row, status);
                }
              }}
              onReject={() => { vscode?.postMessage({ type: 'open-detail', payload: { path: qpath(row), autoAction: 'reject' } }); }}
              onEstimate={estimate => { vscode?.postMessage({ type: 'set-estimate', payload: { path: qpath(row), estimate } }); }}
            />
          ))}
        </div>
      </div>
      )}

      {/* Backlog. The current iteration is just the top iteration
          band; there's no separate Current column. */}
      {visibleColumns.has('backlog') && (
      <div className="col">
        <div className="col-head">
          <span className="col-head-icon"><Icon.Stories /></span>
          <span className="col-title">Backlog</span>
          <span className="col-meta">{currentBandMeta(bands)}</span>
          <div className="col-actions">
            <button
              className="col-add"
              aria-label="New story in Backlog"
              title="New story in Backlog"
              onClick={() => onAddStory('priority')}
            ><Icon.Plus /></button>
            <button
              className="col-close"
              aria-label="Hide Backlog column"
              title="Hide Backlog column"
              onClick={() => onColumnToggle('backlog', false)}
            ><Icon.X /></button>
          </div>
        </div>
        <div className="col-body" onDragOver={e => e.preventDefault()} onDrop={e => onDrop(e, 'priority')}>
          {bands.length === 0 && (
            <div className="empty">Priority is empty.</div>
          )}
          {bands.map((band, i) => (
            <div key={i} className={`iter-band ${i === 0 ? 'iter-current' : ''} ${band.untimed ? 'iter-untimed' : ''}`}>
              <IterMarker num={band.iter} points={band.points} accepted={band.accepted} isCurrent={i === 0} untimed={band.untimed} />
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
                    selected={selectedPath === qpath(row)}
                    onClick={() => onSelect(qpath(row))}
                    onTagDoubleClick={t => onOpenFilter('tag', t)}
                onEpicClick={slug => onOpenFilter('epic', slug)}
                    onAction={status => postStatus(row, status)}
                    onReject={() => {
                      vscode?.postMessage({
                        type: 'open-detail',
                        payload: { path: qpath(row), autoAction: 'reject' },
                      });
                    }}
                    onEstimate={estimate => {
                      vscode?.postMessage({
                        type: 'set-estimate',
                        payload: { path: qpath(row), estimate },
                      });
                    }}
                  />
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
      )}

      {/* Icebox */}
      {visibleColumns.has('icebox') && (
      <div className="col">
        <div className="col-head">
          <span className="col-head-icon"><Icon.Snowflake /></span>
          <span className="col-title">Icebox</span>
          <span className="col-meta">{iceboxItems.length} {iceboxItems.length === 1 ? 'story' : 'stories'}</span>
          <div className="col-actions">
            <button
              className="col-add"
              aria-label="New story in Icebox"
              title="New story in Icebox"
              onClick={() => onAddStory('icebox')}
            ><Icon.Plus /></button>
            <button
              className="col-close"
              aria-label="Hide Icebox column"
              title="Hide Icebox column"
              onClick={() => onColumnToggle('icebox', false)}
            ><Icon.X /></button>
          </div>
        </div>
        <div className="col-body" onDragOver={e => e.preventDefault()} onDrop={e => onDrop(e, 'icebox')}>
          {iceboxItems.length === 0 && <div className="empty">Icebox is empty.</div>}
          {iceboxItems.map(row => (
            <div
              key={row.path}
              draggable
              onDragStart={e => {
                e.dataTransfer.setData('text/am-row', JSON.stringify({ path: row.path, status: row.status, source: 'icebox' }));
              }}
            >
              <StoryCard
                row={row}
                selected={selectedPath === qpath(row)}
                inIcebox
                onClick={() => onSelect(qpath(row))}
                onTagDoubleClick={t => onOpenFilter('tag', t)}
                onEpicClick={slug => onOpenFilter('epic', slug)}
                onEstimate={estimate => {
                  vscode?.postMessage({
                    type: 'set-estimate',
                    payload: { path: qpath(row), estimate },
                  });
                }}
                onAction={status => {
                  // Starting an icebox story promotes it to priority
                  // and flips the status in one step. Without the
                  // promote the card would change color but stay in
                  // the Icebox column, which contradicts the user's
                  // intent ("I'm starting on this now").
                  if (status === 'started') {
                    vscode?.postMessage({
                      type: 'pull-from-icebox',
                      payload: { backlog: snap.backlog, item_path: row.path },
                    });
                  } else {
                    postStatus(row, status);
                  }
                }}
              />
            </div>
          ))}
        </div>
      </div>
      )}

      {/* Epics - read-only roll-up. */}
      {visibleColumns.has('epics') && (
      <div className="col">
        <div className="col-head">
          <span className="col-head-icon"><Icon.Layers /></span>
          <span className="col-title">Epics</span>
          <span className="col-meta">{snap.epics.length}</span>
          <div className="col-actions">
            <button
              className="col-close"
              aria-label="Hide Epics column"
              title="Hide Epics column"
              onClick={() => onColumnToggle('epics', false)}
            ><Icon.X /></button>
          </div>
        </div>
        <div className="epic-list">
          {sortEpicsByDelivery(snap.epics, allItems, velocity).map(e => (
            <EpicCard
              key={e.slug}
              e={e}
              meta={computeEpicMeta(e, allItems, velocity)}
              onDoubleClick={() => onOpenFilter('epic', e.slug)}
            />
          ))}
          {snap.epics.length === 0 && <div className="empty">No epics yet.</div>}
        </div>
      </div>
      )}

      {/* Done - stories accepted in prior iterations. Read-only;
          no action buttons. Grouped by historical sprint based on
          each row's `accepted` timestamp. These markers reflect what
          actually happened and are never recomputed against today. */}
      {visibleColumns.has('done') && (
      <div className="col col-done">
        <div className="col-head">
          <span className="col-head-icon"><Icon.CheckAll /></span>
          <span className="col-title">Done</span>
          <span className="col-meta">{doneItems.length} {doneItems.length === 1 ? 'story' : 'stories'}</span>
          <div className="col-actions">
            <button className="col-close" aria-label="Hide Done column" title="Hide Done column"
              onClick={() => onColumnToggle('done', false)}><Icon.X /></button>
          </div>
        </div>
        <div className="col-body">
          {doneItems.length === 0 && (
            <div className="empty">No stories accepted in prior iterations yet.</div>
          )}
          {groupDoneByIteration(doneItems).map(group => (
            <div key={group.key} className="iter-band">
              <DoneIterMarker startISO={group.start} points={group.points} />
              {group.items.map(row => (
                <StoryCard
                  key={row.path}
                  row={row}
                  selected={selectedPath === qpath(row)}
                  readOnly
                  onClick={() => onSelect(qpath(row))}
                  onTagDoubleClick={t => onOpenFilter('tag', t)}
                onEpicClick={slug => onOpenFilter('epic', slug)}
                />
              ))}
            </div>
          ))}
        </div>
      </div>
      )}

      {/* Dynamic filter columns opened by double-clicking an epic
          card or tag chip. Each one shows the stories matching the
          predicate; closing removes it. */}
      {filterColumns.map(f => {
        const matched = rowsForFilter(f);
        const epicHeader = f.kind === 'epic'
          ? snap.epics.find(ep => ep.slug.toLowerCase() === f.value.toLowerCase())
          : null;
        return (
          <div key={f.id} className="col col-filter">
            <div className="col-head">
              <span className="col-head-icon">{f.kind === 'epic' ? <Icon.Layers /> : <Icon.Tag />}</span>
              <span className="col-title">{f.kind === 'epic' ? 'Epic' : 'Tag'}: {f.value}</span>
              <span className="col-meta">{matched.length} {matched.length === 1 ? 'story' : 'stories'}</span>
              <div className="col-actions">
                <button className="col-close" aria-label="Close filter" title="Close filter"
                  onClick={() => onCloseFilter(f.id)}><Icon.X /></button>
              </div>
            </div>
            <div className="col-body">
              {epicHeader && (
                <div className="filter-header">
                  <EpicCard e={epicHeader} meta={computeEpicMeta(epicHeader, allItems, velocity)} />
                </div>
              )}
              {matched.length === 0 && <div className="empty">No stories.</div>}
              {matched.map(row => (
                <StoryCard
                  key={row.path}
                  row={row}
                  selected={selectedPath === qpath(row)}
                  onClick={() => onSelect(qpath(row))}
                  onTagDoubleClick={t => onOpenFilter('tag', t)}
                onEpicClick={slug => onOpenFilter('epic', slug)}
                  onAction={status => postStatus(row, status)}
                  onReject={() => { vscode?.postMessage({ type: 'open-detail', payload: { path: qpath(row), autoAction: 'reject' } }); }}
                  onEstimate={estimate => { vscode?.postMessage({ type: 'set-estimate', payload: { path: qpath(row), estimate } }); }}
                />
              ))}
            </div>
          </div>
        );
      })}
    </>
  );
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

function IterMarker({ num, points, accepted, isCurrent, untimed }: { num: number; points: number; accepted?: number; isCurrent?: boolean; untimed?: boolean }) {
  if (untimed) {
    return (
      <div className="iter-marker">
        <div className="iter-num"></div>
        <div className="iter-info">
          <div className="iter-range">Backlog · untimed</div>
          <div className="iter-sub">{points} pts</div>
        </div>
      </div>
    );
  }
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

// iterationRange renders "May 4 - May 10" for an iteration band by
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
  return `${fmtMonDay(start)} - ${fmtMonDay(end)}`;
}

function fmtMonDay(d: Date): string {
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

// sortEpicsByDelivery orders the Epics column by when each epic
// actually shipped (or is projected to ship). Finished epics use the
// max accepted-date across their stories. In-progress epics use the
// projected ETA from velocity. Epics with any unestimated story have
// no projection and sort to the bottom alphabetically. Earlier
// dates come first so the column reads top-down as "shipped → next".
function sortEpicsByDelivery(epics: EpicProgressResult[], allItems: OrderRow[], velocity: number): EpicProgressResult[] {
  const keyOf = (e: EpicProgressResult): { date: number; rank: number; slug: string } => {
    const meta = computeEpicMeta(e, allItems, velocity);
    if (e.total_stories > 0 && e.accepted_stories === e.total_stories) {
      // Finished: max accepted timestamp across the epic's stories.
      let latest = 0;
      for (const r of allItems) {
        if ((r.epic || '').toLowerCase() !== e.slug.toLowerCase()) continue;
        if (!r.accepted) continue;
        const t = new Date(r.accepted).getTime();
        if (!isNaN(t) && t > latest) latest = t;
      }
      return { date: latest || 0, rank: 0, slug: e.slug };
    }
    if (meta.etaISO) {
      return { date: new Date(meta.etaISO + 'T00:00:00Z').getTime(), rank: 1, slug: e.slug };
    }
    // Pending estimation: no usable date. Sort to the bottom.
    return { date: Number.POSITIVE_INFINITY, rank: 2, slug: e.slug };
  };
  return [...epics].sort((a, b) => {
    const ka = keyOf(a), kb = keyOf(b);
    if (ka.rank !== kb.rank) return ka.rank - kb.rank;
    if (ka.date !== kb.date) return ka.date - kb.date;
    return ka.slug.localeCompare(kb.slug);
  });
}

// EpicMeta carries the projected-ship and unestimated-counts the
// Go side doesn't compute for us. The client has every story, so
// projection lives here.
interface EpicMeta {
  remainingPoints: number;
  unestimatedCount: number;
  // ISO yyyy-mm-dd of the projected ship Monday, or null when
  // projection isn't possible (unestimated stories or zero velocity).
  etaISO: string | null;
}

function computeEpicMeta(e: EpicProgressResult, allItems: OrderRow[], velocity: number): EpicMeta {
  let remaining = 0;
  let unestimated = 0;
  for (const r of allItems) {
    if ((r.epic || '').toLowerCase() !== e.slug.toLowerCase()) continue;
    if (r.type === 'release') continue;
    if (r.status === 'accepted') continue;
    const est = (r.estimate ?? '').trim();
    if (!est) {
      // Bugs and chores don't get points per the coach rule; only
      // unestimated features count as "needs sizing".
      if (!r.type || r.type === 'feature') unestimated += 1;
      continue;
    }
    const p = parseFloat(est) || 0;
    remaining += p;
  }
  let etaISO: string | null = null;
  if (unestimated === 0 && velocity > 0 && remaining > 0) {
    const sprints = Math.ceil(remaining / velocity);
    const today = new Date();
    const dow = (today.getDay() + 6) % 7;
    const monday = new Date(today);
    monday.setDate(today.getDate() - dow);
    monday.setHours(0, 0, 0, 0);
    const eta = new Date(monday);
    eta.setDate(monday.getDate() + 7 * sprints);
    etaISO = eta.toISOString().slice(0, 10);
  }
  return { remainingPoints: remaining, unestimatedCount: unestimated, etaISO };
}

function EpicCard({ e, meta, onDoubleClick }: { e: EpicProgressResult; meta?: EpicMeta; onDoubleClick?: () => void }) {
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
    <div className="epic-card" onDoubleClick={onDoubleClick} title={onDoubleClick ? 'Double-click to open epic filter' : undefined}>
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
        {meta && (meta.etaISO || meta.unestimatedCount > 0) && (
          <div className={`epic-eta ${meta.unestimatedCount > 0 ? 'pending' : ''}`}>
            {meta.etaISO ? (
              <>ETA {formatEtaLabel(meta.etaISO)}</>
            ) : (
              <>ETA pending - {meta.unestimatedCount} {meta.unestimatedCount === 1 ? 'story needs' : 'stories need'} sizing</>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function formatEtaLabel(iso: string): string {
  const d = new Date(iso + 'T00:00:00Z');
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

interface Band {
  iter: number;
  points: number;
  accepted: number;
  items: OrderRow[];
  // True when this band represents the untimed-backlog tail rather
  // than a forecasted sprint. Renders without a date range.
  untimed?: boolean;
}

// Monday of the week containing the given date (UTC). Iterations
// start on Monday in the default config; matches IterationStartFor
// on the Go side for the common case.
function iterationStartFor(d: Date): Date {
  const out = new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate()));
  const dow = (out.getUTCDay() + 6) % 7; // 0 = Monday
  out.setUTCDate(out.getUTCDate() - dow);
  return out;
}

interface DoneGroup {
  key: string;        // ISO date of the iteration's Monday
  start: string;      // same as key, for clarity at the marker site
  points: number;
  items: OrderRow[];
}

// groupDoneByIteration buckets each row by the Monday of the week
// its accepted timestamp falls in. The label reflects the actual
// historical week and never changes as time passes - Pivotal's
// "what shipped, when" record.
function groupDoneByIteration(rows: OrderRow[]): DoneGroup[] {
  const groups = new Map<string, DoneGroup>();
  for (const r of rows) {
    if (!r.accepted) continue;
    const d = new Date(r.accepted);
    if (isNaN(d.getTime())) continue;
    const monday = iterationStartFor(d);
    const key = monday.toISOString().slice(0, 10);
    let g = groups.get(key);
    if (!g) {
      g = { key, start: key, points: 0, items: [] };
      groups.set(key, g);
    }
    g.items.push(r);
    g.points += parseFloat(r.estimate || '0') || 0;
  }
  // Newest week first.
  return Array.from(groups.values()).sort((a, b) => (a.key < b.key ? 1 : -1));
}

function DoneIterMarker({ startISO, points }: { startISO: string; points: number }) {
  const start = new Date(startISO + 'T00:00:00Z');
  const end = new Date(start);
  end.setUTCDate(start.getUTCDate() + 6);
  return (
    <div className="iter-marker iter-done">
      <div className="iter-num"></div>
      <div className="iter-info">
        <div className="iter-range">{fmtMonDay(start)} - {fmtMonDay(end)} · Done</div>
        <div className="iter-sub">{points} pts</div>
      </div>
    </div>
  );
}

// MAX_BANDS caps the number of forward-projected iteration bands the
// board draws. Beyond this, remaining forecast rows pile into a final
// "Backlog (untimed)" band so a long unstarted tail doesn't draw
// dozens of speculative sprint headers. Matches the Go side's
// --iterations default of 2 (current + 1 future, plus untimed).
const MAX_BANDS = 2;

// bucketIntoIterations splits the priority list into iteration bands.
//
// Pass 1: every story whose status is currently being worked on
// (started, finished, delivered) or accepted in the current iteration
// goes into band 0. Status is the authority. A started story cannot
// be in a future sprint, no matter how many points.
//
// Pass 2: unstarted / rejected / release-marker rows fill forecast
// bands by accumulating points against velocity. After MAX_BANDS, the
// rest collapse into one "Backlog (untimed)" band.
//
// Velocity 0 collapses everything into a single band.
function bucketIntoIterations(items: OrderRow[], velocity: number): Band[] {
  if (items.length === 0) return [];
  const cap = velocity > 0 ? velocity : Number.MAX_SAFE_INTEGER;
  const isCurrent = (st?: string) =>
    st === 'started' || st === 'finished' || st === 'delivered' || st === 'accepted';
  const ptsOf = (it: OrderRow) => parseFloat(it.estimate || '0') || 0;
  const currentSprint: OrderRow[] = [];
  const forecast: OrderRow[] = [];
  for (const it of items) {
    if (isCurrent(it.status)) currentSprint.push(it);
    else forecast.push(it);
  }
  // Inside the current sprint, sort by status so the PM's eye lands
  // first on what needs attention: accepted (just shipped, great),
  // delivered (awaiting accept, PM moment), finished (awaiting
  // deploy), then started (in flight). Preserves rank order within
  // each status group so the dev pair's manual rank still matters.
  const statusOrder: Record<string, number> = {
    accepted: 0, delivered: 1, finished: 2, started: 3,
  };
  currentSprint.sort((a, b) => {
    const sa = statusOrder[a.status || ''] ?? 99;
    const sb = statusOrder[b.status || ''] ?? 99;
    if (sa !== sb) return sa - sb;
    return (a.index ?? 0) - (b.index ?? 0);
  });
  const bands: Band[] = [];
  const band0: Band = { iter: 1, points: 0, accepted: 0, items: [] };
  for (const it of currentSprint) {
    band0.items.push(it);
    band0.points += ptsOf(it);
    if (it.status === 'accepted') band0.accepted += ptsOf(it);
  }
  bands.push(band0);
  let band: Band = { iter: 2, points: 0, accepted: 0, items: [], untimed: false };
  let bandIndex = 1; // forward bands beyond the current sprint
  for (const it of forecast) {
    const p = ptsOf(it);
    const untimed = bandIndex >= MAX_BANDS;
    if (!untimed && band.points + p > cap && band.items.length > 0) {
      bands.push(band);
      bandIndex++;
      const newUntimed = bandIndex >= MAX_BANDS;
      band = { iter: bands.length + 1, points: 0, accepted: 0, items: [], untimed: newUntimed };
    }
    band.untimed = bandIndex >= MAX_BANDS;
    band.items.push(it);
    band.points += p;
  }
  if (band.items.length > 0) bands.push(band);
  return bands;
}

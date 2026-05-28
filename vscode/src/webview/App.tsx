import * as React from 'react';
import { useEffect, useState } from 'react';
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
import { Board } from './components/Board';
import { DetailPanel } from './components/DetailPanel';
import { NewStoryModal } from './components/NewStoryModal';
import { SettingsPanel } from './components/SettingsPanel';
import { Analytics } from './components/Analytics';
import { Icon } from './components/icons';
import { vscode, useBusy } from './vscode-api';

export interface Snapshot {
  // project is the human-readable workspace folder name. Shown in
  // the topbar so the brand identifies the current project instead
  // of saying "Agile Markdown" everywhere.
  project?: string;
  backlog: string;
  backlogs: string[];
  priority: PriorityListResult;
  icebox: IceboxListResult;
  // Stories accepted in prior iterations. Pivotal's "Done" panel;
  // not part of the active sprint capacity.
  done: IceboxListResult;
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
  // Hint from the extension that the panel should render directly
  // into a specific form on mount. Used by the card-level Reject
  // button to skip the Accept-ceremony step and land on the
  // reject-reason form.
  autoAction?: 'reject' | null;
}

export type SidebarView = 'backlog' | 'my-work' | 'current' | 'icebox' | 'done';
export interface FilterState {
  view: SidebarView;
  label?: string;
  sort: 'rank' | 'points' | 'age';
}

// BoardColumn names the content columns the user can toggle from
// the activity-bar tree. Backlog is the current/working sprint;
// Done is accepted-in-prior-iterations; Search is the multi-field
// query form; Epics and Icebox round out the canonical set.
export type BoardColumn = 'search' | 'backlog' | 'done' | 'icebox' | 'epics';
// In-app filter columns (epic / tag) open dynamically and are
// separate from the user-toggled set above.
export interface FilterColumn {
  id: string;
  kind: 'epic' | 'tag';
  value: string;
}
const ALL_COLUMNS: BoardColumn[] = ['search', 'backlog', 'done', 'icebox', 'epics'];
const COLUMN_LABELS: Record<BoardColumn, string> = {
  search:  'Search',
  backlog: 'Backlog',
  done:    'Done',
  icebox:  'Icebox',
  epics:   'Epics',
};
const VISIBLE_COLUMNS_KEY = 'am-visible-columns';

function loadVisibleColumns(): Set<BoardColumn> {
  if (typeof localStorage === 'undefined') return new Set(['backlog']);
  try {
    const raw = localStorage.getItem(VISIBLE_COLUMNS_KEY);
    if (!raw) return new Set(['backlog']);
    const parsed = JSON.parse(raw) as string[];
    const valid = parsed.filter(c => (ALL_COLUMNS as string[]).includes(c)) as BoardColumn[];
    // Allow the empty set ("no columns visible"), the user can
    // intentionally close everything and the empty state CTA brings
    // a column back. Don't auto-restore Backlog here.
    return new Set(valid);
  } catch {
    return new Set(['backlog']);
  }
}

function saveVisibleColumns(cols: Set<BoardColumn>) {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(VISIBLE_COLUMNS_KEY, JSON.stringify(Array.from(cols)));
  } catch {
    // localStorage may throw in private mode; ignore.
  }
}

export function App() {
  const [snap, setSnap] = useState<Snapshot | null>(null);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [tab, setTab] = useState<'home' | 'analytics' | 'settings'>('home');
  // Search query lifted to App so the SearchBox in the subbar can
  // drive a Search column in the Board. Empty string = no search.
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [me, setMe] = useState<{ name: string; email: string } | null>(null);
  // Project label seeded by the extension's `project` push on
  // webview creation. The snapshot also carries `project` once it
  // lands, but the early push avoids the flash where the topbar
  // briefly read "Agile Markdown" before the snapshot replaced it.
  const [projectName, setProjectName] = useState<string>('');
  // Raw workspace folder name. Used by Settings to show what Display
  // name falls back to when blank.
  const [folderName, setFolderName] = useState<string>('');
  // Which columns are visible on the Home board. Default is just
  // Backlog (one column = full width). Icebox and Epics are opt-in
  // via the "+ Column" picker. Persisted to localStorage so the
  // user's choice survives reloads.
  const [visibleColumns, setVisibleColumns] = useState<Set<BoardColumn>>(() => loadVisibleColumns());
  useEffect(() => { saveVisibleColumns(visibleColumns); }, [visibleColumns]);
  // Target of the New Story modal: 'priority' means the `+` button on
  // the Current column was clicked, 'icebox' means the `+` on the
  // Icebox column header. null means the modal is closed.
  const [addStoryTarget, setAddStoryTarget] = useState<'priority' | 'icebox' | null>(null);
  // Dynamic filter columns opened by double-clicking an epic or tag.
  // Each filter renders its own column; closing it removes the entry.
  const [filterColumns, setFilterColumns] = useState<FilterColumn[]>([]);
  function openFilter(kind: 'epic' | 'tag', value: string) {
    const id = `${kind}:${value}`;
    setFilterColumns(prev => {
      if (prev.some(f => f.id === id)) return prev;
      return [...prev, { id, kind, value }];
    });
  }
  function closeFilter(id: string) {
    setFilterColumns(prev => prev.filter(f => f.id !== id));
  }

  useEffect(() => {
    function onMsg(e: MessageEvent) {
      const m = e.data as { type: string; payload?: any };
      switch (m.type) {
        case 'snapshot':
          setSnap(m.payload);
          // Snapshot also carries project; keep them in sync in case
          // the user renamed the folder or set project.name later.
          if (m.payload?.project) setProjectName(m.payload.project);
          break;
        case 'project':
          if (m.payload?.name) setProjectName(m.payload.name);
          if (m.payload?.folderName) setFolderName(m.payload.folderName);
          break;
        case 'visible-columns':
          if (Array.isArray(m.payload)) {
            const valid = (m.payload as string[]).filter(c => (ALL_COLUMNS as string[]).includes(c)) as BoardColumn[];
            setVisibleColumns(new Set(valid));
          }
          break;
        case 'set-view':
          // Activity-bar sidebar switched the view. Webview just
          // mirrors the choice; persistence lives in extension state.
          if (typeof m.payload === 'string' && (['home','analytics','settings'] as const).includes(m.payload as any)) {
            setTab(m.payload as 'home' | 'analytics' | 'settings');
          }
          break;
        case 'detail':
          setDetail(m.payload);
          break;
        case 'close-detail':
          // Extension-driven panel dismissal. Used after delete-item,
          // when the story the panel is bound to no longer exists.
          setDetail(null);
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


  return (
    <div className="app">
      <header className="topbar">
        <div className="brand" title={projectName ? `Workspace: ${projectName}` : undefined}>
          <span className="brand-dot" /> {projectName}
        </div>
        <div className="topbar-spacer" />
        <BusyPill loading={!snap} />
      </header>

      {error && <div className="error-banner">{error}</div>}

      {!snap ? (
        <div className="empty">Loading…</div>
      ) : !snap.backlog ? (
        <CreateBacklogEmptyState />
      ) : tab === 'home' ? (
        <div className="board">
          <Board
            snap={snap}
            me={me}
            searchQuery={searchQuery}
            onSearchQueryChange={setSearchQuery}
            candidates={computeCandidates(snap)}
            visibleColumns={visibleColumns}
            allColumns={ALL_COLUMNS}
            columnLabels={COLUMN_LABELS}
            filterColumns={filterColumns}
            onOpenFilter={openFilter}
            onCloseFilter={closeFilter}
            onColumnToggle={(col, show) => {
              setVisibleColumns(prev => {
                const next = new Set(prev);
                if (show) next.add(col); else next.delete(col);
                return next;
              });
              vscode?.postMessage({ type: 'toggle-column-from-webview', payload: { col, show } });
            }}
            onSelect={path => vscode?.postMessage({ type: 'open-detail', payload: { path } })}
            selectedPath={detail?.item.path}
            onAddStory={(target) => { setTab('home'); setAddStoryTarget(target); }}
          />
        </div>
      ) : tab === 'settings' ? (
        <SettingsPanel folderName={folderName} />
      ) : (
        <Analytics snap={snap} />
      )}

      {detail && (
        <DetailPanel
          detail={detail}
          candidates={computeCandidates(snap)}
          onClose={() => {
            setDetail(null);
            vscode?.postMessage({ type: 'detail-closed' });
          }}
        />
      )}

      {addStoryTarget && snap?.backlog && (
        <NewStoryModal
          backlog={snap.backlog}
          target={addStoryTarget}
          onCancel={() => setAddStoryTarget(null)}
          onSubmit={(title, type, target, releaseDate) => {
            // create-item lands the story in the chosen target column
            // (priority or icebox) at the configured position. If the
            // user picked a non-feature type, queue a follow-up
            // set-type so the type is correct from the first snapshot.
            // For releases, also stamp release_date.
            vscode?.postMessage({ type: 'create-item', payload: { backlog: snap.backlog, title, target } });
            if (type !== 'feature') {
              const slug = title.trim().replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-+|-+$/g, '').toLowerCase();
              const path = `${snap.backlog}/${slug}.md`;
              vscode?.postMessage({ type: 'set-type', payload: { path, type } });
              if (type === 'release' && releaseDate) {
                vscode?.postMessage({
                  type: 'set-release-date',
                  payload: { path, release_date: releaseDate },
                });
              }
            }
            setAddStoryTarget(null);
          }}
        />
      )}
    </div>
  );
}

// computeCandidates aggregates the values currently in use across
// the project for the DetailPanel autocompletes. Owners and tags come
// from every visible story's frontmatter (priority + icebox); epic
// slugs come from the EpicProgressResult roll-up. Deduped + sorted
// case-insensitively. Returns empty pools when there's no snapshot.
function computeCandidates(snap: Snapshot | null): { owners: string[]; tags: string[]; epics: string[] } {
  if (!snap) return { owners: [], tags: [], epics: [] };
  const owners = new Set<string>();
  const tags = new Set<string>();
  const allRows = [
    ...(snap.priority.items || []),
    ...(snap.icebox.items || []),
    ...(snap.done?.items || []),
  ];
  for (const r of allRows) {
    for (const a of r.assignees || []) if (a) owners.add(a);
    for (const t of r.tags || []) if (t) tags.add(t);
  }
  const epics = new Set<string>();
  for (const e of (snap.epics || [])) if (e?.slug) epics.add(e.slug);
  // Also include any inline epic field on stories, covers fresh
  // slugs not yet aggregated into snap.epics.
  for (const r of allRows) {
    if (r.epic) epics.add(r.epic);
  }
  const sort = (set: Set<string>) => Array.from(set).sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()));
  return { owners: sort(owners), tags: sort(tags), epics: sort(epics) };
}

// CreateBacklogEmptyState: a single button. Single-backlog
// convention means there's nothing to name, the project gets a
// `stories/` folder + `stories.md` overview. Click once and you're
// in. The extension posts create-backlog with no name; the CLI
// defaults to "stories".
function CreateBacklogEmptyState() {
  return (
    <div className="empty-state">
      <h2>This project doesn't have any stories yet</h2>
      <div className="empty-state-form">
        <button
          className="cta"
          onClick={() => vscode?.postMessage({ type: 'create-backlog', payload: { name: 'stories' } })}
        >Set up stories folder</button>
      </div>
    </div>
  );
}

// BusyPill renders a feedback indicator in the topbar.
//  - On first paint, before any snapshot has arrived: "Loading…".
//    Stays up while the initial load + whoami + first snapshot
//    round-trip happens, so the user gets immediate feedback that
//    the extension is reaching for data.
//  - After the first snapshot lands: "Saving…" any time a write
//    is in flight (the original behavior).
// `loading` is a hint from App; when true we render regardless of
// the busy counter so the very first paint already shows the pill.
function BusyPill({ loading }: { loading: boolean }) {
  const busy = useBusy();
  if (!busy && !loading) return null;
  return (
    <>
      <div className="busy-bar" aria-hidden="true" />
      <span className="busy-pill" role="status" aria-live="polite">
        <Icon.Spinner />
        {loading ? 'Loading…' : 'Saving…'}
      </span>
    </>
  );
}


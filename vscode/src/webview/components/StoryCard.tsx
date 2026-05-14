import * as React from 'react';
import { OrderRow } from '../../shared/types';
import { TypeGlyph, StatePill, Avatar, Icon } from './icons';

// Deterministic palette so an epic's color is stable across renders
// without needing a color field on the backlog item.
const EPIC_PALETTE = ['#4F3DDB', '#0EA5A4', '#E97451', '#15803D', '#B45309', '#1F6FE5', '#6D28D9', '#B42318'];
export function epicColor(slug?: string): string | null {
  if (!slug) return null;
  let h = 0;
  for (let i = 0; i < slug.length; i++) h = (h * 31 + slug.charCodeAt(i)) >>> 0;
  return EPIC_PALETTE[h % EPIC_PALETTE.length];
}

export function StoryCard({ row, selected, onClick, onAction }: {
  row: OrderRow;
  selected?: boolean;
  onClick(): void;
  onAction?(state: string): void;
}) {
  const isRelease = row.type === 'release';
  const rail = epicColor(row.epic);
  if (isRelease) {
    return (
      <div className={`story is-release ${selected ? 'selected' : ''}`} onClick={onClick}>
        <span className="release-flag"><Icon.Flag /></span>
        <div className="story-body">
          <div className="story-row1">
            <div className="story-title">{row.title}</div>
          </div>
        </div>
      </div>
    );
  }
  const action = nextAction(row.status);
  return (
    <div className={`story ${selected ? 'selected' : ''}`} onClick={onClick} draggable>
      {rail && <span className="story-rail" style={{ background: rail }} />}
      <TypeGlyph type={row.type} />
      <div className="story-body">
        <div className="story-row1">
          <div className="story-title">{row.title}</div>
          <span className={`story-points ${row.estimate ? '' : 'unestimated'}`}>{row.estimate || '—'}</span>
        </div>
        <div className="story-row2">
          {row.blocked && <span className="story-blocked"><Icon.Block /> Blocked</span>}
          {!row.blocked && <StatePill state={row.status} />}
          {(row.assignees || []).slice(0, 3).map(a => <Avatar key={a} id={a} />)}
          {(row.tags || []).slice(0, 2).map(t => <span key={t} className="label-chip">{t}</span>)}
          {(row.comment_count ?? 0) > 0 && (
            <span className="story-meta"><Icon.Comment /> {row.comment_count}</span>
          )}
          {action && onAction && (
            <button
              className={`btn-mini ${action.kind}`}
              onClick={e => { e.stopPropagation(); onAction(action.next); }}
            >
              {action.label}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

function nextAction(state?: string): { label: string; kind: string; next: string } | null {
  switch (state) {
    case 'unstarted': return { label: 'Start',   kind: 'primary', next: 'started' };
    case 'started':   return { label: 'Finish',  kind: 'primary', next: 'finished' };
    case 'finished':  return { label: 'Deliver', kind: 'primary', next: 'delivered' };
    case 'delivered': return { label: 'Accept',  kind: 'success', next: 'accepted' };
    case 'rejected':  return { label: 'Restart', kind: 'primary', next: 'started' };
    default:          return null;
  }
}

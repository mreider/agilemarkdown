import * as React from 'react';
import { OrderRow } from '../../shared/types';
import { TypeGlyph, StatePill, Avatar, Icon, typeEmoji } from './icons';

// Epics share a single neutral color across the board. Per-epic
// palette was confusing - the rail/chip colors looked like status
// signals. Keep the function signature so existing callers compile,
// but always return the one ink color.
export function epicColor(slug?: string): string | null {
  return slug ? 'var(--ink-3)' : null;
}

// Inline estimate values offered on the card. A trimmed set vs. the
// detail panel's 0/1/2/3/5/8 so the card row stays compact. The first
// entry is "·" (clear/unpointed) so the user can return to no-estimate
// without opening the detail panel.
const CARD_PIPS: Array<{ label: string; value: string }> = [
  { label: '·', value: '' },
  { label: '1', value: '1' },
  { label: '3', value: '3' },
  { label: '5', value: '5' },
  { label: '8', value: '8' },
];

export function StoryCard({ row, selected, inIcebox, readOnly, onClick, onAction, onReject, onEstimate, onTagDoubleClick, onEpicClick }: {
  row: OrderRow;
  selected?: boolean;
  inIcebox?: boolean;
  readOnly?: boolean;
  onClick(): void;
  onAction?(state: string): void;
  onReject?(): void;
  onEstimate?(estimate: string): void;
  // Double-click a tag chip to open a filter column for that tag.
  onTagDoubleClick?(tag: string): void;
  // Single-click the epic chip to open a filter column for that epic.
  onEpicClick?(slug: string): void;
}) {
  const isRelease = row.type === 'release';
  const rail = epicColor(row.epic);
  if (isRelease) {
    // Release markers are date stakes, not stories. Render as a
    // narrow horizontal band with the target date prominent so the
    // PM can read "ship by X" at a glance.
    const dateLabel = formatReleaseDate(row);
    const past = isReleasePast(row);
    return (
      <div className={`release-bar ${selected ? 'selected' : ''}`} onClick={onClick} title="Release marker">
        <span className="release-flag" aria-hidden="true">{typeEmoji('release')}</span>
        <div className="release-title">{row.title}</div>
        {dateLabel && <div className="release-date">{dateLabel}</div>}
        {past && onAction && row.status !== 'accepted' && (
          // The marker's target date is today or earlier: offer a one-
          // click ship affordance. The accept transition stamps an
          // accepted timestamp; the Done view then anchors the marker
          // to that week.
          <button
            className="btn-mini success"
            draggable={false}
            onMouseDown={e => e.stopPropagation()}
            onClick={e => { e.stopPropagation(); onAction('accepted'); }}
          >Ship</button>
        )}
      </div>
    );
  }
  const action = nextAction(row.status);
  // Icebox stories carry their own visual identity (blue tint) so a
  // glance at the column tells you "these are unscheduled," and
  // dragging or starting one feels like a deliberate promotion. The
  // Start action on an icebox card moves the story into priority and
  // flips status in one step - see Board.tsx for the wiring.
  const classes = ['story'];
  if (selected) classes.push('selected');
  if (inIcebox) classes.push('is-icebox');
  // Accepted stories (current iteration or, in the Done column,
  // historical) render with the success palette and no action buttons.
  if (row.status === 'accepted') classes.push('is-accepted');
  if (readOnly) classes.push('is-readonly');
  return (
    <div className={classes.join(' ')} onClick={onClick} draggable>
      <TypeGlyph type={row.type} />
      <div className="story-body">
        <div className="story-row1">
          <div className="story-title">{row.title}</div>
          {/* Inline points picker. Features get the full pip row; bugs
              and chores are unpointed per the coach rule and show
              nothing. If onEstimate is not wired (e.g. an embedded
              read-only context), fall back to the static label. */}
          {(!row.type || row.type === 'feature') ? (
            onEstimate && !readOnly && row.status !== 'accepted' ? (
              <span
                className="story-points-pips"
                draggable={false}
                onMouseDown={e => e.stopPropagation()}
                onClick={e => e.stopPropagation()}
              >
                {CARD_PIPS.map(p => (
                  <button
                    key={p.value || 'clear'}
                    className={`points-pip ${(row.estimate || '') === p.value ? 'active' : ''}`}
                    aria-label={p.value ? `Estimate ${p.value}` : 'Clear estimate'}
                    onClick={e => { e.stopPropagation(); onEstimate(p.value); }}
                  >
                    {p.label}
                  </button>
                ))}
              </span>
            ) : (
              <span className={`story-points ${row.estimate ? '' : 'unestimated'}`}>{row.estimate || '-'}</span>
            )
          ) : null}
        </div>
        <div className="story-row2">
          {row.blocked && <span className="story-blocked"><Icon.Block /> Blocked</span>}
          {!row.blocked && <StatePill state={row.status} />}
          {(row.assignees || []).slice(0, 3).map(a => <Avatar key={a} id={a} />)}
          {row.epic && (
            <span
              className="epic-chip"
              style={rail ? { background: `${rail}1A`, borderColor: `${rail}66`, color: rail } : undefined}
              title={onEpicClick ? `Open ${row.epic} epic` : `Epic: ${row.epic}`}
              onClick={onEpicClick ? e => { e.stopPropagation(); onEpicClick(row.epic!); } : undefined}
            >
              <Icon.Layers /> {row.epic}
            </span>
          )}
          {(row.tags || []).slice(0, 2).map(t => (
            <span
              key={t}
              className="label-chip"
              title={onTagDoubleClick ? 'Double-click to open tag filter' : undefined}
              onDoubleClick={onTagDoubleClick ? e => { e.stopPropagation(); onTagDoubleClick(t); } : undefined}
            >
              {t}
            </span>
          ))}
          {(row.comment_count ?? 0) > 0 && (
            <span className="story-meta"><Icon.Comment /> {row.comment_count}</span>
          )}
          {!readOnly && row.status !== 'accepted' && ((action && onAction) || (row.status === 'delivered' && onReject)) ? (
            // Group the action buttons so margin-left:auto applies once
            // to the whole cluster instead of once per button. Without
            // the wrapper, Accept and Reject each consumed their own
            // auto-margin and the wide gap pushed Reject to the right
            // edge while Accept stayed pinned to the column.
            <div className="story-actions">
              {action && onAction && (
                <button
                  className={`btn-mini ${action.kind}`}
                  // mousedown also has to stop bubbling: the StoryCard's
                  // outer wrapper in Board.tsx is `draggable`, so a
                  // mousedown that reaches it starts an HTML5 drag and
                  // the matching click never fires. Without this guard
                  // the action button reads as a no-op even though the
                  // visual button highlights on hover.
                  draggable={false}
                  onMouseDown={e => e.stopPropagation()}
                  onClick={e => { e.stopPropagation(); onAction(action.next); }}
                >
                  {action.label}
                </button>
              )}
              {row.status === 'delivered' && onReject && (
                // For delivered stories, expose Reject (red) right next
                // to Accept (green). The Board wires this to open the
                // detail panel pre-flighted into the reject-reason form
                // so the PM never bypasses the "rejection carries a
                // reason" rule.
                <button
                  className="btn-mini danger"
                  aria-label="Reject"
                  draggable={false}
                  onMouseDown={e => e.stopPropagation()}
                  onClick={e => { e.stopPropagation(); onReject(); }}
                >
                  Reject
                </button>
              )}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function formatReleaseDate(row: OrderRow): string {
  const raw = row.release_date;
  if (!raw) return '';
  const d = new Date(raw);
  if (isNaN(d.getTime())) return raw;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

function isReleasePast(row: OrderRow): boolean {
  const raw = row.release_date;
  if (!raw) return false;
  const d = new Date(raw);
  if (isNaN(d.getTime())) return false;
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return d.getTime() <= today.getTime();
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

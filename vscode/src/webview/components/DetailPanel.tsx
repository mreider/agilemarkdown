import * as React from 'react';
import { useEffect, useState } from 'react';
import { Detail } from '../App';
import { TypeGlyph, StatePill, Avatar, Icon, typeEmoji } from './icons';
import { vscode } from '../vscode-api';
import { AutocompleteInput } from './AutocompleteInput';

const STATE_ORDER = ['unstarted', 'started', 'finished', 'delivered', 'accepted'];

export function DetailPanel({ detail, onClose, candidates }: {
  detail: Detail;
  onClose(): void;
  // Autocomplete pools sourced from the current snapshot. Owners
  // and tags come from every visible story's frontmatter; epic
  // candidates come from the EpicProgressResult slugs the snapshot
  // already aggregates. Optional so tests can render without a
  // candidate set.
  candidates?: { owners: string[]; tags: string[]; epics: string[] };
}) {
  const item = detail.item;
  const [estimate, setEstimate] = useState(item.estimate || '');
  const [tagsInput, setTagsInput] = useState((item.tags || []).join(', '));
  const [assignInput, setAssignInput] = useState((item.assignees || []).join(', '));
  const [epicInput, setEpicInput] = useState(item.epic || '');
  const [comment, setComment] = useState('');
  const [taskDraft, setTaskDraft] = useState('');
  const [editingDesc, setEditingDesc] = useState(false);
  const [descDraft, setDescDraft] = useState(item.body);
  const [confirmingAccept, setConfirmingAccept] = useState(false);
  const [rejecting, setRejecting] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [confirmingBlock, setConfirmingBlock] = useState(false);
  const [blockReason, setBlockReason] = useState('');

  useEffect(() => {
    setEstimate(item.estimate || '');
    setTagsInput((item.tags || []).join(', '));
    setAssignInput((item.assignees || []).join(', '));
    setEpicInput(item.epic || '');
    setDescDraft(item.body);
    setEditingDesc(false);
    setConfirmingAccept(false);
    // autoAction === 'reject' lands the panel directly in the
    // reject-reason form. Defaults to false (no auto-open) so normal
    // open-detail flows behave exactly as before.
    const wantsReject = detail.autoAction === 'reject';
    setRejecting(wantsReject);
    setRejectReason('');
    setConfirmingDelete(false);
    setConfirmingBlock(false);
    setBlockReason('');
  }, [item.path]);

  const stateIdx = STATE_ORDER.indexOf(item.status);
  const isReleaseLike = item.type === 'release';

  function send(type: string, payload: any) {
    vscode?.postMessage({ type, payload });
  }

  // The three inline ceremonies (Accept, Reject, Delete) are mutually
  // exclusive, opening one must close any other that was already
  // visible. Without these helpers, clicking Reject while the Accept
  // ceremony was open left both rendered on top of each other.
  function openAccept() {
    setRejecting(false);
    setConfirmingDelete(false);
    setConfirmingBlock(false);
    setConfirmingAccept(true);
  }
  function openReject() {
    setConfirmingAccept(false);
    setConfirmingDelete(false);
    setConfirmingBlock(false);
    setRejecting(true);
  }
  function openDelete() {
    setConfirmingAccept(false);
    setRejecting(false);
    setConfirmingBlock(false);
    setConfirmingDelete(true);
  }
  function openBlock() {
    setConfirmingAccept(false);
    setRejecting(false);
    setConfirmingDelete(false);
    setBlockReason('');
    setConfirmingBlock(true);
  }

  function commitEstimate(v: string) {
    if (v.trim() === (item.estimate || '')) return;
    send('set-estimate', { path: item.path, estimate: v.trim() });
  }
  function commitAssignees() {
    const xs = assignInput.split(',').map(s => s.trim()).filter(Boolean);
    send('set-assigned', { path: item.path, assignees: xs });
  }
  function commitTags() {
    const xs = tagsInput.split(',').map(s => s.trim()).filter(Boolean);
    send('set-tags', { path: item.path, tags: xs });
  }
  function commitEpic() {
    send('set-epic', { path: item.path, epic: epicInput.trim() });
  }

  return (
    <>
      <div className="detail-overlay open" onClick={onClose} />
      <aside className="detail-panel open">
        <div className="detail-head">
          <TypeGlyph type={item.type} />
          <span className="story-id" title={item.path}>{shorten(item.path)}</span>
          <div style={{ marginLeft: 'auto', display: 'flex', gap: 4 }}>
            <button
              className="icon-btn detail-delete"
              title="Delete story"
              aria-label="Delete story"
              onClick={openDelete}
            >
              <Icon.Trash />
            </button>
            <button className="icon-btn" onClick={onClose}><Icon.X /></button>
          </div>
        </div>
        <div className="detail-body">
          <h1 className="detail-title">{item.title}</h1>
          <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap', marginBottom: 14 }}>
            {item.epic && <span className="label-chip">{item.epic}</span>}
            {(item.tags || []).map(t => <span key={t} className="label-chip">{t}</span>)}
            {item.blocked
              ? <button className="story-blocked" onClick={() => send('unblock', { path: item.path })}><Icon.Block /> Blocked · click to clear</button>
              : <button className="btn" onClick={openBlock}><Icon.Block /> Block</button>}
          </div>

          {!isReleaseLike && (
            <div className="state-machine">
              {STATE_ORDER.map((st, i) => {
                // Apply `s-<state>` to every step (not just the
                // current one) so the per-state button color hint
                // fires for transition targets. The chip styling for
                // the current state lives in `.state-step.current.s-<state>`.
                const positionCls = i === stateIdx ? 'current' : i < stateIdx ? 'done' : '';
                const cls = `${positionCls} s-${st}`;
                return (
                  <button
                    key={st}
                    className={`state-step ${cls}`}
                    onClick={() => {
                      if (st === 'accepted') {
                        // Render the PM ceremony before flipping. The
                        // criteria + diff give the human a moment to
                        // pause; the seam is the point of agilemarkdown.
                        openAccept();
                        return;
                      }
                      send('set-status', { path: item.path, status: st });
                    }}
                  >
                    {st}
                  </button>
                );
              })}
              {/* The rejected state-step follows the same color
                  pattern as every other step: tinted button when not
                  current, solid red chip when the story is actually
                  rejected. Earlier this button carried a permanent
                  `danger` class that kept it always-red, which made
                  it visually inconsistent with the rest of the row. */}
              <button
                className={`state-step s-rejected ${item.status === 'rejected' ? 'current' : ''}`}
                onClick={openReject}
              >
                rejected
              </button>
            </div>
          )}

          {confirmingAccept && (
            <div className="ceremony" role="dialog" aria-label="PM acceptance ceremony">
              <div className="ceremony-head">Accept <strong>{item.title}</strong>?</div>
              <div className="ceremony-actions">
                <button className="btn-mini" onClick={() => setConfirmingAccept(false)}>Cancel</button>
                <button className="btn-mini danger" onClick={openReject}>Reject</button>
                <button className="btn-mini success" onClick={() => { send('set-status', { path: item.path, status: 'accepted' }); setConfirmingAccept(false); }}>Accept</button>
              </div>
            </div>
          )}

          {confirmingDelete && (
            <div className="ceremony" role="alertdialog" aria-label="Confirm delete">
              <div className="ceremony-head">Delete <strong>{item.title}</strong>?</div>
              <div className="ceremony-actions">
                <button
                  className="btn-mini"
                  onClick={() => setConfirmingDelete(false)}
                >Cancel</button>
                <button
                  className="btn-mini danger"
                  onClick={() => {
                    send('delete-item', { path: item.path });
                    setConfirmingDelete(false);
                  }}
                >Delete</button>
              </div>
            </div>
          )}

          {confirmingBlock && (
            <div className="ceremony" role="dialog" aria-label="Block with reason">
              <div className="ceremony-head">Block</div>
              <textarea
                className="reject-reason"
                value={blockReason}
                onChange={e => setBlockReason(e.target.value)}
                placeholder="Reason (optional)"
                autoFocus
              />
              <div className="ceremony-actions">
                <button
                  className="btn-mini"
                  onClick={() => { setConfirmingBlock(false); setBlockReason(''); }}
                >Cancel</button>
                <button
                  className="btn-mini primary"
                  onClick={() => {
                    send('block', { path: item.path, reason: blockReason.trim() || undefined });
                    setConfirmingBlock(false);
                    setBlockReason('');
                  }}
                >Block</button>
              </div>
            </div>
          )}

          {rejecting && (
            <div className="ceremony" role="dialog" aria-label="Reject with reason">
              <div className="ceremony-head">Reject</div>
              <textarea
                className="reject-reason"
                value={rejectReason}
                onChange={e => setRejectReason(e.target.value)}
                placeholder="Reason"
                autoFocus
              />
              <div className="ceremony-actions">
                <button className="btn-mini" onClick={() => { setRejecting(false); setRejectReason(''); }}>Cancel</button>
                <button
                  className="btn-mini danger"
                  disabled={rejectReason.trim().length === 0}
                  onClick={() => { send('reject', { path: item.path, reason: rejectReason.trim() }); setRejecting(false); setRejectReason(''); }}
                >Reject</button>
              </div>
            </div>
          )}

          <div className="field-grid">
            <div className="field-label">State</div>
            <div className="field-value"><StatePill state={item.status} /></div>

            <div className="field-label">Type</div>
            <div className="field-value">
              {/* feature/bug/chore picker. Release type is intentionally
                  omitted from the UI; releases are date markers and
                  flipping a regular story to release would skip the
                  state machine. Flipping to bug or chore clears the
                  estimate (the extension's set-type handler enforces
                  the coach rule). */}
              <div className="type-picker">
                {(['feature', 'bug', 'chore'] as const).map(t => (
                  <button
                    key={t}
                    className={`type-chip type-${t} ${(item.type || 'feature') === t ? 'active' : ''}`}
                    onClick={() => send('set-type', { path: item.path, type: t })}
                  >
                    <span className="type-chip-emoji" aria-hidden="true">{typeEmoji(t)}</span> {t}
                  </button>
                ))}
              </div>
            </div>

            {/* Hide the Points row for bugs and chores since they are
                not estimated. The coach rule lives server-side too
                (the extension's set-type handler strips the estimate
                when flipping), but suppressing the picker here
                prevents the user from setting one in the first place. */}
            {(item.type || 'feature') === 'feature' && (
              <>
                <div className="field-label">Points</div>
                <div className="field-value">
                  <div className="points-picker">
                    {[0, 1, 2, 3, 5, 8].map(n => (
                      <button
                        key={n}
                        className={`points-pip ${(item.estimate || '') === String(n) ? 'active' : ''}`}
                        onClick={() => { setEstimate(String(n)); commitEstimate(String(n)); }}
                      >
                        {n === 0 ? '·' : n}
                      </button>
                    ))}
                  </div>
                </div>
              </>
            )}

            <div className="field-label">Owners</div>
            <div className="field-value">
              <AutocompleteInput
                value={assignInput}
                onChange={setAssignInput}
                onCommit={commitAssignees}
                candidates={candidates?.owners || []}
                mode="multi"
                placeholder="alice, bob"
                ariaLabel="Owners"
              />
              <div className="owner-row">
                {(item.assignees || []).map(a => <Avatar key={a} id={a} size={22} />)}
              </div>
            </div>

            <div className="field-label">Tags</div>
            <div className="field-value">
              <AutocompleteInput
                value={tagsInput}
                onChange={setTagsInput}
                onCommit={commitTags}
                candidates={candidates?.tags || []}
                mode="multi"
                placeholder="auth, q2"
                ariaLabel="Tags"
              />
            </div>

            <div className="field-label">Epic</div>
            <div className="field-value">
              <AutocompleteInput
                value={epicInput}
                onChange={setEpicInput}
                onCommit={commitEpic}
                candidates={candidates?.epics || []}
                mode="single"
                placeholder="(unset)"
                ariaLabel="Epic"
              />
            </div>

            {item.author && (
              <>
                <div className="field-label">Reporter</div>
                <div className="field-value">
                  <Avatar id={item.author} size={22} />
                  <span style={{ marginLeft: 6 }}>{item.author}</span>
                </div>
              </>
            )}

            {(item.iteration || item.iteration_label) && (
              <>
                <div className="field-label">Iteration</div>
                <div className="field-value">
                  {item.iteration ? (
                    <>
                      <span style={{ fontFamily: 'var(--font-mono)' }}>#{item.iteration}</span>
                      {item.iteration_label && <span style={{ color: 'var(--ink-3)', marginLeft: 6 }}>· {item.iteration_label}</span>}
                    </>
                  ) : (
                    <span style={{ color: 'var(--ink-3)' }}>{item.iteration_label}</span>
                  )}
                </div>
              </>
            )}
          </div>

          <div className="section-h">
            Description
            {!editingDesc && <button className="btn-mini primary" style={{ marginLeft: 8 }} onClick={() => { setDescDraft(item.body); setEditingDesc(true); }}>Edit</button>}
            {editingDesc && (
              <>
                <button className="btn-mini primary" style={{ marginLeft: 8 }} onClick={() => { send('set-description', { path: item.path, body: descDraft }); setEditingDesc(false); }}>Save</button>
                <button className="btn-mini" style={{ marginLeft: 4 }} onClick={() => { setDescDraft(item.body); setEditingDesc(false); }}>Cancel</button>
              </>
            )}
          </div>
          {editingDesc ? (
            <textarea className="desc-edit" value={descDraft} onChange={e => setDescDraft(e.target.value)} />
          ) : (
            <pre className="desc-view">{item.body}</pre>
          )}

          <div className="section-h">
            Tasks <span style={{ color: 'var(--ink-4)', fontWeight: 500, marginLeft: 4 }}>
              {detail.tasks.tasks.filter(t => t.done).length} of {detail.tasks.count}
            </span>
          </div>
          <div className="tasks-list">
            {detail.tasks.tasks.map(t => (
              <div key={t.index} className="task-row">
                <button
                  className={`task-check ${t.done ? 'done' : ''}`}
                  onClick={() => send('tick-task', { path: item.path, index: t.index, done: !t.done })}
                >
                  {t.done && <Icon.Check />}
                </button>
                <div className={`task-text ${t.done ? 'done' : ''}`}>{t.text}</div>
              </div>
            ))}
            <div className="task-row">
              <span className="task-check" />
              <input
                className="inline-input"
                placeholder="New task"
                value={taskDraft}
                onChange={e => setTaskDraft(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter' && taskDraft.trim()) {
                    send('add-task', { path: item.path, text: taskDraft.trim() });
                    setTaskDraft('');
                  }
                }}
              />
            </div>
          </div>

          <div className="section-h">
            Comments <span style={{ color: 'var(--ink-4)', fontWeight: 500, marginLeft: 4 }}>{detail.comments.count}</span>
          </div>
          {detail.comments.comments.map((c, i) => (
            <div key={i} className="comment">
              <Avatar id={c.author || 'user'} size={28} />
              <div>
                <div>
                  <span className="comment-name">{c.author || 'user'}</span>
                  {c.when && <span className="comment-when">{c.when}</span>}
                </div>
                <div className="comment-body">{c.text}</div>
              </div>
            </div>
          ))}
          <div className="comment add">
            <input
              className="inline-input"
              placeholder="Add a comment"
              value={comment}
              onChange={e => setComment(e.target.value)}
              onKeyDown={e => {
                if (e.key === 'Enter' && comment.trim()) {
                  send('add-comment', { path: item.path, text: comment.trim() });
                  setComment('');
                }
              }}
            />
          </div>

          {detail.history && detail.history.entries.length > 0 && (
            <>
              <div className="section-h">
                History <span style={{ color: 'var(--ink-4)', fontWeight: 500, marginLeft: 4 }}>{detail.history.count}</span>
              </div>
              <ul className="history-list">
                {detail.history.entries.map(h => (
                  <li key={h.hash} className="history-entry">
                    <div className="history-row1">
                      <span className="history-author">{h.author}</span>
                      <span className="history-when">{formatHistoryDate(h.when)}</span>
                    </div>
                    <div className="history-subject">{h.subject}</div>
                    <div className="history-hash">{h.hash.slice(0, 7)}</div>
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>
      </aside>
    </>
  );
}

function formatHistoryDate(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) + ' ' +
      d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' });
  } catch {
    return iso;
  }
}

function shorten(p: string): string {
  return p.length > 40 ? '…' + p.slice(-39) : p;
}

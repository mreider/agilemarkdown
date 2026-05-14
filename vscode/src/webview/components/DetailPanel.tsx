import * as React from 'react';
import { useEffect, useState } from 'react';
import { Detail } from '../App';
import { TypeGlyph, StatePill, Avatar, Icon } from './icons';
import { vscode } from '../vscode-api';

const STATE_ORDER = ['unstarted', 'started', 'finished', 'delivered', 'accepted'];

export function DetailPanel({ detail, onClose }: { detail: Detail; onClose(): void }) {
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

  // Pull bullets from the body's `## Acceptance` section. The PM
  // ceremony reads from this; we display the same list here so the
  // dev pair sees what they will be checked on.
  const acceptanceBullets = extractAcceptance(item.body);

  useEffect(() => {
    setEstimate(item.estimate || '');
    setTagsInput((item.tags || []).join(', '));
    setAssignInput((item.assignees || []).join(', '));
    setEpicInput(item.epic || '');
    setDescDraft(item.body);
    setEditingDesc(false);
    setConfirmingAccept(false);
    setRejecting(false);
    setRejectReason('');
  }, [item.path]);

  const stateIdx = STATE_ORDER.indexOf(item.status);
  const isReleaseLike = item.type === 'release';

  function send(type: string, payload: any) {
    vscode?.postMessage({ type, payload });
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
              : <button className="btn" onClick={() => {
                  const reason = prompt('Block reason (optional)') ?? undefined;
                  send('block', { path: item.path, reason });
                }}><Icon.Block /> Block</button>}
          </div>

          {!isReleaseLike && (
            <div className="state-machine">
              {STATE_ORDER.map((st, i) => {
                const cls = i < stateIdx ? 'done' : i === stateIdx ? `current s-${st}` : '';
                return (
                  <button
                    key={st}
                    className={`state-step ${cls}`}
                    onClick={() => {
                      if (st === 'accepted') {
                        // Render the PM ceremony before flipping. The
                        // criteria + diff give the human a moment to
                        // pause; the seam is the point of agilemarkdown.
                        setConfirmingAccept(true);
                        return;
                      }
                      send('set-status', { path: item.path, status: st });
                    }}
                  >
                    {st}
                  </button>
                );
              })}
              <button
                className={`state-step ${item.status === 'rejected' ? 'current s-rejected' : ''}`}
                onClick={() => setRejecting(true)}
              >
                rejected
              </button>
            </div>
          )}

          {confirmingAccept && (
            <div className="ceremony" role="dialog" aria-label="PM acceptance ceremony">
              <div className="ceremony-head">As PM, do you accept?</div>
              <div className="ceremony-body">
                <div className="ceremony-row"><span className="ceremony-label">Story</span> {item.title}</div>
                <div className="ceremony-row"><span className="ceremony-label">Type</span> {item.type || 'feature'}</div>
                <div className="ceremony-row"><span className="ceremony-label">Estimate</span> {item.estimate || '—'} pts</div>
                <div className="ceremony-row"><span className="ceremony-label">Status</span> {item.status} → accepted</div>
                <div className="ceremony-criteria">
                  <div className="ceremony-criteria-head">What to verify</div>
                  {acceptanceBullets.length === 0 ? (
                    <div className="ceremony-empty">(no `## Acceptance` section in body; review the diff)</div>
                  ) : (
                    <ul>{acceptanceBullets.map((b, i) => <li key={i}>{b}</li>)}</ul>
                  )}
                </div>
              </div>
              <div className="ceremony-actions">
                <button className="btn-mini success" onClick={() => { send('set-status', { path: item.path, status: 'accepted' }); setConfirmingAccept(false); }}>Yes, accept</button>
                <button className="btn-mini" onClick={() => setConfirmingAccept(false)}>Cancel</button>
              </div>
            </div>
          )}

          {rejecting && (
            <div className="ceremony" role="dialog" aria-label="Reject with reason">
              <div className="ceremony-head">Reject — record the reason</div>
              <textarea
                className="reject-reason"
                value={rejectReason}
                onChange={e => setRejectReason(e.target.value)}
                placeholder="What's wrong with the delivered work? The reason lands in `## Rejection notes` on the body."
                autoFocus
              />
              <div className="ceremony-actions">
                <button
                  className="btn-mini danger"
                  disabled={rejectReason.trim().length === 0}
                  onClick={() => { send('reject', { path: item.path, reason: rejectReason.trim() }); setRejecting(false); setRejectReason(''); }}
                >Reject with reason</button>
                <button className="btn-mini" onClick={() => { setRejecting(false); setRejectReason(''); }}>Cancel</button>
              </div>
            </div>
          )}

          <div className="field-grid">
            <div className="field-label">State</div>
            <div className="field-value"><StatePill state={item.status} /></div>

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

            <div className="field-label">Owners</div>
            <div className="field-value">
              <input
                className="inline-input"
                value={assignInput}
                onChange={e => setAssignInput(e.target.value)}
                placeholder="alice, bob"
                onBlur={commitAssignees}
                onKeyDown={e => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur(); }}
              />
              <div className="owner-row">
                {(item.assignees || []).map(a => <Avatar key={a} id={a} size={22} />)}
              </div>
            </div>

            <div className="field-label">Tags</div>
            <div className="field-value">
              <input
                className="inline-input"
                value={tagsInput}
                onChange={e => setTagsInput(e.target.value)}
                placeholder="auth, q2"
                onBlur={commitTags}
                onKeyDown={e => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur(); }}
              />
            </div>

            <div className="field-label">Epic</div>
            <div className="field-value">
              <input
                className="inline-input"
                value={epicInput}
                onChange={e => setEpicInput(e.target.value)}
                placeholder="(unset)"
                onBlur={commitEpic}
                onKeyDown={e => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur(); }}
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

          <div className="section-h">Acceptance</div>
          {acceptanceBullets.length === 0 ? (
            <div className="ceremony-empty">Add a `## Acceptance` section to the body. Without it, the PM ceremony has nothing to render at delivery.</div>
          ) : (
            <ul className="acceptance-list">
              {acceptanceBullets.map((b, i) => <li key={i}>{b}</li>)}
            </ul>
          )}

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

// extractAcceptance pulls bullets from a `## Acceptance` section in the
// item body. Mirrors the server-side extractVerifyBullets used by the
// acceptance_prompt MCP tool. Returns at most six bullets.
function extractAcceptance(body: string): string[] {
  const lines = body.split('\n');
  let start = -1;
  for (let i = 0; i < lines.length; i++) {
    const t = lines[i].trim();
    if (t.startsWith('##') && t.toLowerCase().includes('acceptance')) {
      start = i + 1;
      break;
    }
  }
  if (start < 0) return [];
  const out: string[] = [];
  for (let i = start; i < lines.length && out.length < 6; i++) {
    const t = lines[i].trim();
    if (t.startsWith('#')) break;
    if (t.startsWith('- ') || t.startsWith('* ')) {
      out.push(t.replace(/^[-*]\s+/, '').trim());
    }
  }
  return out;
}

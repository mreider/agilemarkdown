// NewStoryModal: column-agnostic create-story dialog.
//
// Types:
//   - feature / bug / chore: regular stories. Bug and chore are
//     unpointed per the coach rule.
//   - release: a date stake, not a story. Selecting release reveals a
//     date input; the resulting create-item write stamps release_date.
//
// Wire contract:
//   - title required; Enter submits, Escape cancels
//   - type defaults to feature
//   - release requires both title + date
//   - on submit, posts create-item with title + backlog. Type and
//     release_date follow-ups happen via set-type / set-frontmatter
//     in the extension if the type differs from feature.

import * as React from 'react';
import { useEffect, useRef, useState } from 'react';
import { Icon, typeEmoji } from './icons';

type NewType = 'feature' | 'bug' | 'chore' | 'release';

export function NewStoryModal({ backlog, target, onSubmit, onCancel }: {
  backlog: string;
  target: 'priority' | 'icebox';
  onSubmit(
    title: string,
    type: NewType,
    target: 'priority' | 'icebox',
    releaseDate?: string,
  ): void;
  onCancel(): void;
}) {
  const [title, setTitle] = useState('');
  const [type, setType] = useState<NewType>('feature');
  const [releaseDate, setReleaseDate] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const trimmed = title.trim();
  const needsDate = type === 'release';
  const canSubmit = trimmed.length > 0 && (!needsDate || releaseDate.length > 0);

  useEffect(() => {
    inputRef.current?.focus();
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel();
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onCancel]);

  function submit() {
    if (!canSubmit) return;
    onSubmit(trimmed, type, target, needsDate ? releaseDate : undefined);
  }

  return (
    <div className="modal-overlay" role="dialog" aria-label="New story" onClick={onCancel}>
      <div className="modal-panel" onClick={e => e.stopPropagation()}>
        <div className="modal-head">
          <div className="modal-title">New {type === 'release' ? 'release marker' : 'story'}</div>
          <button className="icon-btn" aria-label="Close" onClick={onCancel}><Icon.X /></button>
        </div>
        <div className="modal-body">
          <label className="modal-field">
            <span className="modal-label">Title</span>
            <input
              ref={inputRef}
              className="inline-input"
              value={title}
              onChange={e => setTitle(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') submit(); }}
              placeholder={type === 'release' ? 'First-frost' : 'Login flow'}
              aria-label="Story title"
            />
          </label>
          <div className="modal-field">
            <span className="modal-label">Type</span>
            <div className="type-picker">
              {(['feature', 'bug', 'chore', 'release'] as const).map(t => (
                <button
                  key={t}
                  className={`type-chip type-${t} ${type === t ? 'active' : ''}`}
                  onClick={() => setType(t)}
                ><span className="type-chip-emoji" aria-hidden="true">{typeEmoji(t)}</span> {t}</button>
              ))}
            </div>
          </div>
          {needsDate && (
            <label className="modal-field">
              <span className="modal-label">Target date</span>
              <input
                type="date"
                className="inline-input"
                value={releaseDate}
                onChange={e => setReleaseDate(e.target.value)}
                aria-label="Release target date"
              />
            </label>
          )}
        </div>
        <div className="modal-actions">
          <button className="btn-mini" onClick={onCancel}>Cancel</button>
          <button className="btn-mini primary" disabled={!canSubmit} onClick={submit}>Create</button>
        </div>
      </div>
    </div>
  );
}

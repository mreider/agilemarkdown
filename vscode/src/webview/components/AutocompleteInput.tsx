// AutocompleteInput renders a text input with a dropdown of matching
// candidates. The pattern is the classic GitHub / Linear inline
// autocomplete: type a partial value, see a dropdown of matches with
// the prefix bolded, press Tab/Enter/click to accept. Arrow keys
// navigate the dropdown; Esc dismisses.
//
// Two modes:
//   - mode="single": the whole input value is the autocomplete target
//     (used for the Epic field; a story has one epic slug).
//   - mode="multi": the input is comma-separated; autocomplete fires
//     on the partial after the last comma (used for Owners and Tags).
//
// Candidates are static for the lifetime of this component, the
// caller passes the list, no async fetching. Matches are case-
// insensitive substring; ordering puts prefix matches first so the
// most-likely intent is the default.

import * as React from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';

export function AutocompleteInput({
  value, onChange, onCommit, candidates, mode, placeholder, ariaLabel,
}: {
  value: string;
  onChange(next: string): void;
  // Fires when the user finishes editing (blur / Enter without an
  // active match). Caller commits to the wire here.
  onCommit?(value: string): void;
  candidates: string[];
  mode: 'single' | 'multi';
  placeholder?: string;
  ariaLabel?: string;
}) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);

  // Extract the partial the user is currently typing. In multi mode
  // that's the substring after the last comma; in single mode it's
  // the whole value.
  const { partial, prefix } = splitForAutocomplete(value, mode);

  const matches = useMemo(() => {
    const q = partial.trim().toLowerCase();
    if (!q) return [];
    const taken = mode === 'multi'
      ? new Set(prefix.split(',').map(s => s.trim().toLowerCase()).filter(Boolean))
      : new Set<string>();
    const hits: string[] = [];
    const lower = (s: string) => s.toLowerCase();
    // Two-pass for ranking: prefix matches first, then substring.
    for (const c of candidates) {
      if (taken.has(lower(c))) continue;
      if (lower(c).startsWith(q)) hits.push(c);
    }
    for (const c of candidates) {
      if (taken.has(lower(c))) continue;
      if (!lower(c).startsWith(q) && lower(c).includes(q)) hits.push(c);
    }
    return hits.slice(0, 8);
  }, [partial, candidates, mode, prefix]);

  // Reset highlight when the candidate list changes.
  useEffect(() => { setActive(0); }, [matches.length, partial]);

  function applyMatch(match: string) {
    if (mode === 'multi') {
      // Replace the partial after the last comma with the match,
      // then add a trailing ", " so the user can keep typing the
      // next value. No trailing comma if the candidate has spaces
      // or punctuation (rare for tags/owners but keep it simple).
      const next = `${prefix}${match}, `;
      onChange(next);
    } else {
      onChange(match);
    }
    setOpen(false);
    setActive(0);
    inputRef.current?.focus();
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (!open || matches.length === 0) {
      if (e.key === 'Enter') {
        onCommit?.(value);
      }
      return;
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActive(a => Math.min(a + 1, matches.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActive(a => Math.max(a - 1, 0));
    } else if (e.key === 'Tab' || e.key === 'Enter') {
      e.preventDefault();
      applyMatch(matches[active]);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      setOpen(false);
    }
  }

  return (
    <div className="autocomplete">
      <input
        ref={inputRef}
        className="inline-input"
        value={value}
        placeholder={placeholder}
        aria-label={ariaLabel}
        onChange={e => { onChange(e.target.value); setOpen(true); }}
        onFocus={() => setOpen(true)}
        onBlur={() => {
          // Defer so a click on a dropdown item registers before the
          // dropdown unmounts on blur.
          window.setTimeout(() => {
            setOpen(false);
            onCommit?.(value);
          }, 120);
        }}
        onKeyDown={onKeyDown}
      />
      {open && matches.length > 0 && (
        <div className="autocomplete-pop" role="listbox">
          {matches.map((m, i) => (
            <button
              key={m}
              type="button"
              role="option"
              aria-selected={i === active}
              className={`autocomplete-opt ${i === active ? 'active' : ''}`}
              // mousedown rather than click so the blur handler
              // doesn't unmount the dropdown before the pick fires.
              onMouseDown={e => { e.preventDefault(); applyMatch(m); }}
              onMouseEnter={() => setActive(i)}
            >
              {highlight(m, partial)}
            </button>
          ))}
          <div className="autocomplete-hint">Tab to accept · Esc to dismiss</div>
        </div>
      )}
    </div>
  );
}

function splitForAutocomplete(value: string, mode: 'single' | 'multi'): { partial: string; prefix: string } {
  if (mode === 'single') return { partial: value, prefix: '' };
  const i = value.lastIndexOf(',');
  if (i < 0) return { partial: value.trim(), prefix: '' };
  return {
    partial: value.slice(i + 1).trim(),
    // Keep the trailing comma + space so a re-emitted value reads
    // naturally as "alice, bob, c…".
    prefix: value.slice(0, i + 1) + (value.slice(i + 1).startsWith(' ') ? ' ' : ' '),
  };
}

function highlight(candidate: string, partial: string): React.ReactNode {
  if (!partial) return candidate;
  const idx = candidate.toLowerCase().indexOf(partial.toLowerCase());
  if (idx < 0) return candidate;
  return (
    <>
      {candidate.slice(0, idx)}
      <strong>{candidate.slice(idx, idx + partial.length)}</strong>
      {candidate.slice(idx + partial.length)}
    </>
  );
}

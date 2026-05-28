import * as React from 'react';
import { useEffect, useRef } from 'react';
import { Icon } from './icons';

// SearchBox is a controlled input in the subbar. Typing updates the
// App-level `searchQuery` state, which makes the Search column in the
// Board show/hide. Hits are filtered client-side from the existing
// snapshot (no extension round-trip), so the column updates as fast
// as the keystrokes.
//
// ⌘K / Ctrl+K focuses the input. Escape clears it.

export function SearchBox({ value, onChange }: {
  value: string;
  onChange(query: string): void;
}) {
  const inputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
        inputRef.current?.select();
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  return (
    <div className="searchbox">
      <span className="searchbox-icon"><Icon.Search /></span>
      <input
        ref={inputRef}
        className="searchbox-input"
        placeholder="Search stories"
        value={value}
        onChange={e => onChange(e.target.value)}
        onKeyDown={e => {
          if (e.key === 'Escape') { onChange(''); inputRef.current?.blur(); }
        }}
      />
      <span className="searchbox-kbd">⌘K</span>
    </div>
  );
}

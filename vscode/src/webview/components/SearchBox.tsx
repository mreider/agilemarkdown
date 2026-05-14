import * as React from 'react';
import { useEffect, useRef, useState } from 'react';
import { Icon } from './icons';
import { vscode } from '../vscode-api';
import { SearchResult, SearchHit } from '../../shared/types';

// SearchBox renders the subbar search input and a results popover.
// Behavior:
//   - ⌘K / Ctrl+K focuses the input.
//   - Typing fires `{type: 'search', payload: {query}}` to the
//     extension (debounced) and renders the returned hits.
//   - Enter on a hit opens the detail panel for that path.
//   - Escape clears the query and closes the popover.

export function SearchBox({ onSelect }: { onSelect(path: string): void }) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult | null>(null);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const debounceRef = useRef<number | null>(null);

  useEffect(() => {
    function onMsg(e: MessageEvent) {
      const m = e.data as { type: string; payload?: SearchResult };
      if (m.type === 'search-results') {
        setResults(m.payload || null);
        setActive(0);
      }
    }
    window.addEventListener('message', onMsg);
    return () => window.removeEventListener('message', onMsg);
  }, []);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
        setOpen(true);
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  function runSearch(q: string) {
    setQuery(q);
    if (debounceRef.current) window.clearTimeout(debounceRef.current);
    debounceRef.current = window.setTimeout(() => {
      vscode?.postMessage({ type: 'search', payload: { query: q } });
    }, 150);
    setOpen(q.length > 0);
  }

  function pick(hit: SearchHit) {
    onSelect(hit.path);
    setOpen(false);
    setQuery('');
    setResults(null);
  }

  const hits = results?.hits ?? [];

  return (
    <div className="searchbox">
      <span className="searchbox-icon"><Icon.Search /></span>
      <input
        ref={inputRef}
        className="searchbox-input"
        placeholder="Search stories"
        value={query}
        onChange={e => runSearch(e.target.value)}
        onFocus={() => { if (query) setOpen(true); }}
        onBlur={() => window.setTimeout(() => setOpen(false), 150)}
        onKeyDown={e => {
          if (e.key === 'Escape') { setQuery(''); setResults(null); setOpen(false); }
          else if (e.key === 'ArrowDown') { e.preventDefault(); setActive(a => Math.min(a + 1, hits.length - 1)); }
          else if (e.key === 'ArrowUp') { e.preventDefault(); setActive(a => Math.max(a - 1, 0)); }
          else if (e.key === 'Enter' && hits[active]) { e.preventDefault(); pick(hits[active]); }
        }}
      />
      <span className="searchbox-kbd">⌘K</span>
      {open && hits.length > 0 && (
        <div className="searchbox-popover">
          {hits.map((h, i) => (
            <button
              key={h.path}
              className={`searchbox-hit ${i === active ? 'active' : ''}`}
              onMouseDown={e => { e.preventDefault(); pick(h); }}
            >
              <div className="searchbox-hit-title">{h.title}</div>
              <div className="searchbox-hit-meta">{h.path}{h.status ? ` · ${h.status}` : ''}</div>
              {h.snippet && <div className="searchbox-hit-snippet">{h.snippet}</div>}
            </button>
          ))}
        </div>
      )}
      {open && query && hits.length === 0 && results && (
        <div className="searchbox-popover">
          <div className="searchbox-empty">No matches for "{query}"</div>
        </div>
      )}
    </div>
  );
}

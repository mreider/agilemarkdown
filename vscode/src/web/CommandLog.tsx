// CommandLog renders a scrolling terminal-styled view of every write
// the laboratory backend performs. Each line maps a UI action to the
// equivalent `am` CLI verb so a tutorial reader sees the connection
// between the GUI and the command line.
//
// Data source: the `/ws/log` WebSocket. Each event is a JSON object:
//
//   {ts, verb, ok, summary}
//
// The shim in web-api.ts opens the socket once and republishes events
// as window-level "am:log" CustomEvents so this component (and any
// other) can subscribe without owning the connection.

import * as React from 'react';
import { useEffect, useRef, useState } from 'react';

interface LogEvent {
  ts: string;
  verb: string;
  ok: boolean;
  summary?: string;
}

export function CommandLog() {
  const [events, setEvents] = useState<LogEvent[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const on = (e: any) => {
      const ev = e?.detail as LogEvent | undefined;
      if (!ev) return;
      setEvents((prev) => {
        const next = [...prev, ev];
        // Hard cap so the panel doesn't grow without bound. 500 lines
        // is plenty for a tutorial walkthrough.
        return next.length > 500 ? next.slice(next.length - 500) : next;
      });
    };
    window.addEventListener('am:log', on);
    return () => window.removeEventListener('am:log', on);
  }, []);

  useEffect(() => {
    // Auto-scroll the latest line into view when a new event lands.
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
  }, [events.length]);

  return (
    <div className="lab-cmdlog" ref={scrollRef}>
      {events.length === 0 && (
        <div className="lab-cmdlog-empty">
          Every action in the board echoes the equivalent <code>am</code> verb here.
        </div>
      )}
      {events.map((ev, i) => (
        <div key={i} className={`lab-cmdlog-line ${ev.ok ? 'ok' : 'fail'}`}>
          <span className="lab-cmdlog-prompt">$</span>
          <span className="lab-cmdlog-verb">{ev.verb}</span>
          <span className="lab-cmdlog-status">{ev.ok ? '✓' : '✗'}</span>
          {ev.summary ? <div className="lab-cmdlog-summary">{ev.summary}</div> : null}
        </div>
      ))}
    </div>
  );
}

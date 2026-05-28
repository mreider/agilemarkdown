// Terminal hosts an xterm.js instance wired to /ws/pty. Bytes flow
// both ways: keystrokes → PTY stdin, PTY stdout → terminal. The PTY
// is a real bash rooted at the project, visitors can run `am show
// priority`, `git log`, anything.

import * as React from 'react';
import { useEffect, useRef } from 'react';
import { Terminal as XTerm } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

export function Terminal() {
  const hostRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!hostRef.current) return;
    const term = new XTerm({
      convertEol: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
      fontSize: 12,
      theme: {
        background: '#0E1014',
        foreground: '#E6E8EE',
      },
      cursorBlink: true,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(hostRef.current);
    fit.fit();

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${proto}//${window.location.host}/ws/pty`);
    ws.binaryType = 'arraybuffer';

    ws.onmessage = (e) => {
      if (typeof e.data === 'string') {
        term.write(e.data);
      } else {
        const data = e.data as ArrayBuffer;
        term.write(new Uint8Array(data));
      }
    };
    ws.onclose = () => {
      term.write('\r\n\x1b[90m[connection closed]\x1b[0m\r\n');
    };
    ws.onerror = () => {
      term.write('\r\n\x1b[31m[connection error]\x1b[0m\r\n');
    };

    const disposeData = term.onData((d) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(d);
    });

    const onResize = () => fit.fit();
    window.addEventListener('resize', onResize);

    return () => {
      window.removeEventListener('resize', onResize);
      disposeData.dispose();
      ws.close();
      term.dispose();
    };
  }, []);

  return <div className="lab-term" ref={hostRef} />;
}

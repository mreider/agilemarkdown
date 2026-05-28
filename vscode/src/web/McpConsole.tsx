// McpConsole renders a tool dropdown + JSON argument editor + Send
// button, then shows both the raw JSON-RPC request the server sent
// and the raw response. The whole point is the wire view: a tutorial
// reader sees exactly what an MCP client (Claude Desktop, Cursor,
// your own agent) exchanges with the project, byte for byte.
//
// Data source:
//   GET  /api/mcp-tools           returns the JSON-RPC `tools/list` result
//   POST /api/mcp/:tool           body is the args object; response is
//                                 a `{request, response}` trace.

import * as React from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';

interface ToolEntry {
  name: string;
  description?: string;
  inputSchema?: any;
}

interface CallTrace {
  request: any;
  response: any;
}

export function McpConsole() {
  const [tools, setTools] = useState<ToolEntry[]>([]);
  const [selected, setSelected] = useState<string>('');
  const [argsText, setArgsText] = useState<string>('{}');
  const [trace, setTrace] = useState<CallTrace | null>(null);
  const [err, setErr] = useState<string>('');

  useEffect(() => {
    fetch('/api/mcp-tools')
      .then((r) => r.json())
      .then((res) => {
        // tools/list returns {tools: [...]}; fall back to the raw
        // shape if the server's MCP build emits something different.
        const list: ToolEntry[] = res?.tools ?? res?.result?.tools ?? [];
        setTools(list);
        if (list.length > 0) {
          setSelected(list[0].name);
        }
      })
      .catch((e) => setErr(String(e?.message || e)));
  }, []);

  const currentTool = useMemo(() => tools.find((t) => t.name === selected), [tools, selected]);

  // Seed the args editor with an empty object skeleton derived from
  // the tool's input schema. The reader can fill values in or paste
  // a real example.
  useEffect(() => {
    if (!currentTool?.inputSchema) {
      setArgsText('{}');
      return;
    }
    const skel: Record<string, any> = {};
    const props = currentTool.inputSchema?.properties || {};
    for (const k of Object.keys(props)) {
      const t = props[k]?.type;
      skel[k] = t === 'number' || t === 'integer' ? 0 : t === 'boolean' ? false : t === 'array' ? [] : '';
    }
    setArgsText(JSON.stringify(skel, null, 2));
  }, [currentTool]);

  async function send() {
    if (!selected) return;
    setErr('');
    let args: any = {};
    try {
      args = JSON.parse(argsText);
    } catch (e: any) {
      setErr('Args is not valid JSON: ' + e?.message);
      return;
    }
    try {
      const r = await fetch(`/api/mcp/${encodeURIComponent(selected)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(args),
      });
      if (!r.ok) {
        setErr(await r.text());
        return;
      }
      const t = (await r.json()) as CallTrace;
      setTrace(t);
    } catch (e: any) {
      setErr(String(e?.message || e));
    }
  }

  return (
    <div className="lab-mcp">
      <div className="lab-mcp-form">
        <label className="lab-mcp-label">Tool</label>
        <select className="lab-mcp-select" value={selected} onChange={(e) => setSelected(e.target.value)}>
          {tools.map((t) => (
            <option key={t.name} value={t.name}>{t.name}</option>
          ))}
        </select>
        {currentTool?.description && (
          <div className="lab-mcp-desc">{currentTool.description}</div>
        )}
        <label className="lab-mcp-label">Arguments</label>
        <textarea
          className="lab-mcp-args"
          value={argsText}
          onChange={(e) => setArgsText(e.target.value)}
          spellCheck={false}
          rows={6}
        />
        <button className="btn-mini primary lab-mcp-send" onClick={send}>Send</button>
        {err && <div className="lab-mcp-err">{err}</div>}
      </div>
      <div className="lab-mcp-wire">
        <WireBlock label="Request"  json={trace?.request}  />
        <WireBlock label="Response" json={trace?.response} />
      </div>
    </div>
  );
}

function WireBlock({ label, json }: { label: string; json: any }) {
  const text = json == null ? '' : JSON.stringify(json, null, 2);
  return (
    <div className="lab-mcp-block">
      <div className="lab-mcp-block-head">{label}</div>
      <pre className="lab-mcp-block-body">{text || `(no ${label.toLowerCase()} yet, pick a tool and Send)`}</pre>
    </div>
  );
}

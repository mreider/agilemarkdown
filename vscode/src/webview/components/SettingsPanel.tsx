// SettingsPanel renders an editable view of `.am/config.yaml`.
// Loads via the extension's `config-show` message on mount; saves via
// `config-save` on the Save button. Form state mirrors the Config
// struct field-for-field; unknown fields the caller may have added by
// hand round-trip through the YAML untouched because the Go side does
// JSON → Config → YAML, preserving anything the struct knows about.
//
// The Settings tab is for human-readable config the user might
// realistically want to change. Generated/derived state (velocity
// rows, tag indexes, archive paths) is not exposed; those live under
// `.am/` for tooling, not for editing.

import * as React from 'react';
import { useEffect, useState } from 'react';
import { vscode } from '../vscode-api';

interface AmConfig {
  Project?: { Name?: string };
  NewStory?: { PriorityPosition?: string; IceboxPosition?: string };
  Estimation?: { Scale?: string };
  Iteration?: { LengthWeeks?: number; StartDay?: string; Timezone?: string };
  Velocity?: { Strategy?: string; Lookback?: number; InitialVelocity?: number };
  StoryTypes?: { BugEstimable?: boolean; ChoreEstimable?: boolean };
}

export function SettingsPanel({ folderName }: { folderName?: string }) {
  const [cfg, setCfg] = useState<AmConfig | null>(null);
  const [dirty, setDirty] = useState(false);
  const [savedAt, setSavedAt] = useState(0);

  useEffect(() => {
    function onMsg(e: MessageEvent) {
      const m = e.data as { type: string; payload?: any };
      if (m.type === 'config') {
        setCfg(m.payload);
        setDirty(false);
      }
    }
    window.addEventListener('message', onMsg);
    vscode?.postMessage({ type: 'config-show' });
    return () => window.removeEventListener('message', onMsg);
  }, []);
  // Drives the placeholder on the Name field. When the user leaves
  // Name blank, the topbar falls back to the workspace folder name;
  // showing that here makes the fallback obvious instead of
  // displaying the cryptic literal "(folder name)".
  const fallbackName = folderName || '';

  function patch<K extends keyof AmConfig>(section: K, key: string, value: any) {
    setCfg(prev => {
      const next: AmConfig = { ...(prev || {}) };
      next[section] = { ...(next[section] || {}), [key]: value } as any;
      return next;
    });
    setDirty(true);
  }

  function save() {
    if (!cfg) return;
    vscode?.postMessage({ type: 'config-save', payload: cfg });
    setSavedAt(Date.now());
    setDirty(false);
  }

  if (!cfg) {
    return <div className="settings empty">Loading…</div>;
  }

  return (
    <div className="settings">
      <div className="settings-section">
        <h3>Project</h3>
        <Field label="Display name">
          <input
            className="inline-input"
            value={cfg.Project?.Name ?? ''}
            onChange={e => patch('Project', 'Name', e.target.value)}
            placeholder={fallbackName}
          />
        </Field>
        <div className="settings-hint">
          Shown in the topbar. Leave blank to use the folder name{fallbackName ? ` (${fallbackName})` : ''}.
        </div>
      </div>

      <div className="settings-section">
        <h3>New story</h3>
        <Field label="Position in Current">
          <Select
            value={cfg.NewStory?.PriorityPosition || 'bottom'}
            options={['bottom', 'top']}
            onChange={v => patch('NewStory', 'PriorityPosition', v)}
          />
        </Field>
        <Field label="Position in Icebox">
          <Select
            value={cfg.NewStory?.IceboxPosition || 'bottom'}
            options={['bottom', 'top']}
            onChange={v => patch('NewStory', 'IceboxPosition', v)}
          />
        </Field>
      </div>

      <div className="settings-section">
        <h3>Estimation</h3>
        <Field label="Scale">
          <Select
            value={cfg.Estimation?.Scale || 'fibonacci'}
            options={['fibonacci', 'linear', 'powers', 'custom']}
            onChange={v => patch('Estimation', 'Scale', v)}
          />
        </Field>
      </div>

      <div className="settings-section">
        <h3>Iteration</h3>
        <Field label="Length (weeks)">
          <input
            type="number"
            min={1}
            max={4}
            className="inline-input"
            value={cfg.Iteration?.LengthWeeks ?? 1}
            onChange={e => patch('Iteration', 'LengthWeeks', Math.max(1, Math.min(4, parseInt(e.target.value || '1', 10))))}
          />
        </Field>
        <Field label="Start day">
          <Select
            value={cfg.Iteration?.StartDay || 'monday'}
            options={['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']}
            onChange={v => patch('Iteration', 'StartDay', v)}
          />
        </Field>
        <Field label="Timezone">
          <input
            className="inline-input"
            value={cfg.Iteration?.Timezone || 'UTC'}
            onChange={e => patch('Iteration', 'Timezone', e.target.value)}
            placeholder="UTC"
          />
        </Field>
      </div>

      <div className="settings-section">
        <h3>Velocity</h3>
        <Field label="Strategy">
          <Select
            value={cfg.Velocity?.Strategy || 'rolling'}
            options={['rolling', 'strict', 'manual']}
            onChange={v => patch('Velocity', 'Strategy', v)}
          />
        </Field>
        <Field label="Lookback (iterations)">
          <input
            type="number"
            min={1}
            className="inline-input"
            value={cfg.Velocity?.Lookback ?? 3}
            onChange={e => patch('Velocity', 'Lookback', Math.max(1, parseInt(e.target.value || '3', 10)))}
          />
        </Field>
        <Field label="Initial velocity (pts)">
          <input
            type="number"
            min={0}
            className="inline-input"
            value={cfg.Velocity?.InitialVelocity ?? 10}
            onChange={e => patch('Velocity', 'InitialVelocity', Math.max(0, parseInt(e.target.value || '10', 10)))}
          />
        </Field>
      </div>

      <div className="settings-section">
        <h3>Story types</h3>
        <Field label="Bugs are estimable">
          <input
            type="checkbox"
            checked={!!cfg.StoryTypes?.BugEstimable}
            onChange={e => patch('StoryTypes', 'BugEstimable', e.target.checked)}
          />
        </Field>
        <Field label="Chores are estimable">
          <input
            type="checkbox"
            checked={!!cfg.StoryTypes?.ChoreEstimable}
            onChange={e => patch('StoryTypes', 'ChoreEstimable', e.target.checked)}
          />
        </Field>
      </div>

      <div className="settings-footer">
        <button className="btn-mini primary" disabled={!dirty} onClick={save}>Save</button>
        {!dirty && savedAt > 0 && <span className="settings-saved">Saved</span>}
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="settings-field">
      <div className="settings-label">{label}</div>
      <div className="settings-value">{children}</div>
    </div>
  );
}

function Select({ value, options, onChange }: { value: string; options: string[]; onChange(v: string): void }) {
  return (
    <select className="inline-input" value={value} onChange={e => onChange(e.target.value)}>
      {options.map(o => <option key={o} value={o}>{o}</option>)}
    </select>
  );
}

import * as React from 'react';
import { Snapshot } from '../App';
import { BurnupResult, CFDResult } from '../../shared/types';

export function Analytics({ snap }: { snap: Snapshot }) {
  const dash = snap.dashboard;
  // Acceptance rate complements rejection rate: of the stories the PM
  // looked at this iteration, what fraction were accepted on first
  // pass. Computed client-side from rejection_rate so we don't need a
  // new MCP field.
  const acceptanceRate = Math.max(0, 100 - (dash?.rejection_rate_latest_percent ?? 0));
  return (
    <div className="analytics">
      <div className="analytics-grid">
        <div className="kpi-card">
          <div className="kpi-label">Velocity (3-iter avg)</div>
          <div className="kpi-value">{Math.round(dash?.velocity ?? 0)} <span className="kpi-unit">pts</span></div>
          <div className="kpi-delta">{dash?.velocity_bootstrap ? 'bootstrap' : `volatility ${Math.round(dash?.volatility_percent ?? 0)}%`}</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-label">Acceptance rate</div>
          <div className="kpi-value">{Math.round(acceptanceRate)}<span className="kpi-unit">%</span></div>
          <div className="kpi-delta">latest iteration</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-label">Cycle time</div>
          <div className="kpi-value">{((dash?.cycle_time_median_hours ?? 0) / 24).toFixed(1)} <span className="kpi-unit">days</span></div>
          <div className="kpi-delta">median, started → accepted</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-label">Stories accepted</div>
          <div className="kpi-value">{dash?.stories_accepted_total ?? 0}</div>
          <div className="kpi-delta">all-time</div>
        </div>

        <div className="chart-card span-7">
          <div className="chart-title">Velocity</div>
          <div className="chart-sub">Planned vs accepted, last {snap.velocity.rows.length} iterations</div>
          <VelocityChart rows={snap.velocity.rows} />
        </div>

        <div className="chart-card span-5">
          <div className="chart-title">Burnup · current iteration</div>
          <BurnupView burnup={snap.burnup} />
        </div>

        <div className="chart-card span-7">
          <div className="chart-title">Cumulative flow</div>
          <div className="chart-sub">Story counts by state, last 30 days</div>
          <CumulativeFlowChart cfd={snap.cfd} />
        </div>

        <div className="chart-card span-5">
          <div className="chart-title">Story type mix</div>
          <div className="chart-sub">Accepted in lookback window</div>
          <TypeMixView snap={snap} />
        </div>
      </div>
    </div>
  );
}

function CumulativeFlowChart({ cfd }: { cfd: CFDResult | null }) {
  if (!cfd || cfd.rows.length === 0) {
    return <div className="empty">Not enough history yet.</div>;
  }
  const data = cfd.rows;
  const w = 640, h = 220, padL = 32, padR = 8, padT = 16, padB = 28;
  const max = Math.max(...data.map(d => d.accepted + d.in_flight + d.backlog), 1) + 1;
  const xs = (i: number) => padL + (i / Math.max(1, data.length - 1)) * (w - padL - padR);
  const ys = (v: number) => padT + (1 - v / max) * (h - padT - padB);
  // Three stacked areas, bottom-up: accepted, in-flight, backlog.
  const acceptedTop = data.map((d, i) => `${xs(i)},${ys(d.accepted)}`);
  const inFlightTop = data.map((d, i) => `${xs(i)},${ys(d.accepted + d.in_flight)}`);
  const totalTop = data.map((d, i) => `${xs(i)},${ys(d.accepted + d.in_flight + d.backlog)}`);
  const baseline = `${xs(data.length - 1)},${ys(0)} ${xs(0)},${ys(0)}`;
  const acceptedArea = `M ${acceptedTop.join(' L ')} L ${baseline} Z`;
  const inFlightArea = `M ${inFlightTop.join(' L ')} L ${acceptedTop.slice().reverse().join(' L ')} Z`;
  const backlogArea = `M ${totalTop.join(' L ')} L ${inFlightTop.slice().reverse().join(' L ')} Z`;
  const ticks = [0, Math.round(max / 4), Math.round(max / 2), Math.round((3 * max) / 4)];
  return (
    <svg width="100%" viewBox={`0 0 ${w} ${h}`} style={{ marginTop: 12 }}>
      {ticks.map(g => (
        <g key={g}>
          <line x1={padL} x2={w - padR} y1={ys(g)} y2={ys(g)} stroke="var(--line)" strokeDasharray="2 4" />
          <text x={padL - 6} y={ys(g) + 3} fontSize={10} fontFamily="var(--font-mono)" fill="var(--ink-4)" textAnchor="end">{g}</text>
        </g>
      ))}
      <path d={backlogArea} fill="var(--line)" />
      <path d={inFlightArea} fill="var(--st-started)" opacity={0.8} />
      <path d={acceptedArea} fill="var(--st-accepted)" opacity={0.9} />
      <text x={padL} y={h - 10} fontSize={10} fontFamily="var(--font-mono)" fill="var(--ink-4)">{data[0]?.day}</text>
      <text x={w - padR} y={h - 10} fontSize={10} fontFamily="var(--font-mono)" fill="var(--ink-4)" textAnchor="end">{data[data.length - 1]?.day}</text>
      {/* Legend */}
      <g transform={`translate(${padL + 8} ${padT + 8})`} fontSize={10} fontFamily="var(--font-mono)" fill="var(--ink-4)">
        <rect x={0} y={0} width={10} height={10} fill="var(--st-accepted)" />
        <text x={14} y={9}>accepted</text>
        <rect x={80} y={0} width={10} height={10} fill="var(--st-started)" />
        <text x={94} y={9}>in flight</text>
        <rect x={160} y={0} width={10} height={10} fill="var(--line)" />
        <text x={174} y={9}>backlog</text>
      </g>
    </svg>
  );
}

function VelocityChart({ rows }: { rows: { iteration: number; planned: number; accepted: number }[] }) {
  if (rows.length === 0) return <div className="empty">No history yet.</div>;
  const w = 640, h = 220, padL = 32, padR = 8, padT = 16, padB = 28;
  const max = Math.max(...rows.flatMap(r => [r.planned, r.accepted]), 1) + 4;
  const bw = (w - padL - padR) / rows.length;
  const ys = (v: number) => padT + (1 - v / max) * (h - padT - padB);
  const ticks = [0, Math.round(max / 4), Math.round(max / 2), Math.round((3 * max) / 4)];
  // Trend line through accepted values so over/under-delivery is
  // immediately visible without squinting at the bars.
  const trend = rows.map((d, i) => `${i === 0 ? 'M' : 'L'} ${padL + i * bw + bw / 2} ${ys(d.accepted)}`).join(' ');
  return (
    <svg width="100%" viewBox={`0 0 ${w} ${h}`} style={{ marginTop: 12 }}>
      {ticks.map(g => (
        <g key={g}>
          <line x1={padL} x2={w - padR} y1={ys(g)} y2={ys(g)} stroke="var(--line)" strokeDasharray="2 4" />
          <text x={padL - 6} y={ys(g) + 3} fontSize={10} fontFamily="var(--font-mono)" fill="var(--ink-4)" textAnchor="end">{g}</text>
        </g>
      ))}
      {rows.map((d, i) => {
        const x = padL + i * bw + bw * 0.2;
        const bw2 = bw * 0.32;
        return (
          <g key={d.iteration}>
            <rect x={x} y={ys(d.planned)} width={bw2} height={h - padB - ys(d.planned)} fill="var(--line)" rx={2} />
            <rect x={x + bw2 + 3} y={ys(d.accepted)} width={bw2} height={h - padB - ys(d.accepted)} fill="var(--accent)" rx={2} />
            <text x={padL + i * bw + bw / 2} y={h - 10} fontSize={10} fontFamily="var(--font-mono)" fill="var(--ink-4)" textAnchor="middle">{d.iteration}</text>
          </g>
        );
      })}
      <path d={trend} fill="none" stroke="var(--ink)" strokeWidth={1.2} strokeOpacity={0.4} strokeDasharray="3 3" />
    </svg>
  );
}

function BurnupView({ burnup }: { burnup: BurnupResult | null }) {
  if (!burnup || burnup.rows.length === 0) {
    return <div className="empty">No burnup data for this iteration.</div>;
  }
  const data = burnup.rows;
  const w = 460, h = 220, padL = 32, padR = 8, padT = 18, padB = 28;
  const max = Math.max(...data.flatMap(d => [d.scope, d.done]), 1) + 4;
  const xs = (i: number) => padL + (i / Math.max(1, data.length - 1)) * (w - padL - padR);
  const ys = (v: number) => padT + (1 - v / max) * (h - padT - padB);
  const scopePath = data.map((d, i) => `${i === 0 ? 'M' : 'L'} ${xs(i)} ${ys(d.scope)}`).join(' ');
  const donePath = data.map((d, i) => `${i === 0 ? 'M' : 'L'} ${xs(i)} ${ys(d.done)}`).join(' ');
  const doneArea = donePath + ` L ${xs(data.length - 1)} ${ys(0)} L ${xs(0)} ${ys(0)} Z`;
  const finalScope = data[data.length - 1]?.scope ?? 0;
  const ideal = `M ${xs(0)} ${ys(0)} L ${xs(data.length - 1)} ${ys(finalScope)}`;
  const ticks = [0, Math.round(max / 4), Math.round(max / 2), Math.round((3 * max) / 4)];
  return (
    <svg width="100%" viewBox={`0 0 ${w} ${h}`} style={{ marginTop: 12 }}>
      {ticks.map(g => (
        <g key={g}>
          <line x1={padL} x2={w - padR} y1={ys(g)} y2={ys(g)} stroke="var(--line)" strokeDasharray="2 4" />
          <text x={padL - 6} y={ys(g) + 3} fontSize={10} fontFamily="var(--font-mono)" fill="var(--ink-4)" textAnchor="end">{g}</text>
        </g>
      ))}
      <path d={doneArea} fill="var(--accent-soft)" />
      <path d={scopePath} fill="none" stroke="var(--ink)" strokeWidth={1.5} strokeOpacity={0.45} />
      <path d={donePath} fill="none" stroke="var(--accent)" strokeWidth={2} />
      <path d={ideal} fill="none" stroke="var(--st-accepted)" strokeWidth={1.2} strokeDasharray="4 3" />
      {data.map((d, i) => (
        <circle key={i} cx={xs(i)} cy={ys(d.done)} r={3} fill="var(--accent)" />
      ))}
    </svg>
  );
}

function TypeMixView({ snap }: { snap: Snapshot }) {
  const rows = snap.typeMix.rows;
  const total = snap.typeMix.total;
  if (total === 0) return <div className="empty">No accepted stories in window.</div>;
  return (
    <div className="type-mix">
      <div className="type-mix-bar">
        {rows.map(r => (
          <div key={r.type} className={`type-mix-seg type-${r.type}`} style={{ flex: r.count }} />
        ))}
      </div>
      <div className="type-mix-legend">
        {rows.map(r => (
          <div key={r.type} className="type-mix-row">
            <span className={`type-mix-swatch type-${r.type}`} />
            <span className="type-mix-label">{r.type}</span>
            <span className="type-mix-count">{r.count}</span>
            <span className="type-mix-pct">{Math.round(r.percent)}%</span>
          </div>
        ))}
      </div>
    </div>
  );
}

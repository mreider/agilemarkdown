import * as React from 'react';

export function Sparkline({ data }: { data: number[] }) {
  if (data.length === 0) return <svg width="100%" height={28} />;
  const w = 192, h = 28, pad = 2;
  const max = Math.max(...data, 1);
  const min = Math.min(...data, 0);
  const xs = (i: number) => pad + (i * (w - pad * 2)) / Math.max(1, data.length - 1);
  const ys = (v: number) => h - pad - ((v - min) / Math.max(1, max - min)) * (h - pad * 2);
  const path = data.map((v, i) => (i === 0 ? 'M' : 'L') + xs(i) + ',' + ys(v)).join(' ');
  const last = data[data.length - 1];
  return (
    <svg width="100%" height={h} viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none">
      <path d={path} fill="none" stroke="var(--accent)" strokeWidth={1.5} strokeLinejoin="round" />
      <circle cx={xs(data.length - 1)} cy={ys(last)} r={2.5} fill="var(--accent)" />
    </svg>
  );
}

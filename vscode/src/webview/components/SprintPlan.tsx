import * as React from 'react';

interface SprintPlanRow {
  index: number;
  path: string;
  title: string;
  type?: string;
  estimate?: string;
  status?: string;
  has_acceptance?: boolean;
  acceptance_count?: number;
  oversized_feature?: boolean;
  unestimated_feature?: boolean;
}

interface SprintPlanData {
  backlog: string;
  velocity: number;
  iteration_length_weeks?: number;
  committed: SprintPlanRow[];
  below_line: SprintPlanRow[];
  committed_points: number;
  overcommitted: boolean;
  warnings: string[];
}

export function SprintPlan({ data, onRefresh }: { data: SprintPlanData | null; onRefresh(): void }) {
  if (!data) {
    return (
      <div className="analytics">
        <button className="btn" onClick={onRefresh}>Load plan</button>
      </div>
    );
  }
  const fits = !data.overcommitted;
  return (
    <div className="analytics">
      <div className="plan-head">
        <div>
          <div className="chart-title">Iteration plan · {data.backlog}</div>
          <div className="chart-sub">velocity {Math.round(data.velocity)} pts / iteration · committed {Math.round(data.committed_points)} pts across {data.committed.length} stories</div>
        </div>
        <button className="btn" onClick={onRefresh}>Refresh</button>
      </div>

      {!fits && (
        <div className="plan-banner over">
          Overcommit: {Math.round(data.committed_points - data.velocity)} points above rolling velocity. Trim from the bottom of the committed set.
        </div>
      )}

      <h3 className="plan-section">Committed</h3>
      <RowList rows={data.committed} />

      <h3 className="plan-section">Below the line</h3>
      {data.below_line.length === 0 ? (
        <div className="empty">Nothing below the line.</div>
      ) : (
        <RowList rows={data.below_line} />
      )}

      {data.warnings.length > 0 && (
        <>
          <h3 className="plan-section">Warnings</h3>
          <ul className="plan-warnings">
            {data.warnings.map((w, i) => <li key={i}>{w}</li>)}
          </ul>
        </>
      )}
    </div>
  );
}

function RowList({ rows }: { rows: SprintPlanRow[] }) {
  return (
    <table className="plan-table">
      <thead>
        <tr>
          <th>#</th>
          <th>Title</th>
          <th>Type</th>
          <th>Pts</th>
          <th>Acceptance</th>
          <th>Status</th>
          <th>Flags</th>
        </tr>
      </thead>
      <tbody>
        {rows.map(r => (
          <tr key={r.path}>
            <td>{r.index + 1}</td>
            <td>{r.title}</td>
            <td>{r.type || '—'}</td>
            <td>{r.estimate || '—'}</td>
            <td>{r.has_acceptance ? `${r.acceptance_count} bullets` : <span className="plan-flag">missing</span>}</td>
            <td>{r.status}</td>
            <td>
              {r.oversized_feature && <span className="plan-flag">oversized</span>}
              {r.unestimated_feature && <span className="plan-flag">unestimated</span>}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

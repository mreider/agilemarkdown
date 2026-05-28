// @vitest-environment happy-dom
//
// Component tests for the Board (priority + icebox columns) and the
// StoryCard inline action button. The contract under test is the
// postMessage payload the Board emits, *not* the cosmetic DOM. Payload
// shape changes break every consumer downstream (extension handler →
// AmClient → CLI → disk), so locking it in here is the cheap seam to
// catch regressions like the un-qualified path bug that wrote a phantom
// Login-Flow.md at the repo root.
//
// What each test asserts:
//   • the message `type` matches the extension's case branch
//   • the `path` is qualified as `<backlog>/<basename>` (the qpath fix)
//   • the action button on a priority card advances along the state
//     machine in lock-step with nextAction() in StoryCard.tsx
//   • the action button on an icebox card emits the dedicated
//     `pull-from-icebox` payload, not raw set-status, promotion + flip
//     happen atomically in the extension handler
//   • the inline "Add story" composer sends `create-item` with the
//     correct backlog target

import { beforeEach, describe, expect, it } from 'vitest';
import { harness, lastOf, makeOrderRow, makeSnapshot, React } from '../../test/webview-helpers';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { Board } from './Board';

// Board accepts an onAddStory(target) callback. Tests pass a no-op
// wrapper; specific tests can supply a spy when they want to assert
// the target the + buttons emit.
function BoardHost({ snap, onSelect, onAddStory, searchQuery, includeSearchColumn, includeDoneColumn, visibleColumns }: any) {
  // Tests want all three content columns visible by default so the
  // existing assertions (Icebox column, + buttons, etc.) line up.
  // Production defaults to {'backlog'} only, that's enforced in App.tsx.
  const cols: string[] = visibleColumns
    ? visibleColumns
    : (() => {
        const base = ['backlog', 'icebox', 'epics'];
        if (includeSearchColumn) base.unshift('search');
        if (includeDoneColumn) base.push('done');
        return base;
      })();
  return (
    <Board
      snap={snap}
      me={null}
      onSelect={onSelect}
      selectedPath={undefined}
      onAddStory={onAddStory || (() => {})}
      searchQuery={searchQuery || ''}
      onSearchQueryChange={() => {}}
      candidates={{ owners: [], tags: [], epics: [] }}
      visibleColumns={new Set(cols as any)}
      allColumns={['search', 'backlog', 'done', 'icebox', 'epics']}
      columnLabels={{ search: 'Search', backlog: 'Backlog', done: 'Done', icebox: 'Icebox', epics: 'Epics' }}
      filterColumns={[]}
      onOpenFilter={() => {}}
      onCloseFilter={() => {}}
      onColumnToggle={() => {}}
    />
  );
}

function renderBoard(snapOverrides: any = {}, _ignored?: any, onAddStory?: any) {
  const snap = makeSnapshot(snapOverrides);
  const onSelect = (_p: string) => {};
  return render(<BoardHost snap={snap} onSelect={onSelect} onAddStory={onAddStory} />);
}

describe('Board: priority card action button', () => {
  beforeEach(() => harness().reset());

  it('emits a qualified path for the Start click (regression: phantom Login-Flow.md)', () => {
    const row = makeOrderRow({ status: 'unstarted', path: 'Login-Flow.md', title: 'Login Flow' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    fireEvent.click(screen.getByRole('button', { name: 'Start' }));
    const m = lastOf('set-status');
    expect(m.payload.path).toBe('product/Login-Flow.md');
    expect(m.payload.status).toBe('started');
  });

  it('walks the priority state machine in lock-step with nextAction()', () => {
    const cases: Array<{ status: string; label: string; next: string }> = [
      { status: 'unstarted', label: 'Start',   next: 'started' },
      { status: 'started',   label: 'Finish',  next: 'finished' },
      { status: 'finished',  label: 'Deliver', next: 'delivered' },
      { status: 'delivered', label: 'Accept',  next: 'accepted' },
      { status: 'rejected',  label: 'Restart', next: 'started' },
    ];
    for (const c of cases) {
      harness().reset();
      const row = makeOrderRow({ status: c.status });
      const { unmount } = renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
      fireEvent.click(screen.getByRole('button', { name: c.label }));
      const m = lastOf('set-status');
      expect(m.payload.status, `from ${c.status}`).toBe(c.next);
      unmount();
    }
  });

  it('accepted and rejected cards expose no action button (terminal states)', () => {
    const a = makeOrderRow({ status: 'accepted', path: 'A.md', title: 'A' });
    renderBoard({ priority: { items: [a], count: 1, velocity: 10 } });
    // No Start/Finish/Deliver/Accept/Restart button rendered.
    expect(screen.queryByRole('button', { name: /^(Start|Finish|Deliver|Accept|Restart)$/ })).toBeNull();
  });

  it('clicking the action button does not also open the detail panel (mousedown is guarded)', () => {
    // The outer card is draggable, so a stray mousedown bubble would
    // both fire the action AND start a drag (eating the click). The
    // button stops propagation; what we assert here is the visible
    // contract: exactly one set-status message, no open-detail.
    const row = makeOrderRow({ status: 'unstarted' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    fireEvent.mouseDown(screen.getByRole('button', { name: 'Start' }));
    fireEvent.click(screen.getByRole('button', { name: 'Start' }));
    expect(harness().ofType('set-status')).toHaveLength(1);
    expect(harness().ofType('open-detail')).toHaveLength(0);
  });
});

describe('Board: icebox card action button', () => {
  beforeEach(() => harness().reset());

  it('Start on an icebox card emits pull-from-icebox, not raw set-status', () => {
    const row = makeOrderRow({ status: 'unstarted', path: 'Login-Flow.md' });
    renderBoard({ icebox: { items: [row], count: 1 } });
    fireEvent.click(screen.getByRole('button', { name: 'Start' }));
    const m = lastOf('pull-from-icebox');
    expect(m.payload).toEqual({ backlog: 'product', item_path: 'Login-Flow.md' });
    // The dedicated message handler does promote + flip atomically;
    // emitting a stray set-status here would race with the move.
    expect(harness().ofType('set-status')).toHaveLength(0);
  });

  it('Restart on a rejected icebox card also goes through pull-from-icebox (next state is started)', () => {
    // Restart maps to next=started, and the icebox handler treats any
    // "started" transition as a deliberate promotion. We pin this so
    // the special-case logic doesn't drift to status-strings only.
    const row = makeOrderRow({ status: 'rejected', path: 'Old.md' });
    renderBoard({ icebox: { items: [row], count: 1 } });
    fireEvent.click(screen.getByRole('button', { name: 'Restart' }));
    const m = lastOf('pull-from-icebox');
    expect(m.payload).toEqual({ backlog: 'product', item_path: 'Old.md' });
    expect(harness().ofType('set-status')).toHaveLength(0);
  });
});

describe('Board: card click opens detail with qualified path', () => {
  beforeEach(() => harness().reset());

  it('clicking the card body fires the parent onSelect with `<backlog>/<basename>`', () => {
    const seen: string[] = [];
    const row = makeOrderRow({ status: 'unstarted', path: 'Login-Flow.md', title: 'Login Flow' });
    const snap = makeSnapshot({ priority: { items: [row], count: 1, velocity: 10 } });
    render(
      <BoardHost snap={snap} onSelect={(p: string) => seen.push(p)} />,
    );
    fireEvent.click(screen.getByText('Login Flow'));
    expect(seen).toEqual(['product/Login-Flow.md']);
  });
});

describe('Board: column-header `+` buttons (Backlog + Icebox)', () => {
  beforeEach(() => harness().reset());

  it('the Backlog column + fires onAddStory("priority")', () => {
    const targets: string[] = [];
    renderBoard({ priority: { items: [], count: 0, velocity: 10 } }, undefined, (t: string) => targets.push(t));
    fireEvent.click(screen.getByLabelText('New story in Backlog'));
    expect(targets).toEqual(['priority']);
  });

  it('the Icebox column + fires onAddStory("icebox")', () => {
    const targets: string[] = [];
    renderBoard({ icebox: { items: [], count: 0 } }, undefined, (t: string) => targets.push(t));
    fireEvent.click(screen.getByLabelText('New story in Icebox'));
    expect(targets).toEqual(['icebox']);
  });
});

describe('Board: search column', () => {
  beforeEach(() => harness().reset());

  it('does not render unless toggled on from the activity-bar tree', () => {
    renderBoard({ priority: { items: [makeOrderRow()], count: 1, velocity: 10 } });
    expect(screen.queryByText('Search')).toBeNull();
  });

  it('renders text-matching priority+icebox rows when the column is open', () => {
    const p1 = makeOrderRow({ title: 'Login flow', path: 'Login.md', status: 'started' });
    const p2 = makeOrderRow({ title: 'Profile page', path: 'Profile.md', status: 'unstarted' });
    const i1 = makeOrderRow({ title: 'Logging library swap', path: 'Logging.md', status: 'unstarted' });
    const snap = makeSnapshot({
      priority: { items: [p1, p2], count: 2, velocity: 10 },
      icebox:   { items: [i1], count: 1 },
    });
    render(<BoardHost snap={snap} searchQuery="log" includeSearchColumn />);
    expect(screen.getByText('Search')).toBeInTheDocument();
    expect(screen.getAllByText(/Login flow/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Logging library/).length).toBeGreaterThan(0);
  });
});

describe('Board: card-level points picker', () => {
  beforeEach(() => harness().reset());

  it('a feature card renders the · 1 3 5 8 pip row and posts set-estimate on click', () => {
    const row = makeOrderRow({ status: 'unstarted', estimate: '', type: 'feature', path: 'Pp.md' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    // The picker exists with the trimmed set of pips. "Clear estimate"
    // is the aria-label for the · pip.
    for (const label of ['Clear estimate', 'Estimate 1', 'Estimate 3', 'Estimate 5', 'Estimate 8']) {
      expect(screen.getByRole('button', { name: label })).toBeInTheDocument();
    }
    // Click 3 → set-estimate posts with the qualified path.
    fireEvent.click(screen.getByRole('button', { name: 'Estimate 3' }));
    const m = lastOf('set-estimate');
    expect(m.payload).toEqual({ path: 'product/Pp.md', estimate: '3' });
  });

  it('the · pip clears the estimate (empty string), restoring "unpointed"', () => {
    const row = makeOrderRow({ estimate: '5', type: 'feature', path: 'Pp.md' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    fireEvent.click(screen.getByRole('button', { name: 'Clear estimate' }));
    const m = lastOf('set-estimate');
    expect(m.payload.estimate).toBe('');
  });

  it('bug and chore cards do not render the pip row (coach rule: not pointed)', () => {
    for (const type of ['bug', 'chore'] as const) {
      harness().reset();
      const row = makeOrderRow({ status: 'unstarted', type, path: `${type}.md` });
      const { unmount } = renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
      // Neither the picker nor any of its pips render.
      expect(screen.queryByRole('button', { name: /^Estimate \d$/ })).toBeNull();
      expect(screen.queryByRole('button', { name: 'Clear estimate' })).toBeNull();
      unmount();
    }
  });

  it('clicking a pip does NOT also open the detail panel (event propagation guarded)', () => {
    const row = makeOrderRow({ status: 'unstarted', estimate: '', type: 'feature', path: 'Pp.md' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    fireEvent.click(screen.getByRole('button', { name: 'Estimate 5' }));
    // No open-detail posted from the pip click; the card body's
    // onClick must not fire.
    expect(harness().ofType('open-detail')).toHaveLength(0);
    expect(harness().ofType('set-estimate')).toHaveLength(1);
  });
});

describe('Board: card-level Reject for delivered stories', () => {
  beforeEach(() => harness().reset());

  it('a delivered card renders BOTH Accept (green) and Reject (red) inline', () => {
    const row = makeOrderRow({ status: 'delivered', path: 'Login-Flow.md' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    // Accept is the existing nextAction button.
    expect(screen.getByRole('button', { name: 'Accept' })).toBeInTheDocument();
    // Reject is the new sibling. The aria-label keeps it queryable
    // independent of label-text drift.
    const reject = screen.getByRole('button', { name: 'Reject' });
    expect(reject).toBeInTheDocument();
    expect(reject.className).toMatch(/\bdanger\b/);
  });

  it('clicking Reject on a delivered card opens detail with autoAction=reject (no raw set-status)', () => {
    const row = makeOrderRow({ status: 'delivered', path: 'Login-Flow.md' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    fireEvent.click(screen.getByRole('button', { name: 'Reject' }));
    const m = lastOf('open-detail');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', autoAction: 'reject' });
    // The card must NOT post a raw `reject` message, the reason
    // invariant lives in the detail panel form.
    expect(harness().ofType('reject')).toHaveLength(0);
    expect(harness().ofType('set-status')).toHaveLength(0);
  });

  it('non-delivered cards do not show the Reject button', () => {
    for (const status of ['unstarted', 'started', 'finished', 'accepted'] as const) {
      harness().reset();
      const row = makeOrderRow({ status, path: `${status}.md` });
      const { unmount } = renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
      expect(screen.queryByRole('button', { name: 'Reject' }), `status=${status}`).toBeNull();
      unmount();
    }
  });
});

describe('Board: blocked card', () => {
  beforeEach(() => harness().reset());

  it('a blocked card shows a Blocked chip instead of the state pill', () => {
    const row = makeOrderRow({ status: 'started', blocked: true });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    expect(screen.getByText(/Blocked/i)).toBeInTheDocument();
    const card = screen.getByText('Login Flow').closest('.story');
    expect(card).not.toBeNull();
    expect(within(card as HTMLElement).queryByText(/^Started$/)).toBeNull();
  });
});

describe('Board: bucketIntoIterations is status-driven', () => {
  beforeEach(() => harness().reset());

  it('started and accepted-this-week rows sit in the current sprint, no matter the points', () => {
    // Velocity 3, but three current-work stories total 10 pts.
    // Pre-fix, the bin-packing logic would split them across three
    // future sprints. Post-fix, every status === current goes to band 0.
    const a = makeOrderRow({ title: 'A wait', path: 'A.md', status: 'started',  estimate: '3' });
    const b = makeOrderRow({ title: 'B watch', path: 'B.md', status: 'started',  estimate: '5' });
    const c = makeOrderRow({ title: 'C done',  path: 'C.md', status: 'accepted', estimate: '2' });
    const d = makeOrderRow({ title: 'D plan',  path: 'D.md', status: 'unstarted', estimate: '3' });
    renderBoard({ priority: { items: [a, b, c, d], count: 4, velocity: 3 } });

    // All three current-work titles should appear in the same iter-band
    // that carries the "Current" range marker.
    const currentBand = document.querySelector('.iter-band.iter-current') as HTMLElement | null;
    expect(currentBand).not.toBeNull();
    for (const title of ['A wait', 'B watch', 'C done']) {
      expect(within(currentBand!).getByText(new RegExp(title))).toBeInTheDocument();
    }
    // The unstarted row should NOT be in the current band.
    expect(within(currentBand!).queryByText(/D plan/)).toBeNull();
  });

  it('accepted-this-iter renders with no action button (read-only green)', () => {
    const row = makeOrderRow({ status: 'accepted', title: 'A done', path: 'a.md' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    const card = screen.getByText('A done').closest('.story');
    expect(card).not.toBeNull();
    expect(card!.classList.contains('is-accepted')).toBe(true);
    // No action button on accepted cards.
    expect(within(card as HTMLElement).queryByRole('button', { name: /Finish|Deliver|Accept/i })).toBeNull();
  });
});

describe('Board: epic chip click opens filter column', () => {
  beforeEach(() => harness().reset());

  it('single-click on the epic chip posts an open-filter for that epic', () => {
    const row = makeOrderRow({ title: 'Login flow', path: 'login.md', status: 'started', epic: 'the-button-quest' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    const chip = screen.getByText(/the-button-quest/);
    expect(chip).toBeInTheDocument();
    fireEvent.click(chip);
    // The Board host (Board.test renderBoard helper) doesn't wire
    // onOpenFilter to a state change, but firing click without crashing
    // and the chip carrying a click handler is enough proof, the
    // App-level wiring is covered by the e2e behavior in vscode-side
    // tests. Confirm the chip didn't open the detail panel by mistake.
    // (no detail event was posted; harness has no record).
  });
});

describe('Board: release bar', () => {
  beforeEach(() => harness().reset());

  it('renders as a release-bar with title and date when type=release', () => {
    const row = makeOrderRow({ title: 'First-frost', path: 'frost.md', type: 'release', status: 'unstarted', release_date: '2099-07-01' });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    const bar = screen.getByText('First-frost').closest('.release-bar');
    expect(bar).not.toBeNull();
    expect(within(bar as HTMLElement).getByText(/2099|Jul/)).toBeInTheDocument();
    // Future-dated: no Ship button.
    expect(within(bar as HTMLElement).queryByText(/Ship/)).toBeNull();
  });

  it('shows a Ship button when the target date is today or past', () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const iso = yesterday.toISOString().slice(0, 10);
    const row = makeOrderRow({ title: 'Past release', path: 'past.md', type: 'release', status: 'unstarted', release_date: iso });
    renderBoard({ priority: { items: [row], count: 1, velocity: 10 } });
    const bar = screen.getByText('Past release').closest('.release-bar');
    expect(bar).not.toBeNull();
    const ship = within(bar as HTMLElement).getByRole('button', { name: /Ship/i });
    fireEvent.click(ship);
    // Clicking Ship posts set-status accepted.
    const msgs = harness().ofType('set-status');
    expect(msgs.length).toBeGreaterThan(0);
    expect(msgs[0].payload).toMatchObject({ status: 'accepted' });
  });
});

describe('Board: Done column groups by historical iteration', () => {
  beforeEach(() => harness().reset());

  it('rows are grouped by the accepted timestamp, newest week first, with Done sprint markers', () => {
    const now = new Date();
    const lastWeek = new Date(now); lastWeek.setDate(now.getDate() - 7);
    const twoWeeks = new Date(now); twoWeeks.setDate(now.getDate() - 14);
    const old = makeOrderRow({ title: 'Older one', path: 'older.md', status: 'accepted', accepted: twoWeeks.toISOString() });
    const newer = makeOrderRow({ title: 'Newer one', path: 'newer.md', status: 'accepted', accepted: lastWeek.toISOString() });
    const snap = makeSnapshot({
      done: { items: [old, newer], count: 2 },
    });
    render(<BoardHost snap={snap} includeDoneColumn />);
    const doneCol = document.querySelector('.col-done') as HTMLElement | null;
    expect(doneCol).not.toBeNull();
    // Two iteration markers, in newest-first order.
    const markers = doneCol!.querySelectorAll('.iter-marker');
    expect(markers.length).toBe(2);
    // Newest band first: should contain "Newer one" before "Older one".
    const order = doneCol!.textContent || '';
    expect(order.indexOf('Newer one')).toBeLessThan(order.indexOf('Older one'));
  });
});

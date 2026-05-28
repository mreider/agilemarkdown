// @vitest-environment happy-dom
//
// Component tests for the DetailPanel. The detail panel is the second
// surface that emits postMessage payloads (the first is the Board).
// Every field control sends its own message type; getting the path
// qualified and the type right is what keeps the on-disk story in
// sync with the visible UI.
//
// The path under test is always `product/Login-Flow.md` (qualified)
// because the open-detail flow in App.tsx qualifies before posting and
// the server echoes args.Path on the GetItemResult. A regression to
// un-qualified paths would show up immediately as `Login-Flow.md` in
// the assertions below.

import { beforeEach, describe, expect, it } from 'vitest';
import { harness, lastOf, makeDetail, React } from '../../test/webview-helpers';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { DetailPanel } from './DetailPanel';

function renderDetail(over: any = {}) {
  return render(<DetailPanel detail={makeDetail(over)} onClose={() => {}} />);
}

describe('DetailPanel: state-machine buttons', () => {
  beforeEach(() => harness().reset());

  it('clicking "started" posts set-status with status=started and qualified path', () => {
    renderDetail({ item: { status: 'unstarted' } });
    fireEvent.click(screen.getByRole('button', { name: 'started' }));
    const m = lastOf('set-status');
    expect(m.payload.path).toBe('product/Login-Flow.md');
    expect(m.payload.status).toBe('started');
  });

  it('clicking "finished" then "delivered" posts each transition in order', () => {
    renderDetail({ item: { status: 'started' } });
    fireEvent.click(screen.getByRole('button', { name: 'finished' }));
    fireEvent.click(screen.getByRole('button', { name: 'delivered' }));
    const flips = harness().ofType('set-status').map(m => m.payload.status);
    expect(flips).toEqual(['finished', 'delivered']);
  });

  it('clicking "accepted" opens the PM ceremony first instead of flipping directly', () => {
    renderDetail({ item: { status: 'delivered' } });
    fireEvent.click(screen.getByRole('button', { name: 'accepted' }));
    expect(harness().ofType('set-status')).toHaveLength(0);
    const dialog = screen.getByRole('dialog', { name: /PM acceptance ceremony/i });
    // "Yes, accept" is the seam. Clicking it should post set-status.
    fireEvent.click(within(dialog).getByRole('button', { name: /Accept/i }));
    const m = lastOf('set-status');
    expect(m.payload.status).toBe('accepted');
  });

  it('renders feature/bug/chore type chips; clicking one posts set-type', () => {
    renderDetail({ item: { type: 'feature', status: 'unstarted', estimate: '' } });
    for (const t of ['feature', 'bug', 'chore'] as const) {
      expect(screen.getByRole('button', { name: t })).toBeInTheDocument();
    }
    fireEvent.click(screen.getByRole('button', { name: 'bug' }));
    const m = lastOf('set-type');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', type: 'bug' });
  });

  it('hides the Points row when the type is bug or chore (coach rule: not pointed)', () => {
    for (const type of ['bug', 'chore'] as const) {
      const { unmount } = renderDetail({ item: { type, status: 'unstarted', estimate: '' } });
      // The "Points" label is gone from the field grid. The detail
      // panel's Acceptance criteria field also uses a different label,
      // so the lookup stays narrow.
      expect(screen.queryByText(/^Points$/)).toBeNull();
      unmount();
    }
  });

  it('shows the Points row for features (default)', () => {
    renderDetail({ item: { type: 'feature', status: 'unstarted', estimate: '' } });
    expect(screen.getByText(/^Points$/)).toBeInTheDocument();
  });

  it('Block button opens an inline ceremony instead of triggering a JS prompt()', () => {
    // Regression: the Block button used to call window.prompt(),
    // which bounces the user out to the OS-level prompt dialog. The
    // inline ceremony keeps the input in the webview and submits a
    // `block` message with the trimmed reason.
    renderDetail({ item: { blocked: false, status: 'started' } });
    fireEvent.click(screen.getByRole('button', { name: /^Block$/ }));
    const dialog = screen.getByRole('dialog', { name: /Block with reason/i });
    expect(dialog).toBeInTheDocument();
    expect(harness().ofType('block')).toHaveLength(0);

    const textarea = within(dialog).getByPlaceholderText(/Reason/i);
    fireEvent.change(textarea, { target: { value: 'waiting on design' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Block' }));
    const m = lastOf('block');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', reason: 'waiting on design' });
    expect(screen.queryByRole('dialog', { name: /Block with reason/i })).toBeNull();
  });

  it('Block ceremony Cancel dismisses without posting', () => {
    renderDetail({ item: { blocked: false, status: 'started' } });
    fireEvent.click(screen.getByRole('button', { name: /^Block$/ }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(harness().ofType('block')).toHaveLength(0);
  });

  it('opening Block closes any other ceremony that was already open (mutually exclusive)', () => {
    renderDetail({ item: { status: 'delivered', blocked: false } });
    fireEvent.click(screen.getByRole('button', { name: 'accepted' }));
    expect(screen.getByRole('dialog', { name: /PM acceptance ceremony/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /^Block$/ }));
    expect(screen.queryByRole('dialog', { name: /PM acceptance ceremony/i })).toBeNull();
    expect(screen.getByRole('dialog', { name: /Block with reason/i })).toBeInTheDocument();
  });

  it('opening Reject while the Accept ceremony is open closes the Accept ceremony (mutually exclusive)', () => {
    // Regression: the three inline ceremonies (Accept, Reject,
    // Delete) used to be independent booleans, so clicking Reject in
    // the Accept ceremony left BOTH dialogs stacked on top of each
    // other. The user reported seeing both at once after rapid
    // clicks. The openAccept/openReject/openDelete helpers now
    // clear the others.
    renderDetail({ item: { status: 'delivered' } });
    fireEvent.click(screen.getByRole('button', { name: 'accepted' }));
    expect(screen.getByRole('dialog', { name: /PM acceptance ceremony/i })).toBeInTheDocument();
    // Click the Reject button inside the ceremony.
    const acceptDialog = screen.getByRole('dialog', { name: /PM acceptance ceremony/i });
    fireEvent.click(within(acceptDialog).getByRole('button', { name: /^Reject$/ }));
    // Accept dialog gone, reject dialog visible. Never both.
    expect(screen.queryByRole('dialog', { name: /PM acceptance ceremony/i })).toBeNull();
    expect(screen.getByRole('dialog', { name: /Reject with reason/i })).toBeInTheDocument();
  });

  it('opening Delete closes any other ceremony that was open', () => {
    renderDetail({ item: { status: 'delivered' } });
    // Open the Accept ceremony first.
    fireEvent.click(screen.getByRole('button', { name: 'accepted' }));
    expect(screen.getByRole('dialog', { name: /PM acceptance ceremony/i })).toBeInTheDocument();
    // Then open Delete. Accept should close.
    fireEvent.click(screen.getByRole('button', { name: 'Delete story' }));
    expect(screen.queryByRole('dialog', { name: /PM acceptance ceremony/i })).toBeNull();
    expect(screen.getByRole('alertdialog', { name: /Confirm delete/i })).toBeInTheDocument();
  });

  it('every state-step carries an s-<state> class so the per-state button color fires (chip when current, tint otherwise)', () => {
    // Regression: rejected used to carry a permanent `danger` class
    // that made it always-red, visually inconsistent with the rest of
    // the row. All state-steps now follow the same contract: s-<state>
    // applied unconditionally, `current` applied only on the matching
    // state. CSS picks chip-style (solid fill) or button-style (tint)
    // from there.
    renderDetail({ item: { status: 'started' } });
    for (const st of ['unstarted', 'started', 'finished', 'delivered', 'accepted', 'rejected']) {
      const btn = screen.getByRole('button', { name: st });
      expect(btn.className).toMatch(new RegExp(`\\bs-${st}\\b`));
      // `danger` is no longer a permanent class.
      expect(btn.className).not.toMatch(/\bdanger\b/);
    }
    // The current state-step also carries `current`.
    expect(screen.getByRole('button', { name: 'started' }).className).toMatch(/\bcurrent\b/);
    expect(screen.getByRole('button', { name: 'rejected' }).className).not.toMatch(/\bcurrent\b/);
  });

  it('opening the panel with detail.autoAction=reject lands directly in the reject-reason form', async () => {
    // The card-level Reject button posts open-detail with
    // autoAction=reject. The extension passes that through into the
    // detail payload. The panel mounts straight into the reject form
    // so the PM doesn't have to navigate to it manually.
    const detail = makeDetail({
      item: { path: 'product/Pp.md', title: 'Pp', status: 'delivered' },
      autoAction: 'reject',
    } as any);
    render(<DetailPanel detail={detail} onClose={() => {}} />);
    // findBy waits for the useEffect that flips the rejecting flag
    // after mount. getBy would race the initial paint.
    await screen.findByRole('dialog', { name: /Reject with reason/i });
    // No reject posted yet; the form awaits a reason.
    expect(harness().ofType('reject')).toHaveLength(0);
  });

  it('opening the panel without autoAction does NOT auto-open the reject form (default behavior preserved)', () => {
    const detail = makeDetail({ item: { path: 'product/Pp.md', title: 'Pp', status: 'delivered' } });
    render(<DetailPanel detail={detail} onClose={() => {}} />);
    expect(screen.queryByRole('dialog', { name: /Reject with reason/i })).toBeNull();
  });

  it('the acceptance ceremony exposes a Reject (red) button alongside Yes, accept', () => {
    // PM contract: a delivered story can be accepted OR rejected in
    // the same ceremony, without making the PM leave and click the
    // separate "rejected" state step. The Reject button switches into
    // the reason form (the same flow as the state-machine rejected
    // step) so the rejection always carries a reason.
    renderDetail({ item: { status: 'delivered' } });
    fireEvent.click(screen.getByRole('button', { name: 'accepted' }));
    const dialog = screen.getByRole('dialog', { name: /PM acceptance ceremony/i });
    const rejectBtn = within(dialog).getByRole('button', { name: /^Reject$/ });
    // The reject button carries the danger class so it visually reads
    // as the red counterpart to the green Yes, accept.
    expect(rejectBtn.className).toMatch(/\bdanger\b/);
    fireEvent.click(rejectBtn);
    // Acceptance ceremony closes, reject-reason form opens. No
    // set-status posted because the PM hasn't supplied a reason yet.
    expect(screen.queryByRole('dialog', { name: /PM acceptance ceremony/i })).toBeNull();
    expect(screen.getByRole('dialog', { name: /Reject with reason/i })).toBeInTheDocument();
    expect(harness().ofType('set-status')).toHaveLength(0);
    expect(harness().ofType('reject')).toHaveLength(0);
  });

  it('cancel on the acceptance ceremony closes it without posting set-status', () => {
    renderDetail({ item: { status: 'delivered' } });
    fireEvent.click(screen.getByRole('button', { name: 'accepted' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(harness().ofType('set-status')).toHaveLength(0);
    expect(screen.queryByRole('dialog', { name: /PM acceptance ceremony/i })).toBeNull();
  });

  it('clicking "rejected" opens the reason form; submission posts `reject`, not `set-status`', () => {
    renderDetail({ item: { status: 'delivered' } });
    fireEvent.click(screen.getByRole('button', { name: 'rejected' }));
    const dialog = screen.getByRole('dialog', { name: /Reject with reason/i });
    const textarea = within(dialog).getByPlaceholderText(/Reason/i);
    // Submit is disabled while the reason is empty.
    const submit = within(dialog).getByRole('button', { name: /^Reject$/ });
    expect(submit).toBeDisabled();
    fireEvent.change(textarea, { target: { value: 'missed timezone case' } });
    expect(submit).not.toBeDisabled();
    fireEvent.click(submit);
    const m = lastOf('reject');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', reason: 'missed timezone case' });
    expect(harness().ofType('set-status')).toHaveLength(0);
  });

  it('release-type stories hide the state machine entirely', () => {
    renderDetail({ item: { type: 'release', status: 'unstarted' } });
    expect(screen.queryByRole('button', { name: 'started' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'finished' })).toBeNull();
  });
});

describe('DetailPanel: field controls', () => {
  beforeEach(() => harness().reset());

  it('points pip click commits set-estimate with the picked value', () => {
    renderDetail();
    fireEvent.click(screen.getByRole('button', { name: '5' }));
    const m = lastOf('set-estimate');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', estimate: '5' });
  });

  it('"·" pip (0) commits an empty-string estimate (unestimated)', () => {
    renderDetail({ item: { estimate: '3' } });
    fireEvent.click(screen.getByRole('button', { name: '·' }));
    const m = lastOf('set-estimate');
    // The button writes "0", which the CLI accepts and the renderer
    // shows as "-" via the unestimated path. Locking the wire here
    // protects the "click 0 to clear" gesture.
    expect(m.payload.estimate).toBe('0');
  });

  it('owners input commits on blur with a parsed assignee list', async () => {
    renderDetail();
    const input = screen.getByPlaceholderText('alice, bob') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'alice, bob' } });
    fireEvent.blur(input);
    // AutocompleteInput defers commit by 120ms so a dropdown-option
    // mousedown can fire first. waitFor catches the post after.
    await new Promise(r => setTimeout(r, 160));
    const m = lastOf('set-assigned');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', assignees: ['alice', 'bob'] });
  });

  it('tags input commits on blur with a parsed tag list', async () => {
    renderDetail();
    const input = screen.getByPlaceholderText('auth, q2') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'auth, q2, login' } });
    fireEvent.blur(input);
    await new Promise(r => setTimeout(r, 160));
    const m = lastOf('set-tags');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', tags: ['auth', 'q2', 'login'] });
  });

  it('epic input commits the slug on blur', async () => {
    renderDetail();
    const input = screen.getByPlaceholderText('(unset)') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'auth-rewrite' } });
    fireEvent.blur(input);
    await new Promise(r => setTimeout(r, 160));
    const m = lastOf('set-epic');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', epic: 'auth-rewrite' });
  });

  it('Enter in an inline-input triggers commit (via blur deferred 120ms)', async () => {
    renderDetail();
    const input = screen.getByPlaceholderText('auth, q2') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'auth' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    fireEvent.blur(input);
    await new Promise(r => setTimeout(r, 160));
    expect(lastOf('set-tags').payload.tags).toEqual(['auth']);
  });
});

describe('DetailPanel: block/unblock', () => {
  beforeEach(() => harness().reset());

  it('Block button opens the inline ceremony; submitting posts `block` with the reason', () => {
    // Previous version invoked window.prompt() and asserted the
    // payload. The inline ceremony now collects the reason in a
    // textarea inside the panel; the OS-level prompt is gone.
    renderDetail();
    fireEvent.click(screen.getByRole('button', { name: /^Block$/ }));
    const dialog = screen.getByRole('dialog', { name: /Block with reason/i });
    const textarea = within(dialog).getByPlaceholderText(/Reason/i);
    fireEvent.change(textarea, { target: { value: 'waiting on design' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Block' }));
    const m = lastOf('block');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', reason: 'waiting on design' });
  });

  it('an already-blocked card surfaces an unblock affordance instead of Block', () => {
    renderDetail({ item: { blocked: true } });
    fireEvent.click(screen.getByRole('button', { name: /Blocked.*click to clear/i }));
    const m = lastOf('unblock');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md' });
  });
});

describe('DetailPanel: tasks', () => {
  beforeEach(() => harness().reset());

  it('Enter on the new-task input emits add-task with the trimmed text', () => {
    renderDetail();
    const input = screen.getByPlaceholderText('New task') as HTMLInputElement;
    fireEvent.change(input, { target: { value: '  wire telemetry  ' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    const m = lastOf('add-task');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', text: 'wire telemetry' });
  });

  it('clicking a task checkbox toggles done via tick-task', () => {
    renderDetail({
      tasks: {
        count: 1,
        tasks: [{ index: 0, text: 'wire telemetry', done: false }],
      },
    });
    // The task checkbox is the empty button rendered next to the text.
    const taskText = screen.getByText('wire telemetry');
    const row = taskText.closest('.task-row') as HTMLElement;
    const check = within(row).getByRole('button');
    fireEvent.click(check);
    const m = lastOf('tick-task');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', index: 0, done: true });
  });

  it('clicking a done task posts done=false (untick)', () => {
    renderDetail({
      tasks: {
        count: 1,
        tasks: [{ index: 7, text: 'roll back feature flag', done: true }],
      },
    });
    const row = screen.getByText('roll back feature flag').closest('.task-row') as HTMLElement;
    fireEvent.click(within(row).getByRole('button'));
    expect(lastOf('tick-task').payload).toEqual({
      path: 'product/Login-Flow.md', index: 7, done: false,
    });
  });
});

describe('DetailPanel: comments', () => {
  beforeEach(() => harness().reset());

  it('Enter on the comment input emits add-comment with the trimmed text', () => {
    renderDetail();
    const input = screen.getByPlaceholderText('Add a comment') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'looks good to me' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(lastOf('add-comment').payload).toEqual({
      path: 'product/Login-Flow.md', text: 'looks good to me',
    });
  });
});

describe('DetailPanel: description editor', () => {
  beforeEach(() => harness().reset());

  it('Save commits the draft body via set-description and closes edit mode', () => {
    renderDetail({ item: { body: 'original body' } });
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    const textarea = screen.getByDisplayValue('original body') as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: 'rewritten body' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    const m = lastOf('set-description');
    expect(m.payload).toEqual({ path: 'product/Login-Flow.md', body: 'rewritten body' });
    // After save, the readonly <pre> should be visible again.
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
  });

  it('Cancel discards the draft without sending set-description', () => {
    renderDetail({ item: { body: 'original body' } });
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    const textarea = screen.getByDisplayValue('original body') as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: 'never mind' } });
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(harness().ofType('set-description')).toHaveLength(0);
  });
});

describe('DetailPanel: input state resets when item changes', () => {
  beforeEach(() => harness().reset());

  it('switching to a different item path re-hydrates the inline inputs from the new item', () => {
    const { rerender } = render(<DetailPanel detail={makeDetail({ item: { tags: ['a', 'b'] } })} onClose={() => {}} />);
    const tagsInput = screen.getByPlaceholderText('auth, q2') as HTMLInputElement;
    expect(tagsInput.value).toBe('a, b');

    rerender(
      <DetailPanel
        detail={makeDetail({
          item: { path: 'product/Other-Story.md', tags: ['x'] },
        })}
        onClose={() => {}}
      />,
    );
    const next = screen.getByPlaceholderText('auth, q2') as HTMLInputElement;
    expect(next.value).toBe('x');
  });
});

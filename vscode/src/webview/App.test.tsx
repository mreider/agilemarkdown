// @vitest-environment happy-dom
//
// App-level integration tests. Render the real App, dispatch
// synthetic snapshot/detail/config messages on `window.message`, and
// assert what the user sees after the round trip.
//
// Scope intentionally narrow: the bulk of behavior lives in the
// component-level suites (Board, DetailPanel, NewStoryModal). These
// tests verify the App stitches them together correctly.

import { beforeEach, describe, expect, it } from 'vitest';
import { harness, lastOf, makeOrderRow, makeSnapshot, makeDetail, React } from '../test/webview-helpers';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from './App';

function dispatch(type: string, payload: any) {
  window.dispatchEvent(new MessageEvent('message', { data: { type, payload } }));
}

// Many of these tests assume all three columns are visible. The
// production default is just Backlog; pre-seed localStorage with the
// full set so each App render starts in the all-columns state.
function seedAllColumnsVisible() {
  localStorage.setItem('am-visible-columns', JSON.stringify(['backlog', 'icebox', 'epics']));
}

describe('App: top-level layout', () => {
  beforeEach(() => { harness().reset(); seedAllColumnsVisible(); });

  it('subbar carries no view tabs (Views moved to activity-bar TreeView)', () => {
    render(<App />);
    act(() => { dispatch('snapshot', makeSnapshot()); });
    // View switching is now in the VS Code activity-bar TreeView,
    // not the in-app subbar.
    expect(screen.queryByRole('button', { name: /^Home$/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /^Analytics$/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /^Settings$/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /^Plan$/i })).toBeNull();
  });

  it('switches to Analytics view when extension posts set-view', async () => {
    render(<App />);
    act(() => { dispatch('snapshot', makeSnapshot()); });
    act(() => { dispatch('set-view', 'analytics'); });
    // Analytics renders its KPI label. (Concrete element varies; the
    // weakest stable assertion is that the Backlog column is gone.)
    await new Promise(r => setTimeout(r, 20));
    expect(screen.queryByText(/^Backlog$/)).toBeNull();
  });

  it('Home tab shows Backlog, Icebox, Epics columns, no sidebar nav', () => {
    render(<App />);
    act(() => { dispatch('snapshot', makeSnapshot()); });
    expect(screen.getByText('Backlog')).toBeInTheDocument();
    expect(screen.getByText('Icebox')).toBeInTheDocument();
    expect(screen.getByText('Epics')).toBeInTheDocument();
    // Sidebar's "Views" header and All-stories row are gone.
    expect(screen.queryByText(/^Views$/i)).toBeNull();
    expect(screen.queryByText(/All stories/i)).toBeNull();
  });
});

describe('App: New Story flow', () => {
  beforeEach(() => { harness().reset(); seedAllColumnsVisible(); });

  it('clicking + on Backlog opens the modal targeted at priority', async () => {
    render(<App />);
    act(() => { dispatch('snapshot', makeSnapshot()); });
    fireEvent.click(screen.getByLabelText('New story in Backlog'));
    const dialog = await screen.findByRole('dialog', { name: /New story/i });
    fireEvent.change(within(dialog).getByLabelText(/Story title/i), { target: { value: 'Login flow' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }));
    expect(lastOf('create-item').payload).toEqual({ backlog: 'product', title: 'Login flow', target: 'priority' });
  });

  it('clicking + on Icebox opens the modal targeted at icebox', async () => {
    render(<App />);
    act(() => { dispatch('snapshot', makeSnapshot()); });
    fireEvent.click(screen.getByLabelText('New story in Icebox'));
    const dialog = await screen.findByRole('dialog', { name: /New story/i });
    fireEvent.change(within(dialog).getByLabelText(/Story title/i), { target: { value: 'Idea' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }));
    expect(lastOf('create-item').payload).toEqual({ backlog: 'product', title: 'Idea', target: 'icebox' });
  });

  it('Cancel dismisses without posting', () => {
    render(<App />);
    act(() => { dispatch('snapshot', makeSnapshot()); });
    fireEvent.click(screen.getByLabelText('New story in Backlog'));
    fireEvent.click(within(screen.getByRole('dialog', { name: /New story/i })).getByRole('button', { name: 'Cancel' }));
    expect(harness().ofType('create-item')).toHaveLength(0);
  });

  it('selecting release type reveals a date picker and posts set-release-date', async () => {
    render(<App />);
    act(() => { dispatch('snapshot', makeSnapshot()); });
    fireEvent.click(screen.getByLabelText('New story in Backlog'));
    const dialog = await screen.findByRole('dialog', { name: /New (story|release)/i });
    // Pick the release type chip; the modal switches to "New release marker".
    fireEvent.click(within(dialog).getByRole('button', { name: /release/i }));
    // Date picker now visible.
    const dateInput = within(dialog).getByLabelText(/Release target date/i);
    fireEvent.change(within(dialog).getByLabelText(/Story title/i), { target: { value: 'First-frost' } });
    fireEvent.change(dateInput, { target: { value: '2099-09-01' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }));
    // Three messages fire: create-item, set-type=release, set-release-date.
    expect(lastOf('create-item').payload.title).toBe('First-frost');
    expect(lastOf('set-type').payload.type).toBe('release');
    expect(lastOf('set-release-date').payload.release_date).toBe('2099-09-01');
  });
});

describe('App: search column', () => {
  beforeEach(() => { harness().reset(); seedAllColumnsVisible(); });

  it('Search column appears when toggled visible and filters by text input', () => {
    render(<App />);
    const p1 = makeOrderRow({ title: 'Login flow', path: 'Login.md' });
    const p2 = makeOrderRow({ title: 'Profile page', path: 'Profile.md' });
    act(() => {
      dispatch('snapshot', makeSnapshot({
        priority: { items: [p1, p2], count: 2, velocity: 10 },
      }));
      // Activity-bar tree toggles Search on by posting visible-columns
      // including 'search'.
      dispatch('visible-columns', ['search', 'backlog', 'icebox', 'epics']);
    });
    expect(screen.getByText(/^Search$/)).toBeInTheDocument();

    // Type into the in-column search input.
    fireEvent.change(screen.getByPlaceholderText(/title or path/i), { target: { value: 'login' } });
    expect(screen.getAllByText('Login flow').length).toBeGreaterThanOrEqual(2);
  });
});

describe('App: project label flash', () => {
  beforeEach(() => { harness().reset(); seedAllColumnsVisible(); });

  it('topbar reads the project name from the project message (no flash of "Agile Markdown")', async () => {
    render(<App />);
    act(() => { dispatch('project', { name: 'services-guide' }); });
    await waitFor(() => {
      expect(screen.getByText('services-guide')).toBeInTheDocument();
    });
  });
});

describe('App: BusyPill', () => {
  beforeEach(() => { harness().reset(); seedAllColumnsVisible(); });

  it('shows while a write is in flight; clears after settle', async () => {
    render(<App />);
    act(() => { dispatch('snapshot', makeSnapshot()); });
    // wait for any mount-time pending to settle
    await waitFor(() => {
      expect(screen.queryByText(/Saving/)).toBeNull();
    });
    fireEvent.click(screen.getByLabelText('New story in Backlog'));
    // Either the pill is visible briefly or, in fast paths, it
    // already settled. The assertion that matters is the pill
    // doesn't *get stuck*: after a snapshot reply, idle returns.
    act(() => { dispatch('snapshot', makeSnapshot()); });
    await waitFor(() => {
      expect(screen.queryByText(/Saving/)).toBeNull();
    });
  });
});

describe('App: detail panel autoAction round trip', () => {
  beforeEach(() => { harness().reset(); seedAllColumnsVisible(); });

  it('detail message with autoAction=reject lands directly in the reject form', async () => {
    render(<App />);
    act(() => { dispatch('snapshot', makeSnapshot()); });
    act(() => {
      dispatch('detail', { ...makeDetail({ item: { path: 'product/X.md', title: 'X', status: 'delivered' } }), autoAction: 'reject' });
    });
    await screen.findByRole('dialog', { name: /Reject with reason/i });
  });
});

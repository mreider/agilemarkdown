// @vitest-environment happy-dom
//
// Component tests for the DetailPanel's Delete affordance. The wire
// contract under test:
//   - The header carries a Delete button (icon-button, aria-label).
//   - Clicking Delete does NOT immediately post delete-item; it opens
//     an inline "Are you sure?" ceremony quoting the story title.
//   - Cancel returns to the panel without posting anything.
//   - Confirm posts `delete-item` with the story's qualified path.
//
// The modal gate is the whole point of the feature. A regression that
// drops the gate (e.g. someone wires the button directly to send())
// would silently nuke files, so the "click button → no post" leg of
// these tests is load-bearing.

import { beforeEach, describe, expect, it } from 'vitest';
import { harness, lastOf, makeDetail, React } from '../../test/webview-helpers';
import { fireEvent, render, screen } from '@testing-library/react';
import { DetailPanel } from './DetailPanel';

function renderPanel(over: any = {}) {
  const detail = makeDetail({
    item: { path: 'product/Pp.md', title: 'Pp', status: 'unstarted', ...(over.item || {}) },
    ...over,
  });
  return render(<DetailPanel detail={detail} onClose={() => {}} />);
}

describe('DetailPanel: Delete affordance', () => {
  beforeEach(() => harness().reset());

  it('renders a Delete button in the header', () => {
    renderPanel();
    expect(screen.getByRole('button', { name: 'Delete story' })).toBeInTheDocument();
  });

  it('clicking Delete does NOT immediately post; it opens the confirm ceremony', () => {
    renderPanel();
    fireEvent.click(screen.getByRole('button', { name: 'Delete story' }));
    // Modal visible, no message posted yet. The gate is the feature.
    expect(screen.getByRole('alertdialog', { name: /confirm delete/i })).toBeInTheDocument();
    expect(harness().ofType('delete-item')).toHaveLength(0);
  });

  it('confirm ceremony quotes the story title', () => {
    renderPanel({ item: { path: 'product/Login-Flow.md', title: 'Login Flow', status: 'unstarted' } });
    fireEvent.click(screen.getByRole('button', { name: 'Delete story' }));
    // Title appears in the prompt body so the user can verify which
    // story they're about to nuke.
    const dialog = screen.getByRole('alertdialog', { name: /confirm delete/i });
    expect(dialog.textContent).toContain('Login Flow');
  });

  it('Cancel dismisses the ceremony without posting delete-item', () => {
    renderPanel();
    fireEvent.click(screen.getByRole('button', { name: 'Delete story' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('alertdialog', { name: /confirm delete/i })).toBeNull();
    expect(harness().ofType('delete-item')).toHaveLength(0);
  });

  it('confirming posts delete-item with the story\'s qualified path', () => {
    renderPanel({ item: { path: 'product/Pp.md', title: 'Pp', status: 'unstarted' } });
    fireEvent.click(screen.getByRole('button', { name: 'Delete story' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    const out = lastOf('delete-item');
    expect(out.payload).toEqual({ path: 'product/Pp.md' });
    // Exactly one post, and the modal closed.
    expect(harness().ofType('delete-item')).toHaveLength(1);
    expect(screen.queryByRole('alertdialog', { name: /confirm delete/i })).toBeNull();
  });
});

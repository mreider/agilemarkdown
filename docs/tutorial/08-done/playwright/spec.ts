import { test } from '@playwright/test';
import { waitForBoard, captureBoard, setVisibleColumns } from '../../playwright/harness';

test('Chapter 8 captures', async ({ page }) => {
  await setVisibleColumns(page, ['backlog', 'done']);
  await waitForBoard(page);

  // The Done column (accepted stories from prior iterations) beside
  // the backlog. Board region only, no bottom panel.
  await captureBoard(page, '01-done-column');
});

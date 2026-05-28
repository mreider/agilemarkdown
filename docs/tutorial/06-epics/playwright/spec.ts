// Chapter 6: epics. The three moments are three epic states, so each
// image frames the specific epic card it talks about: a finished epic
// (100%, no ETA), an in-flight epic with a projected ETA, and the
// can't-project case (one story in the-moon-watch epic carries no
// estimate so the card refuses to fake a date).

import { test } from '@playwright/test';
import { waitForBoard, captureRegion, setVisibleColumns } from '../../playwright/harness';

test('Chapter 6 captures', async ({ page }) => {
  await setVisibleColumns(page, ['backlog', 'epics']);
  await waitForBoard(page);

  // 6.1 A finished epic: the-button-quest, all stories accepted, 100%.
  await captureRegion(page.locator('.epic-card').filter({ hasText: 'the-button-quest' }), '01-finished-epic');

  // 6.2 An in-flight epic with a projected ETA: the-garden-of-patience.
  await captureRegion(page.locator('.epic-card').filter({ hasText: 'the-garden-of-patience' }), '02-eta-epic');

  // 6.3 An epic that can't be projected: the-moon-watch has one
  // unestimated story, so the card shows "ETA pending" instead of a date.
  await captureRegion(page.locator('.epic-card').filter({ hasText: 'the-moon-watch' }), '03-eta-pending');
});

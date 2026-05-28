// Chapter 2: first story. The scratch workspace has had the
// inception written. We create one story via the lab UI so the
// reader sees the actual New Story modal mid-creation: title typed,
// type chosen. Framed to the modal, not the whole window.

import { test } from '@playwright/test';
import { waitForBoard, captureRegion } from '../../playwright/harness';

test('Chapter 2 captures', async ({ page }) => {
  await waitForBoard(page);

  // 2.1 Create the story. Open the New Story modal, type the title,
  // and capture the modal itself with the form filled — the creation
  // moment, not the resulting board.
  await page.locator('button[aria-label="New story in Backlog"]').click();
  await page.waitForTimeout(200);
  await page.getByLabel(/Story title/i).fill('Write the README');
  await captureRegion(page.locator('.modal-panel'), 'create-modal');

  await page.getByRole('button', { name: 'Create' }).click();
  await page.waitForTimeout(500);

  const story = () => page.locator('.story').filter({ hasText: 'Write the README' });

  // 2.2 Estimate it: click the 3-point pip; frame the card so the
  // inline points picker (· 1 3 5 8, 3 active) is the focus.
  await story().getByRole('button', { name: 'Estimate 3' }).click();
  await page.waitForTimeout(300);
  await captureRegion(story(), 'estimate');

  // 2.4 Start it: click Start; frame the card now in the started state.
  await story().getByRole('button', { name: /Start/i }).click();
  await page.waitForTimeout(400);
  await captureRegion(story(), 'started');
});

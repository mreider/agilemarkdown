// Chapter 7: release markers. The meadow fixture has First-frost and
// The Long-Sleep markers in the backlog. We scroll the first marker
// into view and capture the board region (no bottom panel) so the
// amber release bar and the surrounding iteration band are visible.

import { test } from '@playwright/test';
import { waitForBoard, captureRegion } from '../../playwright/harness';

test('Chapter 7 captures', async ({ page }) => {
  await waitForBoard(page);

  // 7.1 The marker in the backlog: scroll a release marker into the
  // board viewport, then frame the board region as-is.
  await page.getByText('First-frost release', { exact: false }).first().scrollIntoViewIfNeeded();
  await page.waitForTimeout(300);
  await captureRegion(page.locator('.lab-top'), '02-markers-at-bottom');

  // 7.2 Creating a release marker: open the New Story modal, pick the
  // Release type to reveal the date picker, fill a date. Frame the
  // modal; cancel without creating so the board state is unchanged.
  await page.locator('button[aria-label="New story in Backlog"]').click();
  await page.waitForTimeout(200);
  await page.getByLabel(/Story title/i).fill('Spring planting');
  await page.locator('.modal-panel').getByRole('button', { name: /release/i }).click();
  await page.waitForTimeout(150);
  await page.getByLabel('Release target date').fill('2026-09-01');
  await captureRegion(page.locator('.modal-panel'), 'release-modal');
  await page.locator('.modal-panel').getByRole('button', { name: 'Cancel' }).click();

  // 7.3 Shipping a release: a marker whose target date has passed
  // shows a green Ship button on its bar. Create one dated in the past
  // to surface the button, frame the bar, then delete it so the
  // fixture is unchanged for later chapters.
  await page.locator('button[aria-label="New story in Backlog"]').click();
  await page.waitForTimeout(200);
  await page.getByLabel(/Story title/i).fill('Spring planting');
  await page.locator('.modal-panel').getByRole('button', { name: /release/i }).click();
  await page.waitForTimeout(150);
  await page.getByLabel('Release target date').fill('2026-05-01');
  await page.locator('.modal-panel').getByRole('button', { name: 'Create' }).click();
  await page.waitForTimeout(600);
  const bar = page.locator('.release-bar').filter({ hasText: 'Spring planting' });
  await bar.scrollIntoViewIfNeeded();
  await page.waitForTimeout(200);
  await captureRegion(bar, 'ship');

  // Clean up: delete the temporary marker.
  await bar.click();
  await page.waitForSelector('.detail-panel.open');
  await page.getByRole('button', { name: 'Delete story' }).click();
  await page.locator('.ceremony[aria-label="Confirm delete"]').getByRole('button', { name: 'Delete' }).click();
  await page.waitForTimeout(300);
});

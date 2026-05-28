// Chapter 4: the current sprint. The meadow fixture is fresh: Wait,
// Pace, Watch are started; Build/Set/Send are accepted-this-week;
// Sort is delivered. Each moment captures the board region only (no
// bottom panel), framed on the current-sprint band.

import { test } from '@playwright/test';
import { waitForBoard, captureBoard, captureRegion } from '../../playwright/harness';

test('Chapter 4 captures', async ({ page }) => {
  await waitForBoard(page);

  // Moment 1: the current sprint at rest (the board overview).
  await captureBoard(page, '01-current-sprint');

  // Moments 3-5 follow one story through the loop, so we frame just
  // that card: the status pill and the next-step button are what
  // change, and a single card reads large.
  const wait = () => page.locator('.story').filter({ hasText: 'Wait by the mailbox' });

  await wait().getByRole('button', { name: /Finish/i }).click();
  await page.waitForTimeout(400);
  await captureRegion(wait(), '03-finish-after');

  await wait().getByRole('button', { name: /Deliver/i }).click();
  await page.waitForTimeout(400);
  await captureRegion(wait(), '04-deliver-after');

  await wait().getByRole('button', { name: /Accept/i }).click();
  await page.waitForTimeout(400);
  await captureRegion(wait(), '05-accept-after');
});

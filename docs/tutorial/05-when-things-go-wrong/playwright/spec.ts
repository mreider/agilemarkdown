// Chapter 5: when things go wrong. Each image shows the moment it
// teaches: the reject form (reject WITH A REASON), a card reset to
// unstarted, and a card carrying the Blocked chip.

import { test, Page } from '@playwright/test';
import { waitForBoard, captureRegion } from '../../playwright/harness';

const card = (page: Page, title: string) => page.locator('.story').filter({ hasText: title });
const closePanel = async (page: Page) => {
  await page.locator('.detail-overlay').click({ position: { x: 8, y: 8 } });
  await page.waitForTimeout(300);
};

test('Chapter 5 captures', async ({ page }) => {
  await waitForBoard(page);

  // 5.1 Reject (with a reason): open the delivered story, open the
  // reject ceremony, type the reason. Frame the form, that's the point.
  await card(page, 'Sort the unread letters').click();
  await page.waitForSelector('.detail-panel.open');
  await page.locator('.detail-panel').getByRole('button', { name: 'rejected', exact: true }).click();
  await page.waitForSelector('textarea.reject-reason');
  await page.locator('textarea.reject-reason').fill('PM noticed the third pile is mislabeled.');
  await captureRegion(page.locator('.ceremony[aria-label="Reject with reason"]'), '01-reject');
  await page.locator('.ceremony[aria-label="Reject with reason"]').getByRole('button', { name: 'Cancel' }).click();
  await closePanel(page);

  // 5.2 Unstart: open a started story, click the unstarted state pill
  // (the full reset), close, and frame the card now back to unstarted.
  await card(page, 'Pace exactly seven times').click();
  await page.waitForSelector('.detail-panel.open');
  await page.locator('.detail-panel').getByRole('button', { name: 'unstarted', exact: true }).click();
  await page.waitForTimeout(300);
  await closePanel(page);
  await captureRegion(card(page, 'Pace exactly seven times'), '02-unstart');

  // 5.3 Block: open a started story, block it with a reason, close,
  // and frame the card with its red Blocked chip.
  await card(page, 'Watch the seedling').click();
  await page.waitForSelector('.detail-panel.open');
  await page.locator('.detail-panel').getByRole('button', { name: 'Block', exact: true }).click();
  await page.waitForSelector('.ceremony[aria-label="Block with reason"] textarea');
  await page.locator('.ceremony[aria-label="Block with reason"] textarea')
    .fill('Waiting on the gardener to clear the bed.');
  await page.locator('.ceremony[aria-label="Block with reason"]').getByRole('button', { name: 'Block', exact: true }).click();
  await page.waitForTimeout(300);
  await closePanel(page);
  await captureRegion(card(page, 'Watch the seedling'), '03-block');
});

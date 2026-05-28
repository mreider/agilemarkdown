import { test } from '@playwright/test';
import { waitForBoard, captureRegion } from '../../playwright/harness';

test('Chapter 10 captures', async ({ page }) => {
  await waitForBoard(page);

  // Open detail on the moon's-birthday story (seeded with comments
  // and tasks) and frame the detail panel itself.
  await page.locator('.story').filter({ hasText: "Plan the moons very first birthday party" }).click();
  await page.waitForSelector('.detail-panel.open', { timeout: 5000 });
  await page.waitForTimeout(400);
  await captureRegion(page.locator('.detail-panel'), '01-detail-with-comments-tasks');

  // 10.2 Add a comment: frame a comment (dated, attributed) in the panel.
  await captureRegion(page.locator('.detail-panel .comment').first(), '02-comment');

  // 10.3 Add a task and tick it: frame the task checklist (a ticked
  // task, the rest open, and the New task input).
  await captureRegion(page.locator('.detail-panel .tasks-list'), '03-task');
});

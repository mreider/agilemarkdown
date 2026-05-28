import { test } from '@playwright/test';
import { waitForBoard, captureRegion, setVisibleColumns } from '../../playwright/harness';

test('Chapter 9 captures', async ({ page }) => {
  await setVisibleColumns(page, ['search', 'backlog']);
  await waitForBoard(page);

  // Text search: matches by title. Frame just the search column so
  // the input + filtered results read in focus, matching the size
  // and focus of card-level captures elsewhere.
  const text = page.getByPlaceholder(/title or path/i);
  await text.fill('seedling');
  await page.waitForTimeout(200);
  await captureRegion(page.locator('.col-search'), '02-search-text');

  // Filter by owner on top of the text query.
  await page.locator('.search-select').first().selectOption({ label: 'toad' });
  await page.waitForTimeout(200);
  await captureRegion(page.locator('.col-search'), '03-search-owner');
});

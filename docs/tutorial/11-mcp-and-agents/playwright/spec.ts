import { test } from '@playwright/test';
import { waitForBoard, captureRegion } from '../../playwright/harness';

test('Chapter 11 captures', async ({ page }) => {
  await waitForBoard(page);

  // The MCP Console lives in the bottom panel — here it IS the
  // subject, so we frame the panel (tab strip + console) rather than
  // the board.
  await page.getByRole('tab', { name: /MCP Console/i }).click();
  await page.waitForTimeout(300);
  await captureRegion(page.locator('.lab-bottom'), '01-mcp-console');

  // Pick a tool and fill its arguments.
  await page.locator('select.lab-mcp-select').first().selectOption('set_status');
  await page.waitForTimeout(200);
  await page.locator('.lab-mcp-args').fill(JSON.stringify({
    path: 'stories/plan-the-moons-very-first-birthday-party.md',
    status: 'started',
  }, null, 2));

  // Send, then frame just the wire view (request + response). Let the
  // bottom panel grow to the JSON's full height so nothing is clipped
  // by its fixed 320px.
  await page.getByRole('button', { name: /Send/i }).click();
  await page.waitForTimeout(500);
  await page.addStyleTag({ content: `
    .lab-root { grid-template-rows: 1fr auto !important; }
    .lab-bottom, .lab-bottom-body, .lab-mcp, .lab-mcp-wire,
    .lab-mcp-block, .lab-mcp-block-body {
      height: auto !important; max-height: none !important; overflow: visible !important;
    }
  ` });
  await page.waitForTimeout(200);
  await captureRegion(page.locator('.lab-mcp-wire'), '04-wire-view');
});

// Shared helpers for chapter specs.
//
// Every spec navigates to the lab URL, waits for the board to render,
// and shoots one PNG per moment. The harness wraps two patterns:
//
//   - waitForBoard(page): waits until the Board has at least one
//     visible column. The lab snapshots over WebSocket as soon as
//     the static bundle mounts, so the wait is short.
//   - capture(page, name): screenshots the whole viewport (board +
//     bottom panel) and writes to <chapter>/images/<name>.png.
//
// chapter() returns the directory the orchestrator passed in via
// AM_CHAPTER_DIR so specs don't hardcode paths.

import { Page, Locator } from '@playwright/test';
import * as path from 'path';

export function chapterDir(): string {
  const d = process.env.AM_CHAPTER_DIR;
  if (!d) throw new Error('AM_CHAPTER_DIR not set; run via scripts/tutorial.sh');
  return d;
}

export async function waitForBoard(page: Page): Promise<void> {
  await page.goto('/');
  // Wait for the *loaded* state: `.board` once the snapshot has
  // arrived, or `.empty-state` on a fresh repo with no backlog yet.
  // We must NOT match `.empty` here: React paints `<div class="empty">
  // Loading…</div>` before __amWebBoot connects the snapshot WS, so
  // matching `.empty` resolves instantly and screenshots the loading
  // state instead of the board.
  await page.waitForSelector('.board, .empty-state', { timeout: 15_000 });
}

// setVisibleColumns seeds localStorage before the page navigates so
// the chapter spec can toggle on the Done/Search/Epics columns the
// extension defaults to off. Use this instead of a mid-page
// location.reload(), the explicit init-script avoids the
// "navigation interrupted by another navigation" race.
export async function setVisibleColumns(page: Page, cols: string[]): Promise<void> {
  await page.context().addInitScript((columns: string[]) => {
    try {
      localStorage.setItem('am-visible-columns', JSON.stringify(columns));
    } catch {
      // Private mode / no localStorage. Tutorial captures don't hit
      // this path; silently skip so a non-Chrome target doesn't fail.
    }
  }, cols);
}

export async function capture(page: Page, name: string): Promise<void> {
  // Let any in-flight write settle so the transient "Saving…" pill
  // isn't frozen into the frame. The pill unmounts once the busy
  // counter drains (SETTLE_MS after the last write); if it never
  // clears we shoot anyway rather than fail the capture.
  await page.locator('.busy-pill').waitFor({ state: 'detached', timeout: 5_000 }).catch(() => {});
  const file = path.join(chapterDir(), 'images', `${name}.png`);
  await page.screenshot({ path: file, fullPage: false });
  // eslint-disable-next-line no-console
  console.log(`  wrote ${file}`);
}

// captureBoard shoots the board region only (`.lab-top`), leaving the
// bottom CommandLog/MCP/Terminal panel out of frame entirely. This is
// the default for board-state moments: the reader sees the cards, not
// the lab plumbing. Scrolls to the top first so the current-sprint
// band leads.
export async function captureBoard(page: Page, name: string): Promise<void> {
  await page.locator('.busy-pill').waitFor({ state: 'detached', timeout: 5_000 }).catch(() => {});
  await page.evaluate(() => {
    const el = document.querySelector('.board');
    if (el) (el as HTMLElement).scrollTop = 0;
  });
  await captureRegion(page.locator('.lab-top'), name);
}

// captureRegion shoots a specific locator instead of the full viewport
// (a modal, the detail panel, the MCP console, a single column). Waits
// for any in-flight write to settle so the "Saving…" pill isn't frozen
// into the frame.
export async function captureRegion(loc: Locator, name: string): Promise<void> {
  await loc.page().locator('.busy-pill').waitFor({ state: 'detached', timeout: 5_000 }).catch(() => {});
  const file = path.join(chapterDir(), 'images', `${name}.png`);
  await loc.screenshot({ path: file });
  // eslint-disable-next-line no-console
  console.log(`  wrote ${file}`);
}

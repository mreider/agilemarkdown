// Playwright config for the tutorial capture pipeline.
//
// Each chapter has its own spec under docs/tutorial/<chapter>/playwright/spec.ts.
// scripts/tutorial.sh runs them one chapter at a time so the lab state stays
// coherent across moments within a chapter.
//
// Frames land in docs/tutorial/<chapter>/images/. Filenames are
// `NN-ui.png` where NN matches the moment number.

import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  // Source files live alongside the chapter prose, not under tests/.
  // The orchestrator passes the spec path explicitly to `playwright test`.
  testDir: '../../tutorial',
  testMatch: '**/playwright/spec.ts',
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: false,        // chapter state evolves sequentially
  workers: 1,
  retries: 0,
  reporter: 'line',
  use: {
    // The lab URL is injected by the orchestrator per chapter.
    baseURL: process.env.AM_LAB_URL ?? 'http://localhost:8091',
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,      // crisper screenshots for the static site
    screenshot: 'off',         // we shoot explicitly from inside each test
    trace: 'off',
  },
  projects: [
    // Clean desktop Chromium at 2x so element screenshots (board
    // region, modal, detail panel, MCP console) are crisp on retina.
    // A tall viewport lets the board area render its current-sprint
    // band plus several cards without the bottom panel cropping in.
    {
      name: 'desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1280, height: 1180 },
        deviceScaleFactor: 2,
      },
    },
  ],
});

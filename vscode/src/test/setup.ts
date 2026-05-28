// Global test setup. Loaded for every test file via vitest.config.ts.
//
// jest-dom matchers attach lazily: importing here means component test
// files don't have to re-import per file. Node-environment files
// (extension-bus.test.ts, am-client.test.ts) tolerate the import; the
// matchers just sit unused.
import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';

// React Testing Library does not auto-cleanup the DOM in vitest the way
// it does under jest's globals. Without this, every test in a file
// accumulates renders and ambiguous role/text queries fail with
// "found multiple elements". Cleanup is a no-op in node-environment
// suites because the import is dynamic and there's no DOM to scrub.
afterEach(async () => {
  try {
    const rtl = await import('@testing-library/react');
    rtl.cleanup();
    // Reset the captured postMessage log if the harness is loaded.
    const h = (globalThis as any).__amHarness;
    if (h) h.messages.length = 0;
  } catch {
    // No DOM environment in this file. Skip.
  }
});

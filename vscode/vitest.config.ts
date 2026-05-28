import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
    // Default environment is node (fast). React component tests opt into
    // happy-dom via the `@vitest-environment happy-dom` docblock at the
    // top of the file, keeping the CLI-driven extension-bus suite from
    // paying the DOM cost.
    environment: 'node',
    setupFiles: ['./src/test/setup.ts'],
  },
});

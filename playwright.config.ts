import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  retries: 0,
  use: {
    baseURL: 'http://localhost:9234',
    headless: true,
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
  webServer: {
    command: './kai dashboard --port 9234 --no-open',
    url: 'http://localhost:9234',
    reuseExistingServer: !process.env.CI,
    timeout: 10_000,
  },
});

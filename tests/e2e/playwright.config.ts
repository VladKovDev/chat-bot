import { defineConfig, devices } from '@playwright/test';

const websiteBaseURL = process.env.E2E_WEBSITE_URL ?? 'http://localhost:18081';

export default defineConfig({
  testDir: '.',
  testMatch: /.*\.spec\.ts/,
  globalSetup: './global-setup.ts',
  globalTeardown: './global-teardown.ts',
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  expect: {
    timeout: 10_000,
  },
  reporter: [
    ['list'],
    ['html', { outputFolder: 'test-results/e2e-html', open: 'never' }],
    ['json', { outputFile: 'test-results/e2e-results.json' }],
  ],
  outputDir: 'test-results/e2e-artifacts',
  use: {
    baseURL: websiteBaseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});

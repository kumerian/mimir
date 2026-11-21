import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  fullyParallel: false,
  outputDir: './.angular/playwright-results',
  use: {
    baseURL: 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
    launchOptions: {
      executablePath: process.env['PLAYWRIGHT_EXECUTABLE_PATH'] || '/usr/bin/google-chrome',
    },
  },
  webServer: {
    command: 'python3 -m http.server 4173 --directory dist/mimir/browser',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: false,
  },
});

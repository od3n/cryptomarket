import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: "html",
  use: {
    baseURL: process.env.BASE_URL || "http://localhost:3000",
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  // Run docker-compose before tests in CI
  webServer: process.env.CI
    ? undefined
    : {
        command: "cd .. && docker compose up -d frontend realtime api",
        url: "http://localhost:3000",
        reuseExistingServer: true,
        timeout: 120000,
      },
});

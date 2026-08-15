import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./capture",
  timeout: 120_000,
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["line"]],
  use: {
    browserName: "chromium",
    headless: true,
    colorScheme: "dark",
    reducedMotion: "reduce",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  outputDir: "test-results/p09-capture",
});

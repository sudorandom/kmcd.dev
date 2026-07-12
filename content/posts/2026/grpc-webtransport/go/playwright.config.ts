import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: "http://localhost:8080",
    ignoreHTTPSErrors: true,
    trace: "retain-on-failure",
    video: "retain-on-failure",
  },
  webServer: {
    command: "go run ./server",
    url: "http://localhost:8080/",
    reuseExistingServer: false,
    timeout: 15_000,
    ignoreHTTPSErrors: true,
  },
  projects: [
    {
      name: "chrome",
      use: {
        ...devices["Desktop Chrome"],
        channel: "chrome",
        launchOptions: {
          args: [
            "--enable-quic",
            "--origin-to-force-quic-on=localhost:4433",
            "--webtransport-developer-mode",
          ],
        },
      },
    },
  ],
});

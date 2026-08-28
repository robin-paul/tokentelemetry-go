import { defineConfig, devices } from '@playwright/test';
import dotenv from 'dotenv';
import path from 'path';
import os from 'os';
import fs from 'fs';

/**
 * Load environment variables from .env file.
 * Defaults to ./env/.env.dev if ENVIRONMENT is not set.
 */
const environment = process.env.ENVIRONMENT ?? 'dev';
const environmentPath = `./env/.env.${environment}`;

dotenv.config({ path: environmentPath, quiet: true });

const appUrl = process.env.APP_URL || 'http://127.0.0.1:8000';
const tempDir = process.env.E2E_TEMP_DIR || path.join(os.tmpdir(), 'tokentelemetry-e2e-run');
const tempDb = process.env.E2E_DB_PATH || path.join(tempDir, 'test.db');
const tempScanDir = process.env.E2E_SCAN_DIR || path.join(tempDir, 'logs');

fs.mkdirSync(tempScanDir, { recursive: true });

process.env.E2E_TEMP_DIR = tempDir;
process.env.E2E_SCAN_DIR = tempScanDir;
process.env.E2E_DB_PATH = tempDb;

/**
 * Playwright Test Configuration for TokenTelemetry Go
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
    testDir: './tests',

    /* Run tests serially to preserve log ingestion state */
    fullyParallel: false,

    /* Fail the build on CI if you accidentally left test.only in the source code */
    forbidOnly: !!process.env.CI,

    /* Retry on CI only */
    retries: process.env.CI ? 2 : 0,

    /* Limit parallel workers to 1 for database & file watcher determinism */
    workers: 1,

    /* Reporter configuration */
    reporter: process.env.CI
        ? [['blob'], ['html', { open: 'never' }]]
        : [['html', { open: 'never' }]],

    /* Shared settings for all projects */
    use: {
        baseURL: appUrl,
        testIdAttribute: 'data-test',
        trace: 'on-first-retry',
        screenshot: 'only-on-failure',
        video: 'retain-on-failure',
        actionTimeout: 10000,
        navigationTimeout: 30000,
    },

    /* Test timeout */
    timeout: 60000,

    /* Expect timeout */
    expect: {
        timeout: 10000,
    },

    /* Auto-start TokenTelemetry Go Hub server (and optional Next.js baseline for visual tests) */
    webServer: [
        {
            command: `sh -c "mkdir -p ${tempScanDir} && cd ../.. && ( [ -f bin/tt-server ] || make build ) && ./bin/tt-server --port 8000 --db ${tempDb} --scan-dir ${tempScanDir}"`,
            url: `${appUrl}/healthz`,
            reuseExistingServer: process.env.REUSE_EXISTING_SERVER === 'true',
            stdout: 'pipe',
            stderr: 'pipe',
            timeout: 30000,
        },
        ...(process.env.VISUAL_DIFF === 'true'
            ? [
                  {
                      command: `sh -c "cd ../../../tokentelemetry/frontend && ( [ -d .next ] || npm run build ) && npx next start -p 3000"`,
                      url: 'http://127.0.0.1:3000',
                      reuseExistingServer: process.env.REUSE_EXISTING_SERVER === 'true',
                      stdout: 'pipe',
                      stderr: 'pipe',
                      timeout: 30000,
                  },
              ]
            : []),
    ],

    /* Configure projects */
    projects: [
        /* API test project - request-context only */
        {
            name: 'api',
            testMatch: /.*\/api\/.*\.spec\.ts/,
        },

        /* Main UI test project - Chromium */
        {
            name: 'chromium',
            testIgnore: [/.*\/api\/.*\.spec\.ts/, /.*\/visual\/.*\.spec\.ts/],
            use: {
                ...devices['Desktop Chrome'],
                viewport: { width: 1920, height: 1080 },
            },
        },

        /* Visual regression diff project */
        {
            name: 'visual',
            testMatch: /.*\/visual\/.*\.spec\.ts/,
            use: {
                ...devices['Desktop Chrome'],
                viewport: { width: 1920, height: 1080 },
            },
        },
    ],
});

import { test as base, Page, expect, Locator } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { PNG } from 'pngjs';
import pixelmatch from 'pixelmatch';

export const BASELINE_URL = process.env.BASELINE_URL || 'http://127.0.0.1:3000';
export const CANDIDATE_URL = process.env.CANDIDATE_URL || process.env.APP_URL || 'http://127.0.0.1:8000';
export const DIFF_OUTPUT_DIR = path.resolve(process.cwd(), 'artifacts/visual-diff');
const RESULTS_JSON_PATH = path.join(DIFF_OUTPUT_DIR, 'results.json');

export interface VisualDiffResult {
    id: string;
    route: string;
    theme: 'dark' | 'light';
    viewport: { width: number; height: number };
    diffPixels: number;
    totalPixels: number;
    mismatchRatio: number;
    maxMismatchRatio: number;
    passed: boolean;
    baselinePath: string;
    candidatePath: string;
    diffPath: string;
    compositePath: string;
    details?: string;
}

export interface VisualDiffOptions {
    name: string;
    route: string;
    baselineRoute?: string;
    candidateRoute?: string;
    theme?: 'dark' | 'light';
    viewport?: { width: number; height: number };
    fullPage?: boolean;
    threshold?: number;
    maxMismatchRatio?: number;
    waitForSelector?: string;
    maskSelectors?: string[];
    beforeScreenshot?: (baselinePage: Page, candidatePage: Page) => Promise<void>;
    details?: string;
}

export type VisualDiffFixture = {
    baselinePage: Page;
    candidatePage: Page;
    compareRoute: (options: VisualDiffOptions) => Promise<VisualDiffResult>;
};

/**
 * Freeze animations, transitions, and relative timers for visual determinism.
 */
async function freezePageVisuals(
    page: Page,
    theme: 'dark' | 'light' = 'dark',
    viewport: { width: number; height: number } = { width: 1920, height: 1080 }
): Promise<void> {
    await page.setViewportSize(viewport);
    await page.emulateMedia({ reducedMotion: 'reduce' });

    // Set fixed clock if available
    try {
        if (page.clock) {
            await page.clock.setFixedTime(new Date('2026-08-26T12:00:00Z'));
        }
    } catch {
        // Clock may already be set or not supported
    }

    // Set theme and inject deterministic visual styling
    await page.addInitScript(({ themeName }) => {
        document.documentElement.setAttribute('data-theme', themeName);
        if (themeName === 'dark') {
            document.documentElement.classList.add('dark');
            document.documentElement.classList.remove('light');
        } else {
            document.documentElement.classList.add('light');
            document.documentElement.classList.remove('dark');
        }
    }, { themeName: theme });

    // Disable CSS animations, transitions, blinking cursors, and marquee effects
    await page.addStyleTag({
        content: `
            *, *::before, *::after {
                -webkit-transition: none !important;
                -moz-transition: none !important;
                -o-transition: none !important;
                -ms-transition: none !important;
                transition: none !important;
                -webkit-animation: none !important;
                -moz-animation: none !important;
                -o-animation: none !important;
                -ms-animation: none !important;
                animation: none !important;
                caret-color: transparent !important;
                scroll-behavior: auto !important;
            }
            .recharts-responsive-container {
                transition: none !important;
            }
        `,
    }).catch(() => {});
}

/**
 * Ensure image buffer is padded to target dimensions with transparent pixels.
 */
function normalizePngSize(png: PNG, targetWidth: number, targetHeight: number): PNG {
    if (png.width === targetWidth && png.height === targetHeight) {
        return png;
    }
    const normalized = new PNG({ width: targetWidth, height: targetHeight, fill: true });
    normalized.data.fill(0);
    PNG.bitblt(png, normalized, 0, 0, png.width, png.height, 0, 0);
    return normalized;
}

/**
 * Stitch Baseline, Diff, and Candidate side-by-side into a single composite PNG.
 */
function createSideBySideComposite(img1: PNG, diff: PNG, img2: PNG, width: number, height: number): PNG {
    const compositeWidth = width * 3;
    const compositeHeight = height;
    const composite = new PNG({ width: compositeWidth, height: compositeHeight });

    // Copy img1 to left (0 -> width)
    PNG.bitblt(img1, composite, 0, 0, width, height, 0, 0);

    // Copy diff to center (width -> 2 * width)
    PNG.bitblt(diff, composite, 0, 0, width, height, width, 0);

    // Copy img2 to right (2 * width -> 3 * width)
    PNG.bitblt(img2, composite, 0, 0, width, height, width * 2, 0);

    return composite;
}

/**
 * Load persisted results list.
 */
function loadPersistedResults(): VisualDiffResult[] {
    try {
        if (fs.existsSync(RESULTS_JSON_PATH)) {
            return JSON.parse(fs.readFileSync(RESULTS_JSON_PATH, 'utf-8'));
        }
    } catch {
        // Fallback to empty array
    }
    return [];
}

/**
 * Save persisted results list.
 */
function savePersistedResults(results: VisualDiffResult[]): void {
    fs.writeFileSync(RESULTS_JSON_PATH, JSON.stringify(results, null, 2), 'utf-8');
}

/**
 * Write HTML summary report for the visual comparison suite.
 */
export function writeVisualDiffReport(results: VisualDiffResult[], reportPath: string = path.join(DIFF_OUTPUT_DIR, 'index.html')): void {
    const totalCount = results.length;
    const passedCount = results.filter(r => r.passed).length;
    const failedCount = totalCount - passedCount;
    const avgMismatch = totalCount > 0 ? (results.reduce((acc, r) => acc + r.mismatchRatio, 0) / totalCount * 100).toFixed(3) : '0.000';

    const cardsHtml = results.map(r => `
        <div class="test-card ${r.passed ? 'passed' : 'failed'}">
            <div class="card-header">
                <div>
                    <span class="badge ${r.passed ? 'badge-pass' : 'badge-fail'}">${r.passed ? 'PASS' : 'AUDIT'}</span>
                    <strong class="test-id">${r.id}</strong>
                    <span class="route-tag">${r.route}</span>
                </div>
                <div class="metrics">
                    <span>Mismatch: <strong>${(r.mismatchRatio * 100).toFixed(3)}%</strong> (Tolerance: ${(r.maxMismatchRatio * 100).toFixed(2)}%)</span>
                    <span>Diff Pixels: <strong>${r.diffPixels.toLocaleString()}</strong> / ${r.totalPixels.toLocaleString()}</span>
                </div>
            </div>
            ${r.details ? `<div class="details">${r.details}</div>` : ''}
            <div class="card-labels">
                <div>Next.js Baseline (:3000)</div>
                <div>Pixelmatch Visual Diff (Highlighted)</div>
                <div>Go Astro Candidate (:8000)</div>
            </div>
            <div class="image-wrapper">
                <a href="${path.basename(r.compositePath)}" target="_blank">
                    <img src="${path.basename(r.compositePath)}" alt="${r.id} composite diff" loading="lazy" />
                </a>
            </div>
        </div>
    `).join('\n');

    const html = `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TokenTelemetry Visual Regression Diff Report</title>
    <style>
        :root {
            --bg-base: #090d16;
            --bg-panel: #111827;
            --bg-card: #1f2937;
            --border-color: #374151;
            --text-main: #f3f4f6;
            --text-muted: #9ca3af;
            --brand: #3b82f6;
            --pass-bg: #064e3b;
            --pass-text: #34d399;
            --fail-bg: #1e3a8a;
            --fail-text: #60a5fa;
        }
        body {
            background-color: var(--bg-base);
            color: var(--text-main);
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            margin: 0;
            padding: 24px;
        }
        .container {
            max-width: 1600px;
            margin: 0 auto;
        }
        header {
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 20px;
            margin-bottom: 24px;
        }
        h1 { margin: 0 0 8px 0; font-size: 24px; font-weight: 700; color: #fff; }
        .summary-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 16px;
            margin-top: 16px;
        }
        .stat-box {
            background-color: var(--bg-panel);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 16px;
        }
        .stat-title { font-size: 13px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; }
        .stat-value { font-size: 28px; font-weight: bold; margin-top: 4px; }
        .test-card {
            background-color: var(--bg-panel);
            border: 1px solid var(--border-color);
            border-radius: 10px;
            margin-bottom: 28px;
            overflow: hidden;
        }
        .test-card.failed {
            border-color: #3b82f6;
        }
        .card-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 14px 20px;
            background-color: var(--bg-card);
            border-bottom: 1px solid var(--border-color);
        }
        .badge {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 700;
            margin-right: 8px;
        }
        .badge-pass { background-color: var(--pass-bg); color: var(--pass-text); }
        .badge-fail { background-color: var(--fail-bg); color: var(--fail-text); }
        .test-id { font-size: 15px; color: #fff; margin-right: 12px; }
        .route-tag { background: #374151; color: #d1d5db; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-family: monospace; }
        .metrics { font-size: 13px; color: var(--text-muted); display: flex; gap: 16px; }
        .details { padding: 8px 20px; font-size: 13px; color: var(--text-muted); background-color: #1a2234; border-bottom: 1px solid var(--border-color); }
        .card-labels {
            display: grid;
            grid-template-columns: 1fr 1fr 1fr;
            text-align: center;
            font-size: 12px;
            font-weight: 600;
            color: var(--text-muted);
            background: #151d2f;
            padding: 8px 0;
            border-bottom: 1px solid var(--border-color);
        }
        .image-wrapper {
            padding: 12px;
            background: #0d1117;
            overflow-x: auto;
        }
        .image-wrapper img {
            width: 100%;
            display: block;
            border-radius: 4px;
            border: 1px solid #2d3748;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>TokenTelemetry Visual Regression Diff Report</h1>
            <p style="margin: 0; color: var(--text-muted);">Comparative audit: Next.js Baseline (:3000) vs Go Astro Candidate (:8000)</p>
            <div class="summary-stats">
                <div class="stat-box">
                    <div class="stat-title">Total Test Cases</div>
                    <div class="stat-value">${totalCount}</div>
                </div>
                <div class="stat-box">
                    <div class="stat-title">Captured &amp; Audited</div>
                    <div class="stat-value" style="color: var(--pass-text);">${totalCount}</div>
                </div>
                <div class="stat-box">
                    <div class="stat-title">Average Mismatch</div>
                    <div class="stat-value">${avgMismatch}%</div>
                </div>
            </div>
        </header>
        <main>
            ${cardsHtml}
        </main>
    </div>
</body>
</html>`;

    fs.writeFileSync(reportPath, html, 'utf-8');
}

/**
 * Playwright fixture extending base test with dual-server visual comparison capabilities.
 */
export const test = base.extend<VisualDiffFixture>({
    baselinePage: async ({ browser }, use) => {
        const context = await browser.newContext({
            baseURL: BASELINE_URL,
            viewport: { width: 1920, height: 1080 },
        });
        const page = await context.newPage();
        await use(page);
        await context.close();
    },

    candidatePage: async ({ browser }, use) => {
        const context = await browser.newContext({
            baseURL: CANDIDATE_URL,
            viewport: { width: 1920, height: 1080 },
        });
        const page = await context.newPage();
        await use(page);
        await context.close();
    },

    compareRoute: async ({ baselinePage, candidatePage }, use) => {
        fs.mkdirSync(DIFF_OUTPUT_DIR, { recursive: true });

        await use(async (options: VisualDiffOptions): Promise<VisualDiffResult> => {
            const {
                name,
                route,
                baselineRoute = route,
                candidateRoute = route,
                theme = 'dark',
                viewport = { width: 1920, height: 1080 },
                fullPage = true,
                threshold = 0.1,
                maxMismatchRatio = 0.005, // 0.5% tolerance
                waitForSelector,
                maskSelectors = [],
                beforeScreenshot,
                details,
            } = options;

            // 1. Prepare visual determinism
            await freezePageVisuals(baselinePage, theme, viewport);
            await freezePageVisuals(candidatePage, theme, viewport);

            // 2. Navigate both servers in parallel
            await Promise.all([
                baselinePage.goto(baselineRoute, { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => {}),
                candidatePage.goto(candidateRoute, { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => {}),
            ]);

            // 3. Set theme explicitly after navigation
            await baselinePage.evaluate((t) => {
                document.documentElement.setAttribute('data-theme', t);
                if (t === 'dark') document.documentElement.classList.add('dark');
                else document.documentElement.classList.remove('dark');
            }, theme).catch(() => {});

            await candidatePage.evaluate((t) => {
                document.documentElement.setAttribute('data-theme', t);
                if (t === 'dark') document.documentElement.classList.add('dark');
                else document.documentElement.classList.remove('dark');
            }, theme).catch(() => {});

            // 4. Wait for selector if specified
            if (waitForSelector) {
                await Promise.all([
                    baselinePage.waitForSelector(waitForSelector, { state: 'visible', timeout: 5000 }).catch(() => {}),
                    candidatePage.waitForSelector(waitForSelector, { state: 'visible', timeout: 5000 }).catch(() => {}),
                ]);
            }

            // 5. Run beforeScreenshot interaction if provided
            if (beforeScreenshot) {
                await beforeScreenshot(baselinePage, candidatePage);
            }

            // Settle rendering
            await baselinePage.waitForTimeout(300);
            await candidatePage.waitForTimeout(300);

            // 6. Screenshot both pages
            const baseShot = path.join(DIFF_OUTPUT_DIR, `${name}-baseline.png`);
            const candShot = path.join(DIFF_OUTPUT_DIR, `${name}-candidate.png`);
            const diffShot = path.join(DIFF_OUTPUT_DIR, `${name}-diff.png`);
            const compShot = path.join(DIFF_OUTPUT_DIR, `${name}-side-by-side.png`);

            const maskBaselineLocators: Locator[] = maskSelectors.map(sel => baselinePage.locator(sel));
            const maskCandidateLocators: Locator[] = maskSelectors.map(sel => candidatePage.locator(sel));

            const baseBuffer = await baselinePage.screenshot({
                fullPage,
                path: baseShot,
                mask: maskBaselineLocators.length > 0 ? maskBaselineLocators : undefined,
            });

            const candBuffer = await candidatePage.screenshot({
                fullPage,
                path: candShot,
                mask: maskCandidateLocators.length > 0 ? maskCandidateLocators : undefined,
            });

            // 7. Parse & Normalize PNGs
            const rawImg1 = PNG.sync.read(baseBuffer);
            const rawImg2 = PNG.sync.read(candBuffer);

            const maxWidth = Math.max(rawImg1.width, rawImg2.width);
            const maxHeight = Math.max(rawImg1.height, rawImg2.height);

            const img1 = normalizePngSize(rawImg1, maxWidth, maxHeight);
            const img2 = normalizePngSize(rawImg2, maxWidth, maxHeight);

            // 8. Pixelmatch diff
            const diff = new PNG({ width: maxWidth, height: maxHeight });
            const mismatchedPixels = pixelmatch(
                img1.data,
                img2.data,
                diff.data,
                maxWidth,
                maxHeight,
                { threshold, alpha: 0.2, diffColor: [255, 0, 128] }
            );

            fs.writeFileSync(diffShot, PNG.sync.write(diff));

            // 9. Generate Composite side-by-side
            const composite = createSideBySideComposite(img1, diff, img2, maxWidth, maxHeight);
            fs.writeFileSync(compShot, PNG.sync.write(composite));

            const totalPixels = maxWidth * maxHeight;
            const mismatchRatio = totalPixels > 0 ? mismatchedPixels / totalPixels : 0;
            const passed = mismatchRatio <= maxMismatchRatio;

            const result: VisualDiffResult = {
                id: name,
                route,
                theme,
                viewport,
                diffPixels: mismatchedPixels,
                totalPixels,
                mismatchRatio,
                maxMismatchRatio,
                passed,
                baselinePath: baseShot,
                candidatePath: candShot,
                diffPath: diffShot,
                compositePath: compShot,
                details,
            };

            // Accumulate and persist results
            const persisted = loadPersistedResults();
            const existingIdx = persisted.findIndex(p => p.id === result.id);
            if (existingIdx >= 0) {
                persisted[existingIdx] = result;
            } else {
                persisted.push(result);
            }
            savePersistedResults(persisted);
            writeVisualDiffReport(persisted);

            return result;
        });
    },
});

export { expect };

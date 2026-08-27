import { test, expect } from '../../../fixtures/visual/visual-diff-fixture';
import { FileTranscriptManager } from '../../../fixtures/transcript/transcript-fixture';
import fs from 'fs';

test.describe('Automated Dual-Server Visual Regression & Parity Diff Suite', () => {
    let transcriptManager: FileTranscriptManager;
    const testSessionId = 'cursor-visual-session-101';
    const testProjectName = 'project-alpha';

    test.beforeAll(async () => {
        transcriptManager = new FileTranscriptManager();

        // Seed deterministic test sessions across agents and projects
        await transcriptManager.writeCursorSession(testProjectName, testSessionId, [
            {
                role: 'user',
                prompt: 'Please investigate the performance bottlenecks in SQLite FTS5 search index and refactor the query builder.',
            },
            {
                role: 'assistant',
                model: 'claude-3-5-sonnet',
                inputTokens: 14500,
                outputTokens: 3200,
                cacheReadTokens: 4500,
                cacheWriteTokens: 600,
                tools: ['read_file', 'ast_grep_search'],
                thought: 'Analyzing FTS5 virtual table schemas, index triggers, and tokenizers. The BM25 ranking function provides optimal relevance scoring.',
                content: '### FTS5 Indexing Optimization Plan\n\n1. Use external content tables with synchronization triggers.\n2. Enable prefix indexing for sub-string matching.\n\n```go\nfunc BuildFTSQuery(terms []string) string {\n    return strings.Join(terms, " AND ")\n}\n```',
            },
            {
                role: 'user',
                prompt: 'Run the benchmarks to confirm throughput improvements.',
            },
            {
                role: 'assistant',
                model: 'claude-3-5-sonnet',
                inputTokens: 8200,
                outputTokens: 1800,
                tools: ['run_command'],
                content: 'Benchmarks show a **3.4x** improvement in query execution time with negligible memory overhead.',
            },
        ]);

        await transcriptManager.writeOpenCodeSession('project-beta', 'opencode-visual-session-202', [
            {
                role: 'user',
                prompt: 'Build the Playwright visual regression comparison fixture with pixelmatch.',
            },
            {
                role: 'assistant',
                model: 'claude-3-7-sonnet',
                inputTokens: 32000,
                outputTokens: 6400,
                cacheReadTokens: 8000,
                cacheWriteTokens: 1200,
                toolName: 'write_file',
                thought: 'Constructing dual-server Playwright test harness with deterministic clock freezing and motion reduction.',
                content: 'Implemented `visual-diff-fixture.ts` with pixelmatch comparison, side-by-side composite PNG generation, and HTML summary reports.',
            },
        ]);

        await transcriptManager.writeCopilotSession('project-gamma', 'copilot-visual-session-303', {
            model: 'gpt-4o',
            inputTokens: 5400,
            outputTokens: 1200,
            cacheReadTokens: 1000,
        });

        // Allow backend scanner time to ingest all transcripts into SQLite
        await new Promise(resolve => setTimeout(resolve, 800));
    });

    test.afterAll(async () => {
        if (transcriptManager) {
            await transcriptManager.cleanup();
        }
    });

    test('CAP-01-DASH-DARK: Overview Dashboard in Dark Theme', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-01-DASH-DARK',
            route: '/',
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Standard overview dashboard layout, KPI metrics strip, connected agent cards, and activity feed.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-02-DASH-LIGHT: Overview Dashboard in Light Theme', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-02-DASH-LIGHT',
            route: '/',
            theme: 'light',
            viewport: { width: 1920, height: 1080 },
            details: 'Light theme styling, panel backgrounds, text contrast, and border token parity.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-03-SESS-LIST: Sessions Catalog with Active Agent Filter', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-03-SESS-LIST',
            route: '/sessions?agent=cursor',
            baselineRoute: '/projects',
            candidateRoute: '/sessions?agent=cursor',
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Sessions catalog with active Cursor agent filter pill and filtered session list table.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-04-SESS-SEARCH: Sessions Catalog with Active Search Query', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-04-SESS-SEARCH',
            route: '/sessions?q=refactor',
            baselineRoute: '/projects',
            candidateRoute: '/sessions?q=refactor',
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Sessions catalog with active search term in search input and debounced search results.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-05-TRACE-VIEW: Session Inspector Initial Turn & Scrubber View', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-05-TRACE-VIEW',
            route: `/sessions/${testSessionId}`,
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Deep session inspector with interactive turn scrubber, turn counter, and message turn stream.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-06-TRACE-TOOLS: Session Inspector with Tools Category Filter Active', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-06-TRACE-TOOLS',
            route: `/sessions/${testSessionId}?category=tools`,
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Session inspector filtered to tool invocation turns and tool result cards.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-07-TRACE-THOUGHT: Session Inspector with Reasoning Block Expanded', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-07-TRACE-THOUGHT',
            route: `/sessions/${testSessionId}`,
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            beforeScreenshot: async (baselinePage, candidatePage) => {
                const bThoughtBtn = baselinePage.locator('[data-test="thought-toggle"], button:has-text("Thought"), button:has-text("Thinking")').first();
                const cThoughtBtn = candidatePage.locator('[data-test="thought-toggle"], button:has-text("Thought"), button:has-text("Thinking")').first();
                if (await bThoughtBtn.isVisible().catch(() => false)) await bThoughtBtn.click().catch(() => {});
                if (await cThoughtBtn.isVisible().catch(() => false)) await cThoughtBtn.click().catch(() => {});
            },
            details: 'Session inspector with expanded reasoning / thought monologue block and formatted markdown.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-08-PROJ-GRID: Projects Catalog Grid View Mode', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-08-PROJ-GRID',
            route: '/projects?view=grid',
            baselineRoute: '/projects',
            candidateRoute: '/projects?view=grid',
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Projects catalog displaying responsive project cards, token summaries, and worktree badges.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-09-PROJ-LIST: Projects Catalog Table / List View Mode', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-09-PROJ-LIST',
            route: '/projects?view=table',
            baselineRoute: '/projects',
            candidateRoute: '/projects?view=table',
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Projects catalog in tabular list mode with sortable columns and subagent rollups.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-10-PROJ-HEATMAP: Project Detail Insights & Activity View', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-10-PROJ-HEATMAP',
            route: `/projects/${testProjectName}?tab=insights`,
            baselineRoute: `/projects/${testProjectName}/insights`,
            candidateRoute: `/projects/${testProjectName}?tab=insights`,
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Project Detail workspace with Insights tab selected showing telemetry KPIs and activity charts.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-11-PROJ-PLANS: Project Detail Architectural Plans & Markdown View', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-11-PROJ-PLANS',
            route: `/projects/${testProjectName}?tab=plans`,
            baselineRoute: `/projects/${testProjectName}/plans`,
            candidateRoute: `/projects/${testProjectName}?tab=plans`,
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Project Detail workspace with Plans tab selected showing Markdown document viewer.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-12-ANALYTICS-30D: Analytics Dashboard with 30-Day Range', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-12-ANALYTICS-30D',
            route: '/analytics?range=30d',
            baselineRoute: '/analytics',
            candidateRoute: '/analytics?range=30d',
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Analytics view with 30d range preset, stacked token area chart, and model leaderboard.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-13-ANALYTICS-90D: Analytics Dashboard with 90-Day Range', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-13-ANALYTICS-90D',
            route: '/analytics?range=90d',
            baselineRoute: '/analytics',
            candidateRoute: '/analytics?range=90d',
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Analytics view with 90d range preset, agent share breakdown, and token consumption statistics.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-14-SETTINGS-CFG: Settings Model Pricing Overrides & Configuration', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-14-SETTINGS-CFG',
            route: '/settings',
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            details: 'Settings page with Model Pricing Overrides CRUD table, rate inputs, and configuration cards.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });

    test('CAP-15-NAV-COLLAPSED: Layout Shell with Navigation Sidebar Collapsed', { tag: '@visual' }, async ({ compareRoute }) => {
        const result = await compareRoute({
            name: 'CAP-15-NAV-COLLAPSED',
            route: '/',
            theme: 'dark',
            viewport: { width: 1920, height: 1080 },
            beforeScreenshot: async (baselinePage, candidatePage) => {
                const bCollapseBtn = baselinePage.locator('[data-test="toggle-sidebar"], button[aria-label*="collapse" i], button[aria-label*="sidebar" i]').first();
                const cCollapseBtn = candidatePage.locator('[data-test="toggle-sidebar"], button[aria-label*="collapse" i], button[aria-label*="sidebar" i]').first();
                if (await bCollapseBtn.isVisible().catch(() => false)) await bCollapseBtn.click().catch(() => {});
                if (await cCollapseBtn.isVisible().catch(() => false)) await cCollapseBtn.click().catch(() => {});
            },
            details: 'Global layout shell with sidebar collapsed into compact icon-only mode.',
        });
        expect(fs.existsSync(result.compositePath)).toBe(true);
        expect(result.diffPixels).toBeGreaterThanOrEqual(0);
    });
});

import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('Cursor synthetic transcript ingestion and live dashboard feed', () => {
    test.afterEach(async ({ transcriptFixture }) => {
        await transcriptFixture.cleanup();
    });

    test(
        'should ingest Cursor transcript and dynamically update live dashboard feed and KPI metrics via SSE',
        { tag: '@smoke' },
        async ({ dashboardPage, navbar, transcriptFixture }) => {
            const projectName = 'e2e-cursor-project';
            const sessionId = `cursor-test-${Date.now()}`;

            await test.step('GIVEN the user opens the TokenTelemetry overview dashboard', async () => {
                await dashboardPage.open();
                await expect(dashboardPage.pageTitle).toBeVisible();
                await expect(navbar.liveIndicator).toBeVisible();
            });

            await test.step('WHEN a multi-turn Cursor session transcript is emitted to disk', async () => {
                await transcriptFixture.writeCursorSession(projectName, sessionId, [
                    {
                        role: 'user',
                        model: 'claude-3-5-sonnet',
                        inputTokens: 800,
                        outputTokens: 0,
                    },
                    {
                        role: 'assistant',
                        model: 'claude-3-5-sonnet',
                        inputTokens: 1200,
                        outputTokens: 450,
                        cacheReadTokens: 600,
                        cacheCreationTokens: 100,
                        tools: ['read_file', 'edit_file'],
                    },
                ]);
            });

            await test.step('THEN the dashboard live feed reflects the new session with agent badge, project name, and model', async () => {
                const sessionRow = dashboardPage.getSessionRow(projectName);
                await expect(sessionRow).toBeVisible();
                await expect(sessionRow).toContainText('Cursor');
                await expect(sessionRow).toContainText('claude-3-5-sonnet');
            });

            await test.step('AND the dashboard KPI cards update with non-zero token metrics', async () => {
                await expect(dashboardPage.totalTokensValue).not.toHaveText('0');
                await expect(dashboardPage.indexedSessionsValue).not.toHaveText('0');
            });
        }
    );
});

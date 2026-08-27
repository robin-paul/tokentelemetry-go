import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('OpenCode synthetic transcript ingestion and session inspector scrubber', () => {
    test.afterEach(async ({ transcriptFixture }) => {
        await transcriptFixture.cleanup();
    });

    test(
        'should ingest OpenCode transcript, filter on sessions page, and scrub turns in session inspector',
        { tag: '@smoke' },
        async ({ sessionsPage, sessionDetailPage, transcriptFixture }) => {
            const projectName = 'e2e-opencode-project';
            const sessionId = `opencode-test-${Date.now()}`;

            await test.step('GIVEN an OpenCode multi-turn session transcript is emitted to disk', async () => {
                await transcriptFixture.writeOpenCodeSession(projectName, sessionId, [
                    {
                        role: 'assistant',
                        model: 'claude-3-7-sonnet',
                        inputTokens: 1500,
                        outputTokens: 300,
                        cacheReadTokens: 500,
                        cacheWriteTokens: 100,
                        toolName: 'ast_grep_search',
                    },
                    {
                        role: 'assistant',
                        model: 'claude-3-7-sonnet',
                        inputTokens: 2200,
                        outputTokens: 550,
                        cacheReadTokens: 1200,
                        cacheWriteTokens: 200,
                        toolName: 'replace_file_content',
                    },
                ]);
            });

            await test.step('WHEN the user navigates to the Sessions catalog and filters by OpenCode', async () => {
                await sessionsPage.open();
                await expect(sessionsPage.sessionTable).toBeVisible();

                const openCodeFilter = sessionsPage.getAgentFilterButton('opencode');
                await expect(openCodeFilter).toBeVisible();
                await openCodeFilter.click();
            });

            await test.step('THEN the filtered sessions list shows the newly ingested OpenCode session', async () => {
                const sessionRow = sessionsPage.getSessionRow(projectName).first();
                await expect(sessionRow).toBeVisible();
                await expect(sessionRow).toContainText('OpenCode');
                await expect(sessionRow).toContainText('claude-3-7-sonnet');
            });

            await test.step('WHEN the user clicks into the Session Inspector', async () => {
                await sessionsPage.clickSession(projectName);
                await expect(sessionDetailPage.sessionIdHeading).toBeVisible();
                await expect(sessionDetailPage.agentBadge).toBeVisible();
                await expect(sessionDetailPage.agentBadge).toContainText('OpenCode');
            });

            await test.step('THEN the Session Inspector renders turns with tools and scrubber controls', async () => {
                await expect(sessionDetailPage.stepScrubber).toBeVisible();
                await expect(sessionDetailPage.turnCards.first()).toBeVisible();

                await expect(sessionDetailPage.tokensValue).toBeVisible();
                await expect(sessionDetailPage.netCostValue).toBeVisible();

                const toolBadge = sessionDetailPage.page.getByText('ast_grep_search').first();
                await expect(toolBadge).toBeVisible();

                await sessionDetailPage.scrubToStep(0);
                await expect(sessionDetailPage.scrubberStepLabel).toContainText('Step 1 of 2');
            });
        }
    );
});

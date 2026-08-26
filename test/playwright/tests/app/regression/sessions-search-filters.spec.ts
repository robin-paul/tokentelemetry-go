import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('Session Catalog Multi-Criteria Search and Filtering', () => {
    test.afterEach(async ({ transcriptFixture }) => {
        await transcriptFixture.cleanup();
    });

    test(
        'should search by keyword, filter across agent facets, apply range bounds, and sync URL query state',
        { tag: '@regression' },
        async ({ page, sessionsPage, transcriptFixture }) => {
            await test.step('GIVEN multiple distinct agent sessions exist in telemetry', async () => {
                await transcriptFixture.writeCursorSession('project-alpha', 'cursor-search-session-101', [
                    {
                        role: 'assistant',
                        model: 'claude-3-5-sonnet',
                        inputTokens: 12000,
                        outputTokens: 3000,
                    },
                ]);

                await transcriptFixture.writeOpenCodeSession('project-beta', 'opencode-search-session-202', [
                    {
                        role: 'assistant',
                        model: 'claude-3-7-sonnet',
                        inputTokens: 45000,
                        outputTokens: 8000,
                        toolName: 'run_command',
                    },
                ]);

                await transcriptFixture.writeCopilotSession('project-gamma', 'copilot-search-session-303', [
                    {
                        role: 'assistant',
                        model: 'gpt-4o',
                        inputTokens: 4000,
                        outputTokens: 800,
                    },
                ]);
            });

            await test.step('WHEN the user navigates to the Sessions catalog', async () => {
                await sessionsPage.open();
                await expect(sessionsPage.sessionTable).toBeVisible();
                // All 3 sessions should be listed initially
                await expect(sessionsPage.getSessionRow('project-alpha')).toBeVisible();
                await expect(sessionsPage.getSessionRow('project-beta')).toBeVisible();
                await expect(sessionsPage.getSessionRow('project-gamma')).toBeVisible();
            });

            await test.step('THEN searching by keyword debounces and filters the table matching sessions', async () => {
                await sessionsPage.search('project-beta');
                // Allow 300ms debounce to settle
                await page.waitForTimeout(400);

                await expect(sessionsPage.getSessionRow('project-beta')).toBeVisible();
                await expect(sessionsPage.getSessionRow('project-alpha')).not.toBeVisible();
                await expect(sessionsPage.getSessionRow('project-gamma')).not.toBeVisible();

                // URL should have ?q=project-beta
                expect(page.url()).toContain('q=project-beta');
            });

            await test.step('AND clearing search restores full session catalog', async () => {
                await sessionsPage.searchInput.clear();
                await page.waitForTimeout(400);

                await expect(sessionsPage.getSessionRow('project-alpha')).toBeVisible();
                await expect(sessionsPage.getSessionRow('project-beta')).toBeVisible();
                await expect(sessionsPage.getSessionRow('project-gamma')).toBeVisible();
            });

            await test.step('WHEN filtering by Agent pill (e.g. Cursor)', async () => {
                await sessionsPage.filterByAgent('Cursor');
                await page.waitForTimeout(200);

                await expect(sessionsPage.getSessionRow('project-alpha')).toBeVisible();
                await expect(sessionsPage.getSessionRow('project-beta')).not.toBeVisible();
                await expect(sessionsPage.getSessionRow('project-gamma')).not.toBeVisible();

                expect(page.url()).toContain('agent=cursor');
            });

            await test.step('WHEN toggling range bounds panel and applying cost/token filters', async () => {
                // Reset agent filter
                await sessionsPage.allAgentsFilterButton.click();
                await page.waitForTimeout(200);

                // Open range filters panel
                await sessionsPage.rangeFiltersButton.click();
                await expect(sessionsPage.minTokensInput).toBeVisible();

                // Filter for sessions with at least 30,000 tokens
                await sessionsPage.minTokensInput.fill('30000');
                await page.waitForTimeout(200);

                // Only OpenCode session (45k + 8k tokens) should match
                await expect(sessionsPage.getSessionRow('project-beta')).toBeVisible();
                await expect(sessionsPage.getSessionRow('project-alpha')).not.toBeVisible();
                await expect(sessionsPage.getSessionRow('project-gamma')).not.toBeVisible();
            });

            await test.step('AND resetting all filters restores default view', async () => {
                await sessionsPage.resetFiltersButton.click();
                await page.waitForTimeout(200);

                await expect(sessionsPage.getSessionRow('project-alpha')).toBeVisible();
                await expect(sessionsPage.getSessionRow('project-beta')).toBeVisible();
                await expect(sessionsPage.getSessionRow('project-gamma')).toBeVisible();
            });
        }
    );
});

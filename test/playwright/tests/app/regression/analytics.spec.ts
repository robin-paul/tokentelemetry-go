import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('Analytics charts and leaderboards', () => {
    test.afterEach(async ({ transcriptFixture }) => {
        await transcriptFixture.cleanup();
    });

    test(
        'should render Analytics time-series charts, date range presets, and Model/Agent leaderboards',
        { tag: '@regression' },
        async ({ analyticsPage, transcriptFixture }) => {
            await test.step('GIVEN multi-agent transcripts exist on disk', async () => {
                await transcriptFixture.writeCursorSession('analytics-proj-a', 'cursor-sess-1', [
                    {
                        role: 'assistant',
                        model: 'claude-3-5-sonnet',
                        inputTokens: 5000,
                        outputTokens: 1200,
                    },
                ]);

                await transcriptFixture.writeOpenCodeSession('analytics-proj-b', 'opencode-sess-1', [
                    {
                        role: 'assistant',
                        model: 'claude-3-7-sonnet',
                        inputTokens: 8000,
                        outputTokens: 2400,
                        toolName: 'ast_grep_search',
                    },
                ]);
            });

            await test.step('WHEN the user navigates to the Analytics view', async () => {
                await analyticsPage.open();
                await expect(analyticsPage.pageTitle).toBeVisible();
            });

            await test.step('THEN the stacked token volume area chart and agent share pie chart are rendered', async () => {
                await expect(analyticsPage.tokenConsumptionChart).toBeVisible();
                await expect(analyticsPage.tokenConsumptionChartSvg).toBeVisible();

                await expect(analyticsPage.agentTokenShareChart).toBeVisible();
                await expect(analyticsPage.agentTokenShareChartSvg).toBeVisible();
            });

            await test.step('AND the user can toggle date range presets (7d, 30d, 90d, All)', async () => {
                await expect(analyticsPage.getPresetButton('7d')).toBeVisible();
                await expect(analyticsPage.getPresetButton('30d')).toBeVisible();
                await expect(analyticsPage.getPresetButton('90d')).toBeVisible();
                await expect(analyticsPage.getPresetButton('All')).toBeVisible();

                await analyticsPage.selectDateRange('7d');
                await expect(analyticsPage.getPresetButton('7d')).toHaveClass(/bg-blue-600/);

                await analyticsPage.selectDateRange('All');
                await expect(analyticsPage.getPresetButton('All')).toHaveClass(/bg-blue-600/);
            });

            await test.step('AND the Top Models and Agent Activity leaderboards display consumption rankings and support sorting & filtering', async () => {
                await expect(analyticsPage.modelLeaderboardCard).toBeVisible();
                const sonnetModelRow = analyticsPage.getModelLeaderboardRow('claude-3-7-sonnet');
                await expect(sonnetModelRow).toBeVisible();
                await expect(sonnetModelRow).toContainText('claude-3-7-sonnet');
                await expect(sonnetModelRow).toContainText('$');

                // Filter models by search query
                await analyticsPage.modelSearchInput.fill('claude-3-7');
                await expect(analyticsPage.getModelLeaderboardRow('claude-3-7-sonnet')).toBeVisible();
                await analyticsPage.modelSearchInput.clear();

                // Sort models by Cost
                await analyticsPage.getModelSortButton('Cost').click();
                await expect(analyticsPage.getModelLeaderboardRow('claude-3-7-sonnet')).toBeVisible();

                await expect(analyticsPage.agentLeaderboardCard).toBeVisible();
                const cursorAgentRow = analyticsPage.getAgentLeaderboardRow('Cursor');
                await expect(cursorAgentRow).toBeVisible();
                await expect(cursorAgentRow).toContainText('Cursor');

                // Filter agents by search query
                await analyticsPage.agentSearchInput.fill('OpenCode');
                await expect(analyticsPage.getAgentLeaderboardRow('OpenCode')).toBeVisible();
                await analyticsPage.agentSearchInput.clear();

                // Sort agents by Cost
                await analyticsPage.getAgentSortButton('Cost').click();
                await expect(analyticsPage.getAgentLeaderboardRow('Cursor')).toBeVisible();
            });
        }
    );
});

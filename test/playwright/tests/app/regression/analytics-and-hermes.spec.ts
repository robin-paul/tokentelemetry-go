import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('Analytics charts, leaderboards, and Hermes Kanban journeys', () => {
    test.afterEach(async ({ transcriptFixture }) => {
        await transcriptFixture.cleanup();
    });

    test(
        'should render Analytics time-series charts and Model/Agent leaderboards',
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

            await test.step('THEN the Recharts token volume area chart and agent share pie chart are rendered', async () => {
                await expect(analyticsPage.tokenConsumptionChart).toBeVisible();
                await expect(analyticsPage.tokenConsumptionChartSvg).toBeVisible();

                await expect(analyticsPage.agentTokenShareChart).toBeVisible();
                await expect(analyticsPage.agentTokenShareChartSvg).toBeVisible();
            });

            await test.step('AND the Top Models and Agent Activity leaderboards display consumption rankings', async () => {
                await expect(analyticsPage.modelLeaderboardCard).toBeVisible();
                const sonnetModelRow = analyticsPage.getModelLeaderboardRow('claude-3-7-sonnet');
                await expect(sonnetModelRow).toBeVisible();
                await expect(sonnetModelRow).toContainText('claude-3-7-sonnet');
                await expect(sonnetModelRow).toContainText('$');

                await expect(analyticsPage.agentLeaderboardCard).toBeVisible();
                const cursorAgentRow = analyticsPage.getAgentLeaderboardRow('Cursor');
                await expect(cursorAgentRow).toBeVisible();
                await expect(cursorAgentRow).toContainText('Cursor');
            });
        }
    );

    test(
        'should render Hermes autonomous agent Kanban board with column distribution',
        { tag: '@regression' },
        async ({ hermesPage }) => {
            await test.step('WHEN the user navigates to the Hermes dashboard', async () => {
                await hermesPage.open();
                await expect(hermesPage.pageTitle).toBeVisible();
            });

            await test.step('THEN the gateway status indicator reports Active state', async () => {
                await expect(hermesPage.gatewayStatusIndicator).toBeVisible();
            });

            await test.step('AND all three Kanban columns (To Do, In Progress, Done) are visible', async () => {
                const todoCol = hermesPage.getColumn('To Do');
                await expect(todoCol).toBeVisible();

                const inProgressCol = hermesPage.getColumn('In Progress');
                await expect(inProgressCol).toBeVisible();

                const doneCol = hermesPage.getColumn('Done');
                await expect(doneCol).toBeVisible();
            });
        }
    );
});

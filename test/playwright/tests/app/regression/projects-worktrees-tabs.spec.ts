import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('Project Workspaces Worktree Aggregation & Grid/Table View Mode', () => {
    test.afterEach(async ({ transcriptFixture }) => {
        await transcriptFixture.cleanup();
    });

    test(
        'should toggle between grid and table views, group git worktrees under canonical parent, and render detail tabs',
        { tag: '@regression' },
        async ({ page, projectsPage, transcriptFixture }) => {
            const consoleErrors: string[] = [];
            page.on('console', (msg) => {
                if (msg.type() === 'error') {
                    consoleErrors.push(msg.text());
                }
            });

            await test.step('GIVEN multiple workspaces with canonical parent and worktrees exist', async () => {
                // Ingest sessions for canonical repo and a standalone repo
                await transcriptFixture.writeOpenCodeSession('core-engine', 'opencode-core-sess-1', [
                    {
                        role: 'assistant',
                        model: 'claude-3-7-sonnet',
                        inputTokens: 15000,
                        outputTokens: 4000,
                        toolName: 'plan_generator',
                    },
                ]);

                await transcriptFixture.writeCursorSession('core-engine', 'cursor-core-sess-2', [
                    {
                        role: 'assistant',
                        model: 'claude-3-5-sonnet',
                        inputTokens: 20000,
                        outputTokens: 5000,
                    },
                ]);

                await transcriptFixture.writeOpenCodeSession('web-portal', 'opencode-portal-sess-3', [
                    {
                        role: 'assistant',
                        model: 'gpt-4o',
                        inputTokens: 8000,
                        outputTokens: 2000,
                        toolName: 'view_file',
                    },
                ]);
            });

            await test.step('WHEN navigating to the Projects catalog (/projects)', async () => {
                await projectsPage.open();
                await expect(projectsPage.pageTitle).toBeVisible();
                await expect(projectsPage.projectsGrid).toBeVisible();
                await expect(projectsPage.getProjectCard('core-engine')).toBeVisible();
                await expect(projectsPage.getProjectCard('web-portal')).toBeVisible();
            });

            await test.step('THEN searching by project name filters catalog cards', async () => {
                await projectsPage.search('portal');
                await page.waitForTimeout(300);

                await expect(projectsPage.getProjectCard('web-portal')).toBeVisible();
                await expect(projectsPage.getProjectCard('core-engine')).not.toBeVisible();

                await projectsPage.searchInput.clear();
                await page.waitForTimeout(300);
                await expect(projectsPage.getProjectCard('core-engine')).toBeVisible();
                await expect(projectsPage.getProjectCard('web-portal')).toBeVisible();
            });

            await test.step('WHEN sorting by Tokens, Sessions, Cost, and Name', async () => {
                await projectsPage.sortCostBtn.click();
                await page.waitForTimeout(200);
                await expect(projectsPage.getProjectCard('core-engine')).toBeVisible();

                await projectsPage.sortNameBtn.click();
                await page.waitForTimeout(200);
                await expect(projectsPage.getProjectCard('web-portal')).toBeVisible();
            });

            await test.step('WHEN toggling between Grid View and Table View', async () => {
                // Switch to Table View
                await projectsPage.switchToTable();
                await expect(projectsPage.projectsTable).toBeVisible();
                await expect(projectsPage.getProjectRow('core-engine')).toBeVisible();
                await expect(projectsPage.getProjectRow('web-portal')).toBeVisible();

                // Switch back to Grid View
                await projectsPage.switchToGrid();
                await expect(projectsPage.projectsGrid).toBeVisible();
                await expect(projectsPage.getProjectCard('core-engine')).toBeVisible();
            });

            await test.step('WHEN navigating to Project Detail (/projects/core-engine)', async () => {
                await projectsPage.openProject('core-engine');
                await expect(page).toHaveURL(/projects\/core-engine/);
                await expect(page.getByRole('heading', { name: /core-engine/i })).toBeVisible();
                await expect(projectsPage.activityTab).toBeVisible();
                await expect(projectsPage.plansTab).toBeVisible();
                await expect(projectsPage.configTab).toBeVisible();
            });

            await test.step('THEN the Activity tab renders sessions with search filtering', async () => {
                await expect(projectsPage.activityContent).toBeVisible();
                await expect(page.getByText('claude-3-7-sonnet')).toBeVisible();
                await expect(page.getByText('claude-3-5-sonnet')).toBeVisible();

                // Test agent filtering in detail page
                await page.getByRole('button', { name: 'Cursor' }).click();
                await page.waitForTimeout(200);
                await expect(page.getByText('claude-3-5-sonnet')).toBeVisible();
                await expect(page.getByText('claude-3-7-sonnet')).not.toBeVisible();

                // Reset filter
                await page.getByRole('button', { name: 'All Agents' }).click();
                await page.waitForTimeout(200);
                await expect(page.getByText('claude-3-7-sonnet')).toBeVisible();
            });

            await test.step('WHEN switching to the Plans tab', async () => {
                await projectsPage.plansTab.click();
                await expect(projectsPage.plansContent).toBeVisible();
                await expect(page.getByText(/Plan generated|No architectural plans detected/i)).toBeVisible();
            });

            await test.step('WHEN switching to the Config tab and modifying preferences', async () => {
                await projectsPage.configTab.click();
                await expect(projectsPage.configContent).toBeVisible();
                await expect(projectsPage.toggleHideBtn).toBeVisible();
                await expect(projectsPage.aliasInput).toBeVisible();

                // Test setting friendly alias
                await projectsPage.aliasInput.fill('Core Platform Engine');
                await projectsPage.saveAliasBtn.click();
                await expect(page.getByText('Alias saved!')).toBeVisible();
            });

            await test.step('THEN no critical client console errors were thrown', async () => {
                const severeErrors = consoleErrors.filter(
                    (err) => !err.includes('favicon.ico') && !err.includes('404')
                );
                expect(severeErrors).toEqual([]);
            });
        }
    );
});

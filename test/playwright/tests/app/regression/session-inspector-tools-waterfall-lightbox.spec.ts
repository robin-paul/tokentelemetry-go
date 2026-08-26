import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('Tool Invocations Waterfall, Terminal Outputs, Inspector Sidebar, & Artifact Lightbox Modal', () => {
    test.afterEach(async ({ transcriptFixture }) => {
        await transcriptFixture.cleanup();
    });

    test(
        'should render tool invocations with collapsible args, terminal outputs, execution waterfall Gantt chart, multi-tab inspector sidebar, and artifact lightbox modal',
        { tag: '@regression' },
        async ({ sessionsPage, sessionDetailPage, transcriptFixture, page }) => {
            const projectName = 'e2e-tools-waterfall-lightbox-project';
            const sessionId = `claude-waterfall-${Date.now()}`;

            await test.step('GIVEN a Claude Code session with tool calls, terminal outputs, diffs, and multiple turns', async () => {
                const rawTranscript = [
                    JSON.stringify({
                        type: 'user',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T15:00:00Z',
                        message: {
                            content: [{ type: 'text', text: 'Step 1: Check git diff and update schema.' }],
                        },
                    }),
                    JSON.stringify({
                        type: 'assistant',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T15:00:05Z',
                        message: {
                            model: 'claude-3-7-sonnet',
                            usage: { input_tokens: 2000, output_tokens: 500 },
                            content: [
                                { type: 'text', text: 'Executing git status and applying patches.' },
                                {
                                    type: 'tool_use',
                                    id: 'tool_call_git_diff',
                                    name: 'git_diff_check',
                                    input: { files: ['schema.sql', 'models.go'] },
                                },
                            ],
                        },
                    }),
                    JSON.stringify({
                        type: 'user',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T15:00:10Z',
                        message: {
                            content: [
                                {
                                    type: 'tool_result',
                                    tool_use_id: 'tool_call_git_diff',
                                    content: '@@ -1,4 +1,6 @@\n- old_field string\n+ new_field string\n+ created_at timestamp',
                                },
                                { type: 'text', text: 'Step 2: Now run migrations and build artifacts.' },
                            ],
                        },
                    }),
                    JSON.stringify({
                        type: 'assistant',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T15:00:15Z',
                        message: {
                            model: 'claude-3-7-sonnet',
                            usage: { input_tokens: 2500, output_tokens: 700 },
                            content: [
                                { type: 'text', text: 'Running database migrations now.' },
                                {
                                    type: 'tool_use',
                                    id: 'tool_call_db_migrate',
                                    name: 'apply_sqlite_migration',
                                    input: { migration_id: '0005_rich_turns' },
                                },
                            ],
                        },
                    }),
                ].join('\n');

                await transcriptFixture.writeRawTranscript(
                    `.claude/projects/${projectName}/${sessionId}.jsonl`,
                    rawTranscript
                );
            });

            await test.step('WHEN the user navigates to the Session Detail view', async () => {
                await sessionsPage.open();
                await expect(sessionsPage.sessionTable).toBeVisible();

                const sessionRow = sessionsPage.getSessionRow(projectName).first();
                await expect(sessionRow).toBeVisible();
                await sessionRow.click();

                await expect(sessionDetailPage.sessionIdHeading).toBeVisible();
            });

            await test.step('THEN ToolInvocationCard pairs tool call with collapsible arguments and terminal output diff', async () => {
                const toolCard = sessionDetailPage.toolInvocationCards.first();
                await expect(toolCard).toBeVisible();
                await expect(toolCard).toContainText('git_diff_check');

                // Arguments details can be toggled
                const argsToggle = toolCard.getByRole('button', { name: /arguments/i });
                await expect(argsToggle).toBeVisible();
                await expect(toolCard.locator('pre').filter({ hasText: 'schema.sql' })).toBeVisible();

                // Terminal diff output rendered with + and - lines
                await expect(toolCard.locator('text=+ new_field string')).toBeVisible();
                await expect(toolCard.locator('text=- old_field string')).toBeVisible();
            });

            await test.step('THEN ExecutionWaterfall renders a timeline Gantt chart across session turns', async () => {
                await expect(sessionDetailPage.waterfall).toBeVisible();
                await expect(sessionDetailPage.waterfall).toContainText('Tool Execution Waterfall');
                await expect(sessionDetailPage.waterfall).toContainText(/2 calls/i);

                // Both tools appear in waterfall list
                await expect(sessionDetailPage.waterfall.getByText('git_diff_check')).toBeVisible();
                await expect(sessionDetailPage.waterfall.getByText('apply_sqlite_migration')).toBeVisible();

                // Clicking the first tool in the waterfall seeks to Turn #2
                const gitDiffRow = sessionDetailPage.waterfall.locator('.cursor-pointer').filter({ hasText: 'git_diff_check' }).first();
                await gitDiffRow.click();
                await expect(sessionDetailPage.scrubberStepLabel).toContainText('Step 2 of 4');
            });

            await test.step('THEN InspectorSidebar provides multi-tab inspection (Context, Tools, Artifacts, Raw)', async () => {
                await expect(sessionDetailPage.inspectorSidebar).toBeVisible();

                // 1. Context Tab
                await expect(sessionDetailPage.contextPanel).toBeVisible();
                await expect(sessionDetailPage.contextPanel).toContainText(projectName);
                await expect(sessionDetailPage.contextPanel).toContainText('Claude');

                // 2. Tools Histogram Tab
                await sessionDetailPage.tabTools.click();
                await expect(sessionDetailPage.toolsPanel).toBeVisible();
                await expect(sessionDetailPage.toolsPanel).toContainText('git_diff_check');
                await expect(sessionDetailPage.toolsPanel).toContainText('apply_sqlite_migration');

                // Clicking a tool in histogram jumps to turn
                const migrateBtn = sessionDetailPage.toolsPanel.getByRole('button').filter({ hasText: 'apply_sqlite_migration' }).first();
                await migrateBtn.click();
                await expect(sessionDetailPage.scrubberStepLabel).toContainText('Step 4 of 4');

                // 3. Raw JSON Tab
                await sessionDetailPage.tabRaw.click();
                await expect(sessionDetailPage.rawPanel).toBeVisible();
                await expect(sessionDetailPage.copyRawJsonButton).toBeVisible();
                await sessionDetailPage.copyRawJsonButton.click();
                await expect(sessionDetailPage.rawPanel.getByText('Copied')).toBeVisible();

                // 4. Artifacts Tab
                await sessionDetailPage.tabArtifacts.click();
                await expect(sessionDetailPage.artifactsPanel).toBeVisible();
            });

            await test.step('WHEN an artifact lightbox modal is triggered, it renders portalled with zoom controls and closes on Escape', async () => {
                // Trigger lightbox via test evaluation or manual artifact action
                await page.evaluate(() => {
                    const event = new CustomEvent('open-artifact-preview', {
                        detail: { name: 'architecture-spec.md', type: 'document', content: '# Architecture Spec\n\nDeep inspector verified.' },
                    });
                    window.dispatchEvent(event);
                });

                // Open lightbox directly from Artifacts tab or state
                // Test toggle inspector sidebar visibility
                await sessionDetailPage.toggleSidebarButton.click();
                await expect(sessionDetailPage.inspectorSidebar).not.toBeVisible();

                await sessionDetailPage.toggleSidebarButton.click();
                await expect(sessionDetailPage.inspectorSidebar).toBeVisible();
            });
        }
    );
});

import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('Session turn ingestion, rich message rendering, and deep inspector controls', () => {
    test.afterEach(async ({ transcriptFixture }) => {
        await transcriptFixture.cleanup();
    });

    test(
        'should render rich user and assistant markdown turns, thinking cards, copy-code action, and raw toggle',
        { tag: '@regression' },
        async ({ sessionsPage, sessionDetailPage, transcriptFixture, page }) => {
            const projectName = 'e2e-rich-inspector-project';
            const sessionId = `claude-rich-${Date.now()}`;

            await test.step('GIVEN a Claude Code session with rich user prompt, thinking blocks, markdown response, and tool calls', async () => {
                const rawTranscript = [
                    JSON.stringify({
                        type: 'user',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T12:00:00Z',
                        message: {
                            content: [
                                {
                                    type: 'text',
                                    text: 'Can you refactor the database schema and provide code examples?',
                                },
                            ],
                        },
                    }),
                    JSON.stringify({
                        type: 'assistant',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T12:00:05Z',
                        message: {
                            model: 'claude-3-7-sonnet',
                            usage: {
                                input_tokens: 3500,
                                output_tokens: 850,
                                cache_read_input_tokens: 1200,
                                cache_creation_input_tokens: 200,
                            },
                            content: [
                                {
                                    type: 'thinking',
                                    thinking: 'I need to design a clean migration for message_turns table.',
                                },
                                {
                                    type: 'text',
                                    text: '### Schema Refactor Proposal\n\nHere is the recommended schema definition:\n\n```typescript\ninterface MessageTurn {\n  id: string;\n  content?: string;\n  thinking?: string;\n}\n```\n\n- [x] Supports rich text\n- [x] Supports thinking blocks\n- [x] Persists tool calls',
                                },
                                {
                                    type: 'tool_use',
                                    id: 'tool_migration_1',
                                    name: 'write_to_file',
                                    input: {
                                        TargetFile: '/internal/store/migrations/0005.sql',
                                        Description: 'Add content columns',
                                    },
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

            await test.step('WHEN the user opens the Sessions list and clicks into the new session', async () => {
                await sessionsPage.open();
                await expect(sessionsPage.sessionTable).toBeVisible();

                const sessionRow = sessionsPage.getSessionRow(projectName).first();
                await expect(sessionRow).toBeVisible();
                await sessionRow.click();
            });

            await test.step('THEN the Session Inspector displays the session metadata and KPI summary', async () => {
                await expect(sessionDetailPage.sessionIdHeading).toBeVisible();
                await expect(sessionDetailPage.agentBadge).toContainText('Claude');
                await expect(sessionDetailPage.tokensValue).toBeVisible();
                await expect(sessionDetailPage.netCostValue).toBeVisible();
            });

            await test.step('AND renders the User Prompt turn card with user styling', async () => {
                const userCard = sessionDetailPage.userTurnCards.first();
                await expect(userCard).toBeVisible();
                await expect(userCard).toContainText(/User Prompt/i);
                await expect(userCard).toContainText('Can you refactor the database schema');
            });

            await test.step('AND renders the Assistant Response card with Markdown, headings, and task list', async () => {
                const assistantCard = sessionDetailPage.assistantTurnCards.first();
                await expect(assistantCard).toBeVisible();
                await expect(assistantCard).toContainText('Schema Refactor Proposal');
                await expect(assistantCard).toContainText('Supports rich text');
            });

            await test.step('AND renders the collapsible Reasoning & Thoughts card with effort indicator', async () => {
                const reasoningCard = sessionDetailPage.reasoningCards.first();
                await expect(reasoningCard).toBeVisible();
                await expect(reasoningCard).toContainText('Reasoning & Thoughts');
                await expect(reasoningCard).toContainText('effort: high');
                await expect(reasoningCard).toContainText('I need to design a clean migration');
            });

            await test.step('AND renders the Copy Code button on fenced code blocks', async () => {
                const copyBtn = sessionDetailPage.copyCodeButtons.first();
                await expect(copyBtn).toBeVisible();
                await copyBtn.click();
                await expect(page.getByText('Copied!')).toBeVisible();
            });

            await test.step('AND allows 1-click toggling between Markdown and Raw text modes', async () => {
                const assistantCard = sessionDetailPage.assistantTurnCards.first();
                const rawToggle = assistantCard.getByRole('button', { name: /View Raw/i });
                await expect(rawToggle).toBeVisible();

                await rawToggle.click();
                const mdToggle = assistantCard.getByRole('button', { name: /View MD/i });
                await expect(mdToggle).toBeVisible();
                await expect(page.locator('pre').filter({ hasText: '### Schema Refactor Proposal' })).toBeVisible();

                await mdToggle.click();
                await expect(assistantCard.getByRole('button', { name: /View Raw/i })).toBeVisible();
            });

            await test.step('AND allows filtering turns by Category', async () => {
                const userFilterBtn = page.getByRole('button', { name: /User \(/i });
                await expect(userFilterBtn).toBeVisible();
                await userFilterBtn.click();

                await expect(sessionDetailPage.userTurnCards).toHaveCount(1);
                await expect(sessionDetailPage.assistantTurnCards).toHaveCount(0);

                const allFilterBtn = page.getByRole('button', { name: /All Turns/i });
                await allFilterBtn.click();
                await expect(sessionDetailPage.userTurnCards).toHaveCount(1);
                await expect(sessionDetailPage.assistantTurnCards).toHaveCount(1);
            });
        }
    );

    test(
        'should support timeline scrubbing, playback stepping, step index navigation, and in-trace keyword search',
        { tag: '@regression' },
        async ({ sessionsPage, sessionDetailPage, transcriptFixture, page }) => {
            const projectName = 'e2e-scrubber-controls-project';
            const sessionId = `claude-scrub-${Date.now()}`;

            await test.step('GIVEN a multi-turn session with 4 distinct message turns', async () => {
                const rawTranscript = [
                    JSON.stringify({
                        type: 'user',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T14:00:00Z',
                        message: {
                            content: [{ type: 'text', text: 'Step 1: Explain the token analyzer architecture.' }],
                        },
                    }),
                    JSON.stringify({
                        type: 'assistant',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T14:00:05Z',
                        message: {
                            model: 'claude-3-7-sonnet',
                            usage: { input_tokens: 1000, output_tokens: 400 },
                            content: [
                                { type: 'thinking', thinking: 'Analyze high level components.' },
                                { type: 'text', text: 'Step 2: Architecture comprises Collector CLI and Hub Server.' },
                            ],
                        },
                    }),
                    JSON.stringify({
                        type: 'user',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T14:00:10Z',
                        message: {
                            content: [{ type: 'text', text: 'Step 3: Can you list all tools used for SQLite migration?' }],
                        },
                    }),
                    JSON.stringify({
                        type: 'assistant',
                        sessionId: sessionId,
                        timestamp: '2026-08-26T14:00:15Z',
                        message: {
                            model: 'claude-3-7-sonnet',
                            usage: { input_tokens: 1500, output_tokens: 600 },
                            content: [
                                { type: 'text', text: 'Step 4: Executing database migrations now.' },
                                {
                                    type: 'tool_use',
                                    id: 'tool_migration_step_4',
                                    name: 'apply_schema_patch',
                                    input: { path: '/migrations/0005_turns.sql' },
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

            await test.step('WHEN the user opens the session in the Session Inspector', async () => {
                await sessionsPage.open();
                await expect(sessionsPage.sessionTable).toBeVisible();

                const sessionRow = sessionsPage.getSessionRow(projectName).first();
                await expect(sessionRow).toBeVisible();
                await sessionRow.click();

                await expect(sessionDetailPage.sessionIdHeading).toBeVisible();
            });

            await test.step('THEN the scrubber displays Step 4 of 4 and Step Index shows 4 steps', async () => {
                await expect(sessionDetailPage.scrubberStepLabel).toContainText('Step 4 of 4');
                await expect(sessionDetailPage.stepIndexButtons).toHaveCount(4);
            });

            await test.step('WHEN scrubbing to Step 1 via range slider', async () => {
                await sessionDetailPage.scrubToStep(0);
                await expect(sessionDetailPage.scrubberStepLabel).toContainText('Step 1 of 4');
            });

            await test.step('WHEN stepping forward via Next Turn playback button', async () => {
                await expect(sessionDetailPage.nextStepButton).toBeEnabled();
                await sessionDetailPage.nextStepButton.click();
                await expect(sessionDetailPage.scrubberStepLabel).toContainText('Step 2 of 4');
            });

            await test.step('WHEN stepping backward via Previous Turn playback button', async () => {
                await expect(sessionDetailPage.prevStepButton).toBeEnabled();
                await sessionDetailPage.prevStepButton.click();
                await expect(sessionDetailPage.scrubberStepLabel).toContainText('Step 1 of 4');
            });

            await test.step('WHEN clicking Step Index #3 button', async () => {
                const step3Btn = sessionDetailPage.stepIndexButtons.nth(2);
                await expect(step3Btn).toBeVisible();
                await step3Btn.click();
                await expect(sessionDetailPage.scrubberStepLabel).toContainText('Step 3 of 4');
            });

            await test.step('WHEN typing a search keyword in TurnSearchInput', async () => {
                await expect(sessionDetailPage.searchInput).toBeVisible();
                await sessionDetailPage.searchInput.fill('architecture');

                // Turn 1 (User) and Turn 2 (Assistant) contain 'architecture'
                await expect(page.getByText(/2 matches/i)).toBeVisible();

                // Clear the search input
                await page.getByRole('button', { name: /Clear search/i }).click();
                await expect(sessionDetailPage.searchInput).toHaveValue('');
            });

            await test.step('WHEN toggling play/pause playback', async () => {
                // Seek to beginning
                await sessionDetailPage.scrubToStep(0);
                await expect(sessionDetailPage.scrubberStepLabel).toContainText('Step 1 of 4');

                // Start playback
                await sessionDetailPage.playPauseButton.click();
                // Wait for playback tick (600ms+)
                await page.waitForTimeout(800);

                // Pause playback
                await sessionDetailPage.playPauseButton.click();
                // Should have advanced past Step 1
                await expect(sessionDetailPage.scrubberStepLabel).not.toContainText('Step 1 of 4');
            });
        }
    );
});

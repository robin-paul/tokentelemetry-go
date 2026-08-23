import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('GitHub Copilot synthetic ingestion and custom pricing overrides', () => {
    test.afterEach(async ({ transcriptFixture }) => {
        await transcriptFixture.cleanup();
    });

    test(
        'should apply custom model pricing override from Settings to subsequent Copilot sessions',
        { tag: '@regression' },
        async ({ settingsPage, sessionsPage, transcriptFixture }) => {
            const modelPattern = 'gpt-4o-custom-e2e';
            const projectName = 'e2e-copilot-project';
            const sessionId = `copilot-test-${Date.now()}`;

            await test.step('GIVEN the user opens the Settings page', async () => {
                await settingsPage.open();
                await expect(settingsPage.pageTitle).toBeVisible();
                await expect(settingsPage.modelPatternInput).toBeVisible();
            });

            await test.step('WHEN the user adds a custom pricing override for a model pattern', async () => {
                await settingsPage.addPricingOverride(modelPattern, 10.0, 30.0);
                await expect(settingsPage.statusMessage).toBeVisible();

                const overrideRow = settingsPage.getOverrideRow(modelPattern);
                await expect(overrideRow).toBeVisible();
                await expect(overrideRow).toContainText('$10.00');
                await expect(overrideRow).toContainText('$30.00');
            });

            await test.step('AND a Copilot session using the overridden model is written to disk', async () => {
                await transcriptFixture.writeCopilotSession(projectName, sessionId, {
                    requests: [
                        {
                            modelId: modelPattern,
                            completionTokens: 10000,
                            userPromptText: 'Explain distributed consensus algorithms in detail.',
                        },
                    ],
                });
            });

            await test.step('THEN the sessions list displays the Copilot session with overridden cost calculation', async () => {
                await sessionsPage.open();
                const sessionRow = sessionsPage.getSessionRow(projectName).first();
                await expect(sessionRow).toBeVisible();
                await expect(sessionRow).toContainText('GitHub Copilot');
                await expect(sessionRow).toContainText(modelPattern);
                await expect(sessionRow).toContainText('$0.30');
            });

            await test.step('AND the user can clean up the pricing override from Settings', async () => {
                await settingsPage.open();
                await settingsPage.deleteOverride(modelPattern);
                await expect(settingsPage.getOverrideRow(modelPattern)).toHaveCount(0);
            });
        }
    );
});

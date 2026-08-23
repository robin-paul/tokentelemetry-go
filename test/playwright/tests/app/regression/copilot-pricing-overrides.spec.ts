import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('GitHub Copilot synthetic ingestion and custom pricing overrides', () => {
    const modelPattern = 'gpt-4o-custom-e2e';

    test.afterEach(async ({ settingsPage, transcriptFixture }) => {
        try {
            await settingsPage.open();
            const row = settingsPage.getOverrideRow(modelPattern);
            if (await row.isVisible().catch(() => false)) {
                await settingsPage.deleteOverride(modelPattern);
            }
        } catch {}
        await transcriptFixture.cleanup();
    });

    test(
        'should apply custom model pricing override from Settings to subsequent Copilot sessions',
        { tag: '@regression' },
        async ({ settingsPage, sessionsPage, transcriptFixture }) => {
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
                            userPromptText: 'A'.repeat(40000), // ~10,000 prompt tokens
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
                await expect(sessionRow).toContainText('$0.40');
            });
        }
    );
});

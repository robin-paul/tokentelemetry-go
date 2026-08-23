import { expect, test } from '../../../fixtures/pom/test-options';

test.describe('Layout shell and navigation positioning', () => {
    test(
        'should render navbar side-by-side with main content without horizontal overlap or vertical displacement',
        { tag: '@regression' },
        async ({ page, dashboardPage, navbar }) => {
            await test.step('GIVEN the user opens the application', async () => {
                await dashboardPage.open();
                await expect(navbar.brandTitle).toBeVisible();
            });

            await test.step('THEN navbar and main content are positioned side-by-side from top: 0', async () => {
                const aside = page.locator('aside');
                const main = page.locator('main');

                await expect(aside).toBeVisible();
                await expect(main).toBeVisible();

                const asideBox = await aside.boundingBox();
                const mainBox = await main.boundingBox();

                expect(asideBox).not.toBeNull();
                expect(mainBox).not.toBeNull();

                if (asideBox && mainBox) {
                    // Main content should start at or near the top of the viewport (not pushed below aside)
                    expect(mainBox.y).toBeLessThanOrEqual(10);
                    // Main content should be placed to the right of the navbar without overlap
                    expect(mainBox.x).toBeGreaterThanOrEqual(asideBox.x + asideBox.width - 5);
                }
            });

            await test.step('AND navigating to other routes maintains side-by-side layout', async () => {
                const aside = page.locator('aside');
                const main = page.locator('main');

                for (const navigateAction of [
                    () => navbar.clickSessions(),
                    () => navbar.clickProjects(),
                    () => navbar.clickAnalytics(),
                    () => navbar.clickSettings(),
                ]) {
                    await navigateAction();
                    await expect(main).toBeVisible();

                    const asideBox = await aside.boundingBox();
                    const mainBox = await main.boundingBox();

                    expect(asideBox).not.toBeNull();
                    expect(mainBox).not.toBeNull();

                    if (asideBox && mainBox) {
                        expect(mainBox.y).toBeLessThanOrEqual(10);
                        expect(mainBox.x).toBeGreaterThanOrEqual(asideBox.x + asideBox.width - 5);
                    }
                }
            });
        }
    );
});

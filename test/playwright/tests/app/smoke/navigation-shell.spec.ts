import { expect, test } from '../../../fixtures/pom/test-options';
import { AppRoutes } from '../../../enums/app/app';

test.describe('navigation and dashboard shell', () => {
    test(
        'should render the dashboard shell and navigate across all views without errors',
        { tag: '@smoke' },
        async ({ page, dashboardPage, navbar }) => {
            const consoleErrors: string[] = [];
            page.on('console', (msg) => {
                if (msg.type() === 'error') {
                    consoleErrors.push(msg.text());
                }
            });

            await test.step('GIVEN the user opens the TokenTelemetry dashboard', async () => {
                await dashboardPage.open();
                await expect(navbar.brandTitle).toBeVisible();
                await expect(navbar.liveIndicator).toBeVisible();
            });

            await test.step('THEN all primary navigation items are visible', async () => {
                await expect(navbar.overviewLink).toBeVisible();
                await expect(navbar.sessionsLink).toBeVisible();
                await expect(navbar.projectsLink).toBeVisible();
                await expect(navbar.analyticsLink).toBeVisible();
                await expect(navbar.hermesLink).toBeVisible();
                await expect(navbar.settingsLink).toBeVisible();
            });

            await test.step('WHEN the user navigates to Sessions', async () => {
                await navbar.clickSessions();
                await expect(page).toHaveURL(new RegExp(AppRoutes.SESSIONS));
            });

            await test.step('WHEN the user navigates to Analytics', async () => {
                await navbar.clickAnalytics();
                await expect(page).toHaveURL(new RegExp(AppRoutes.ANALYTICS));
            });

            await test.step('WHEN the user navigates to Hermes', async () => {
                await navbar.clickHermes();
                await expect(page).toHaveURL(new RegExp(AppRoutes.HERMES));
            });

            await test.step('WHEN the user navigates to Settings', async () => {
                await navbar.clickSettings();
                await expect(page).toHaveURL(new RegExp(AppRoutes.SETTINGS));
            });

            await test.step('WHEN the user navigates back to Overview', async () => {
                await navbar.clickOverview();
                await expect(page).toHaveURL(
                    new RegExp(`^http://127.0.0.1:8000/?$`)
                );
                await expect(dashboardPage.trendsChart).toBeVisible();
            });

            await test.step('THEN no critical client console errors were thrown', async () => {
                const severeErrors = consoleErrors.filter(
                    (err) =>
                        !err.includes('favicon.ico') && !err.includes('404')
                );
                expect(severeErrors).toEqual([]);
            });
        }
    );
});

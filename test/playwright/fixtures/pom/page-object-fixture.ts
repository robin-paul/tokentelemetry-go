import { test as base } from '@playwright/test';
import { DashboardPage } from '../../pages/app/dashboard.page';
import { NavbarComponent } from '../../pages/components/navbar.component';

/**
 * Framework fixtures for page objects.
 */
export type FrameworkFixtures = {
    /** TokenTelemetry Overview dashboard page object */
    dashboardPage: DashboardPage;
    /** Navigation sidebar component object */
    navbar: NavbarComponent;
};

/**
 * Extended test with page object fixtures.
 */
export const test = base.extend<FrameworkFixtures>({
    dashboardPage: async ({ page }, use) => {
        await use(new DashboardPage(page));
    },
    navbar: async ({ page }, use) => {
        await use(new NavbarComponent(page));
    },
});

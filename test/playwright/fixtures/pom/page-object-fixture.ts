import { test as base } from '@playwright/test';
import { DashboardPage } from '../../pages/app/dashboard.page';
import { SessionsPage } from '../../pages/app/sessions.page';
import { SessionDetailPage } from '../../pages/app/session-detail.page';
import { SettingsPage } from '../../pages/app/settings.page';
import { AnalyticsPage } from '../../pages/app/analytics.page';
import { NavbarComponent } from '../../pages/components/navbar.component';

/**
 * Framework fixtures for page objects.
 */
export type FrameworkFixtures = {
    /** TokenTelemetry Overview dashboard page object */
    dashboardPage: DashboardPage;
    /** Sessions list page object */
    sessionsPage: SessionsPage;
    /** Session detail inspector page object */
    sessionDetailPage: SessionDetailPage;
    /** Settings and pricing configuration page object */
    settingsPage: SettingsPage;
    /** Analytics and leaderboards page object */
    analyticsPage: AnalyticsPage;
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
    sessionsPage: async ({ page }, use) => {
        await use(new SessionsPage(page));
    },
    sessionDetailPage: async ({ page }, use) => {
        await use(new SessionDetailPage(page));
    },
    settingsPage: async ({ page }, use) => {
        await use(new SettingsPage(page));
    },
    analyticsPage: async ({ page }, use) => {
        await use(new AnalyticsPage(page));
    },
    navbar: async ({ page }, use) => {
        await use(new NavbarComponent(page));
    },
});

import { Locator, Page } from '@playwright/test';

/**
 * Component Object for the TokenTelemetry sidebar / navigation bar.
 */
export class NavbarComponent {
    constructor(private readonly page: Page) {}

    // ==================== Locators ====================

    get brandTitle(): Locator {
        return this.page.getByText('TokenTelemetry');
    }

    get liveIndicator(): Locator {
        return this.page.getByText(/Live Telemetry|Connecting\.\.\./);
    }

    get overviewLink(): Locator {
        return this.page.getByRole('link', { name: 'Overview' });
    }

    get sessionsLink(): Locator {
        return this.page.getByRole('link', { name: 'Sessions' });
    }

    get projectsLink(): Locator {
        return this.page.getByRole('link', { name: 'Projects' });
    }

    get analyticsLink(): Locator {
        return this.page.getByRole('link', { name: 'Analytics' });
    }

    get hermesLink(): Locator {
        return this.page.getByRole('link', { name: 'Hermes' });
    }

    get settingsLink(): Locator {
        return this.page.getByRole('link', { name: 'Settings' });
    }

    get themeToggleButton(): Locator {
        return this.page.getByRole('button', { name: /Switch to/i });
    }

    get activeAgentsPill(): Locator {
        return this.page.getByText(/Active Agents Detected/);
    }

    // ==================== Actions ====================

    /**
     * Navigates to the Overview / Dashboard page.
     */
    async clickOverview(): Promise<void> {
        await this.overviewLink.click();
    }

    /**
     * Navigates to the Sessions list page.
     */
    async clickSessions(): Promise<void> {
        await this.sessionsLink.click();
    }

    /**
     * Navigates to the Projects catalog page.
     */
    async clickProjects(): Promise<void> {
        await this.projectsLink.click();
    }

    /**
     * Navigates to the Analytics visualization page.
     */
    async clickAnalytics(): Promise<void> {
        await this.analyticsLink.click();
    }

    /**
     * Navigates to the Hermes agent dashboard.
     */
    async clickHermes(): Promise<void> {
        await this.hermesLink.click();
    }

    /**
     * Navigates to the Settings page.
     */
    async clickSettings(): Promise<void> {
        await this.settingsLink.click();
    }

    /**
     * Toggles the UI theme between dark and light.
     */
    async toggleTheme(): Promise<void> {
        await this.themeToggleButton.click();
    }
}

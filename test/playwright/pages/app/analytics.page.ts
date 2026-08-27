import { Locator, Page } from '@playwright/test';
import { NavbarComponent } from '../components/navbar.component';
import { AppRoutes } from '../../enums/app/app';

/**
 * Page Object for the Analytics & Leaderboards view (/analytics).
 */
export class AnalyticsPage {
    readonly nav: NavbarComponent;

    constructor(readonly page: Page) {
        this.nav = new NavbarComponent(page);
    }

    // ==================== Locators ====================

    get pageTitle(): Locator {
        return this.page.getByRole('heading', { level: 1 });
    }

    get tokenConsumptionChart(): Locator {
        return this.page
            .locator('div.rounded-xl')
            .filter({ has: this.page.getByText('Token Consumption Volume', { exact: true }) });
    }

    get tokenConsumptionChartSvg(): Locator {
        return this.tokenConsumptionChart.locator('svg.recharts-surface');
    }

    get agentTokenShareChart(): Locator {
        return this.page
            .locator('div.rounded-xl')
            .filter({ has: this.page.getByText('Agent Token Share', { exact: true }) });
    }

    get agentTokenShareChartSvg(): Locator {
        return this.agentTokenShareChart.locator('svg.recharts-surface');
    }

    get modelLeaderboardCard(): Locator {
        return this.page
            .locator('div.rounded-xl')
            .filter({ has: this.page.getByText('Top Models by Consumption', { exact: true }) });
    }

    get agentLeaderboardCard(): Locator {
        return this.page
            .locator('div.rounded-xl')
            .filter({ has: this.page.getByText('Agent Activity Leaderboard', { exact: true }) });
    }

    get modelSearchInput(): Locator {
        return this.modelLeaderboardCard.getByPlaceholder(/Filter models/i);
    }

    get agentSearchInput(): Locator {
        return this.agentLeaderboardCard.getByPlaceholder(/Filter agents/i);
    }

    getPresetButton(preset: '7d' | '30d' | '90d' | 'All' | 'Custom'): Locator {
        return this.page.getByRole('button', { name: new RegExp(`^${preset}$`, 'i') });
    }

    getModelSortButton(field: 'Volume' | 'Cost' | 'Sessions'): Locator {
        return this.modelLeaderboardCard.getByRole('button', { name: new RegExp(`^${field}$`, 'i') });
    }

    getAgentSortButton(field: 'Volume' | 'Cost' | 'Sessions'): Locator {
        return this.agentLeaderboardCard.getByRole('button', { name: new RegExp(`^${field}$`, 'i') });
    }

    // ==================== Actions ====================

    /**
     * Opens the Analytics page at '/analytics'.
     */
    async open(): Promise<void> {
        await this.page.goto(AppRoutes.ANALYTICS, {
            waitUntil: 'domcontentloaded',
        });
    }

    async selectDateRange(preset: '7d' | '30d' | '90d' | 'All'): Promise<void> {
        await this.getPresetButton(preset).click();
    }

    getModelLeaderboardRow(modelName: string): Locator {
        return this.modelLeaderboardCard
            .locator('div.rounded-lg')
            .filter({ hasText: modelName });
    }

    getAgentLeaderboardRow(agentName: string): Locator {
        return this.agentLeaderboardCard
            .locator('div.rounded-lg')
            .filter({ hasText: new RegExp(agentName, 'i') });
    }
}

import { Locator, Page } from '@playwright/test';
import { NavbarComponent } from '../components/navbar.component';
import { AppRoutes } from '../../enums/app/app';

/**
 * Page Object for the main TokenTelemetry Dashboard (Overview).
 */
export class DashboardPage {
    readonly nav: NavbarComponent;

    constructor(private readonly page: Page) {
        this.nav = new NavbarComponent(page);
    }

    // ==================== Locators ====================

    get pageTitle(): Locator {
        return this.page
            .getByRole('heading', { name: 'Overview' })
            .or(this.nav.brandTitle);
    }

    get totalTokensCard(): Locator {
        return this.page.getByText('Total Tokens');
    }

    get netCostCard(): Locator {
        return this.page.getByText('Net Billable Cost');
    }

    get indexedSessionsCard(): Locator {
        return this.page.getByText('Indexed Sessions');
    }

    get activeAgentsCard(): Locator {
        return this.page.getByText('Active Ecosystem');
    }

    get trendsChart(): Locator {
        return this.page.getByText('14-Day Token Consumption Trends');
    }

    get liveFeedHeading(): Locator {
        return this.page.getByText('Live Activity Feed');
    }

    get recentSessionsTable(): Locator {
        return this.page.getByRole('table');
    }

    get recentSessionRows(): Locator {
        return this.page.getByRole('row');
    }

    get emptyFeedNotice(): Locator {
        return this.page.getByText(
            /No agent sessions detected yet|No historical trend data/
        );
    }

    // ==================== Actions ====================

    /**
     * Opens the Dashboard page at root '/'.
     */
    async open(): Promise<void> {
        await this.page.goto(AppRoutes.OVERVIEW, {
            waitUntil: 'domcontentloaded',
        });
    }

    getSessionRow(identifier: string): Locator {
        return this.recentSessionRows.filter({ hasText: identifier });
    }
}

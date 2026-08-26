import { Locator, Page } from '@playwright/test';
import { NavbarComponent } from '../components/navbar.component';
import { AppRoutes } from '../../enums/app/app';

/**
 * Page Object for the Sessions list view (/sessions).
 */
export class SessionsPage {
    readonly nav: NavbarComponent;

    constructor(readonly page: Page) {
        this.nav = new NavbarComponent(page);
    }

    // ==================== Locators ====================

    get pageTitle(): Locator {
        return this.page.getByRole('heading', { name: 'Sessions' }).or(this.nav.brandTitle);
    }

    get searchInput(): Locator {
        return this.page.getByPlaceholder(/Search session/i).or(this.page.getByPlaceholder(/Search/i));
    }

    get allAgentsFilterButton(): Locator {
        return this.page.getByRole('button', { name: /All Agents/i });
    }

    get modelSelect(): Locator {
        return this.page.locator('select').first();
    }

    get datePresetAll(): Locator {
        return this.page.getByRole('button', { name: 'All Time' });
    }

    get datePresetToday(): Locator {
        return this.page.getByRole('button', { name: 'Today' });
    }

    get datePreset7d(): Locator {
        return this.page.getByRole('button', { name: '7 Days' });
    }

    get datePreset30d(): Locator {
        return this.page.getByRole('button', { name: '30 Days' });
    }

    get rangeFiltersButton(): Locator {
        return this.page.getByRole('button', { name: /Range Filters/i });
    }

    get minCostInput(): Locator {
        return this.page.getByPlaceholder('e.g. 0.05');
    }

    get maxCostInput(): Locator {
        return this.page.getByPlaceholder('e.g. 5.00');
    }

    get minTokensInput(): Locator {
        return this.page.getByPlaceholder('e.g. 10000');
    }

    get maxTokensInput(): Locator {
        return this.page.getByPlaceholder('e.g. 500000');
    }

    get resetFiltersButton(): Locator {
        return this.page.getByRole('button', { name: /Reset/i });
    }

    get sessionTable(): Locator {
        return this.page.getByRole('table');
    }

    get sessionRows(): Locator {
        return this.page.getByRole('row');
    }

    get emptyNotice(): Locator {
        return this.page.getByText(/No sessions match the selected query/i);
    }

    // ==================== Actions ====================

    /**
     * Opens the Sessions page at '/sessions'.
     */
    async open(): Promise<void> {
        await this.page.goto(AppRoutes.SESSIONS, {
            waitUntil: 'domcontentloaded',
        });
    }

    getAgentFilterButton(agentName: string): Locator {
        return this.page.getByRole('button', { name: new RegExp(agentName, 'i') });
    }

    async filterByAgent(agentName: string): Promise<void> {
        await this.getAgentFilterButton(agentName).click();
    }

    async search(query: string): Promise<void> {
        await this.searchInput.fill(query);
    }

    getSessionRow(identifier: string): Locator {
        return this.sessionRows.filter({ hasText: identifier });
    }

    async clickSession(identifier: string): Promise<void> {
        await this.getSessionRow(identifier).first().click();
    }
}

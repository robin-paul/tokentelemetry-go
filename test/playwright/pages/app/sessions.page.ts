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
        return this.page.getByPlaceholder('Search sessions or projects...');
    }

    get allAgentsFilterButton(): Locator {
        return this.page.getByRole('button', { name: 'All Agents' });
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
        await this.searchInput.press('Enter');
    }

    getSessionRow(identifier: string): Locator {
        return this.sessionRows.filter({ hasText: identifier });
    }

    async clickSession(identifier: string): Promise<void> {
        await this.getSessionRow(identifier).first().click();
    }
}

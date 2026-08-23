import { Locator, Page } from '@playwright/test';
import { NavbarComponent } from '../components/navbar.component';
import { AppRoutes } from '../../enums/app/app';

/**
 * Page Object for the Hermes Autonomous Agent view (/hermes).
 */
export class HermesPage {
    readonly nav: NavbarComponent;

    constructor(readonly page: Page) {
        this.nav = new NavbarComponent(page);
    }

    // ==================== Locators ====================

    get pageTitle(): Locator {
        return this.page.getByRole('heading', { level: 1 });
    }

    get gatewayStatusIndicator(): Locator {
        return this.page.getByText(/Gateway:\s*Active/i);
    }

    get kanbanColumns(): Locator {
        return this.page.locator('div.grid-cols-1.md\\:grid-cols-3 > div');
    }

    // ==================== Actions ====================

    /**
     * Opens the Hermes page at '/hermes'.
     */
    async open(): Promise<void> {
        await this.page.goto(AppRoutes.HERMES, {
            waitUntil: 'domcontentloaded',
        });
    }

    getColumn(columnTitle: 'To Do' | 'In Progress' | 'Done' | string): Locator {
        return this.kanbanColumns.filter({
            has: this.page.getByRole('heading', { name: new RegExp(columnTitle, 'i') }),
        });
    }

    getTaskCards(columnTitle: string): Locator {
        return this.getColumn(columnTitle).locator('div.rounded-lg');
    }
}

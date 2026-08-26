import { Locator, Page } from '@playwright/test';
import { NavbarComponent } from '../components/navbar.component';
import { AppRoutes } from '../../enums/app/app';

/**
 * Page Object for the Projects catalog and Project Detail views.
 */
export class ProjectsPage {
    readonly nav: NavbarComponent;

    constructor(readonly page: Page) {
        this.nav = new NavbarComponent(page);
    }

    // ==================== Locators ====================

    get pageTitle(): Locator {
        return this.page.getByRole('heading', { name: /Project Workspaces|Projects/i });
    }

    get searchInput(): Locator {
        return this.page.getByTestId('project-search-input');
    }

    get gridToggle(): Locator {
        return this.page.getByTestId('view-mode-grid');
    }

    get tableToggle(): Locator {
        return this.page.getByTestId('view-mode-table');
    }

    get projectsGrid(): Locator {
        return this.page.getByTestId('projects-grid');
    }

    get projectsTable(): Locator {
        return this.page.getByTestId('projects-table');
    }

    get sortSessionsBtn(): Locator {
        return this.page.getByTestId('sort-sessions');
    }

    get sortTokensBtn(): Locator {
        return this.page.getByTestId('sort-tokens');
    }

    get sortCostBtn(): Locator {
        return this.page.getByTestId('sort-cost');
    }

    get sortNameBtn(): Locator {
        return this.page.getByTestId('sort-name');
    }

    getProjectCard(name: string): Locator {
        return this.page.getByTestId(`project-card-${name}`);
    }

    getProjectRow(name: string): Locator {
        return this.page.getByTestId(`project-row-${name}`);
    }

    getWorktreeToggle(name: string): Locator {
        return this.page.getByTestId(`toggle-worktrees-${name}`);
    }

    // ==================== Project Detail Locators ====================

    get worktreeStrip(): Locator {
        return this.page.getByTestId('worktree-strip');
    }

    get activityTab(): Locator {
        return this.page.getByTestId('tab-activity');
    }

    get plansTab(): Locator {
        return this.page.getByTestId('tab-plans');
    }

    get configTab(): Locator {
        return this.page.getByTestId('tab-config');
    }

    get activityContent(): Locator {
        return this.page.getByTestId('tab-content-activity');
    }

    get plansContent(): Locator {
        return this.page.getByTestId('tab-content-plans');
    }

    get configContent(): Locator {
        return this.page.getByTestId('tab-content-config');
    }

    get toggleHideBtn(): Locator {
        return this.page.getByTestId('toggle-hide-btn');
    }

    get aliasInput(): Locator {
        return this.page.getByTestId('alias-input');
    }

    get saveAliasBtn(): Locator {
        return this.page.getByTestId('save-alias-btn');
    }

    // ==================== Actions ====================

    async open(): Promise<void> {
        await this.page.goto(AppRoutes.PROJECTS);
    }

    async openProject(path: string): Promise<void> {
        await this.page.goto(`${AppRoutes.PROJECTS}/${encodeURIComponent(path)}`);
    }

    async search(query: string): Promise<void> {
        await this.searchInput.fill(query);
    }

    async switchToGrid(): Promise<void> {
        await this.gridToggle.click();
    }

    async switchToTable(): Promise<void> {
        await this.tableToggle.click();
    }
}

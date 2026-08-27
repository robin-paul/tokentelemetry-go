import { Locator, Page } from '@playwright/test';
import { NavbarComponent } from '../components/navbar.component';
import { AppRoutes } from '../../enums/app/app';

/**
 * Page Object for the Settings & Pricing view (/settings).
 */
export class SettingsPage {
    readonly nav: NavbarComponent;

    constructor(readonly page: Page) {
        this.nav = new NavbarComponent(page);
    }

    // ==================== Locators ====================

    get pageTitle(): Locator {
        return this.page.getByRole('heading', { level: 1 });
    }

    get modelPatternInput(): Locator {
        return this.page.getByPlaceholder(/Model Pattern/i);
    }

    get inputCostInput(): Locator {
        return this.page.getByPlaceholder(/Input \$\/1M/i);
    }

    get outputCostInput(): Locator {
        return this.page.getByPlaceholder(/Output \$\/1M/i);
    }

    get addOverrideButton(): Locator {
        return this.page.getByRole('button', { name: /Add Rate Override/i });
    }

    get updateOverrideButton(): Locator {
        return this.page.getByRole('button', { name: /Update Override/i });
    }

    get statusMessage(): Locator {
        return this.page.getByText(/(Custom pricing override saved|Pricing override for .* updated|Pricing override for .* removed)/i);
    }

    get overridesTable(): Locator {
        return this.page.getByRole('table');
    }

    get overrideRows(): Locator {
        return this.page.getByRole('row');
    }

    get emptyOverridesNotice(): Locator {
        return this.page.getByText(/No custom overrides configured/i);
    }

    get searchFilterInput(): Locator {
        return this.page.getByPlaceholder(/Filter overrides/i);
    }

    // ==================== Actions ====================

    /**
     * Opens the Settings page at '/settings'.
     */
    async open(): Promise<void> {
        await this.page.goto(AppRoutes.SETTINGS, {
            waitUntil: 'domcontentloaded',
        });
    }

    getOverrideRow(modelPattern: string): Locator {
        return this.overrideRows.filter({ hasText: modelPattern });
    }

    async addPricingOverride(
        modelPattern: string,
        inputCost: number,
        outputCost: number
    ): Promise<void> {
        await this.modelPatternInput.fill(modelPattern);
        await this.inputCostInput.fill(inputCost.toString());
        await this.outputCostInput.fill(outputCost.toString());
        await this.addOverrideButton.click();
    }

    async editPricingOverride(
        modelPattern: string,
        newInputCost: number,
        newOutputCost: number
    ): Promise<void> {
        const row = this.getOverrideRow(modelPattern);
        await row.locator('button[title="Edit override"]').click();
        await this.inputCostInput.fill(newInputCost.toString());
        await this.outputCostInput.fill(newOutputCost.toString());
        await this.updateOverrideButton.click();
    }

    async deleteOverride(modelPattern: string): Promise<void> {
        const row = this.getOverrideRow(modelPattern);
        await row.locator('button[title="Delete override"]').or(row.locator('button').last()).click();
    }
}

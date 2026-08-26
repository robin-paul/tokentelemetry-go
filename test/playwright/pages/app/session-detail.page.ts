import { Locator, Page } from '@playwright/test';
import { NavbarComponent } from '../components/navbar.component';
import { AppRoutes } from '../../enums/app/app';

/**
 * Page Object for the Session Detail / Inspector view (/sessions/:id).
 */
export class SessionDetailPage {
    readonly nav: NavbarComponent;

    constructor(readonly page: Page) {
        this.nav = new NavbarComponent(page);
    }

    // ==================== Locators ====================

    get backToSessionsLink(): Locator {
        return this.page.getByRole('link', { name: /Back to Sessions/i });
    }

    get sessionIdHeading(): Locator {
        return this.page.getByRole('heading', { level: 1 });
    }

    get agentBadge(): Locator {
        return this.page.locator('span.font-semibold').first();
    }

    get projectNameLabel(): Locator {
        return this.page.getByText(/Project:/i);
    }

    get netCostValue(): Locator {
        return this.page.locator('div.text-emerald-400.tabular').first();
    }

    get tokensValue(): Locator {
        return this.page.locator('div.text-blue-400.tabular').first();
    }

    get modelValue(): Locator {
        return this.page
            .locator('div')
            .filter({ has: this.page.getByText('Model', { exact: true }) })
            .locator('.font-mono');
    }

    get stepScrubber(): Locator {
        return this.page.locator('input[type="range"]');
    }

    get scrubberStepLabel(): Locator {
        return this.page.getByTestId('scrubber-step-indicator');
    }

    get turnCards(): Locator {
        return this.page.locator('.rounded-xl').filter({ hasText: /Turn #\d+/ });
    }

    get userTurnCards(): Locator {
        return this.page.locator('.rounded-xl').filter({ hasText: /User Prompt/i });
    }

    get assistantTurnCards(): Locator {
        return this.page.locator('.rounded-xl').filter({ hasText: /Response/i });
    }

    get reasoningCards(): Locator {
        return this.page.locator('.rounded-xl').filter({ hasText: /Reasoning & Thoughts/i });
    }

    get rawMdToggleButtons(): Locator {
        return this.page.getByRole('button', { name: /View Raw|View MD/i });
    }

    get copyCodeButtons(): Locator {
        return this.page.getByRole('button', { name: /Copy Code/i });
    }

    get subagentRunsSection(): Locator {
        return this.page.getByText(/Spawned Subagents/i);
    }

    get playPauseButton(): Locator {
        return this.page.getByRole('button', { name: /Start playback|Pause playback/i });
    }

    get prevStepButton(): Locator {
        return this.page.getByRole('button', { name: /Previous turn/i });
    }

    get nextStepButton(): Locator {
        return this.page.getByRole('button', { name: /Next turn/i });
    }

    get stepIndexButtons(): Locator {
        return this.page.locator('button').filter({ hasText: /^#\d+/ });
    }

    get searchInput(): Locator {
        return this.page.getByLabel('Search within active trace');
    }

    get waterfall(): Locator {
        return this.page.getByTestId('execution-waterfall');
    }

    get waterfallToolRows(): Locator {
        return this.page.getByTestId('execution-waterfall').locator('.cursor-pointer');
    }

    get inspectorSidebar(): Locator {
        return this.page.getByTestId('inspector-sidebar');
    }

    get contextPanel(): Locator {
        return this.page.getByTestId('context-panel');
    }

    get toolsPanel(): Locator {
        return this.page.getByTestId('tools-panel');
    }

    get artifactsPanel(): Locator {
        return this.page.getByTestId('artifacts-panel');
    }

    get rawPanel(): Locator {
        return this.page.getByTestId('raw-panel');
    }

    get tabContext(): Locator {
        return this.page.getByTestId('inspector-tab-context');
    }

    get tabTools(): Locator {
        return this.page.getByTestId('inspector-tab-tools');
    }

    get tabArtifacts(): Locator {
        return this.page.getByTestId('inspector-tab-artifacts');
    }

    get tabRaw(): Locator {
        return this.page.getByTestId('inspector-tab-raw');
    }

    get toolInvocationCards(): Locator {
        return this.page.getByTestId('tool-invocation-card');
    }

    get artifactLightboxModal(): Locator {
        return this.page.getByTestId('artifact-lightbox-modal');
    }

    get copyRawJsonButton(): Locator {
        return this.page.getByTestId('copy-raw-json-button');
    }

    get toggleSidebarButton(): Locator {
        return this.page.getByTestId('toggle-sidebar-button');
    }

    // ==================== Actions ====================

    /**
     * Opens a specific Session Detail page at '/sessions/:id'.
     */
    async open(sessionId: string): Promise<void> {
        await this.page.goto(`${AppRoutes.SESSIONS}/${sessionId}`, {
            waitUntil: 'domcontentloaded',
        });
    }

    getTurnCard(turnIndex: number): Locator {
        return this.turnCards.filter({ hasText: `Turn #${turnIndex}` });
    }

    getToolsInTurn(turnIndex: number): Locator {
        return this.getTurnCard(turnIndex).locator('span').filter({ hasText: /\w+/ });
    }

    async scrubToStep(step: number): Promise<void> {
        await this.stepScrubber.fill(step.toString());
        await this.stepScrubber.dispatchEvent('input');
        await this.stepScrubber.dispatchEvent('change');
    }
}

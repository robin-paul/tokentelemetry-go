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
        return this.page.locator('[data-testid="session-id-heading"], [data-test="session-id-heading"], h1').first();
    }

    get agentBadge(): Locator {
        return this.page.locator('[data-testid="agent-badge"], [data-test="agent-badge"], span.font-semibold').first();
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
        return this.page.locator('[data-testid="execution-waterfall"], [data-test="execution-waterfall"]');
    }

    get waterfallToolRows(): Locator {
        return this.waterfall.locator('.cursor-pointer');
    }

    get inspectorSidebar(): Locator {
        return this.page.locator('[data-testid="inspector-sidebar"], [data-test="inspector-sidebar"]');
    }

    get contextPanel(): Locator {
        return this.page.locator('[data-testid="context-panel"], [data-test="context-panel"]');
    }

    get toolsPanel(): Locator {
        return this.page.locator('[data-testid="tools-panel"], [data-test="tools-panel"]');
    }

    get artifactsPanel(): Locator {
        return this.page.locator('[data-testid="artifacts-panel"], [data-test="artifacts-panel"]');
    }

    get rawPanel(): Locator {
        return this.page.locator('[data-testid="raw-panel"], [data-test="raw-panel"]');
    }

    get tabContext(): Locator {
        return this.page.locator('[data-testid="inspector-tab-context"], [data-test="inspector-tab-context"]');
    }

    get tabTools(): Locator {
        return this.page.locator('[data-testid="inspector-tab-tools"], [data-test="inspector-tab-tools"]');
    }

    get tabArtifacts(): Locator {
        return this.page.locator('[data-testid="inspector-tab-artifacts"], [data-test="inspector-tab-artifacts"]');
    }

    get tabRaw(): Locator {
        return this.page.locator('[data-testid="inspector-tab-raw"], [data-test="inspector-tab-raw"]');
    }

    get toolInvocationCards(): Locator {
        return this.page.locator('[data-testid="tool-invocation-card"], [data-test="tool-invocation-card"]');
    }

    get artifactLightboxModal(): Locator {
        return this.page.locator('[data-testid="artifact-lightbox-modal"], [data-test="artifact-lightbox-modal"]');
    }

    get copyRawJsonButton(): Locator {
        return this.page.locator('[data-testid="copy-raw-json-button"], [data-test="copy-raw-json-button"]');
    }

    get toggleSidebarButton(): Locator {
        return this.page.locator('[data-testid="toggle-sidebar-button"], [data-test="toggle-sidebar-button"]');
    }

    get toggleSplitViewButton(): Locator {
        return this.page.locator('[data-testid="toggle-split-view-button"], [data-test="toggle-split-view-button"]');
    }

    get toggleStaggerViewButton(): Locator {
        return this.page.locator('[data-testid="toggle-stagger-view-button"], [data-test="toggle-stagger-view-button"]');
    }

    get splitViewContainer(): Locator {
        return this.page.locator('[data-testid="split-view-container"], [data-test="split-view-container"]');
    }

    get staggeredViewContainer(): Locator {
        return this.page.locator('[data-testid="staggered-view-container"], [data-test="staggered-view-container"]');
    }

    get staggeredBrainRows(): Locator {
        return this.page.locator('[data-testid="staggered-brain-row"], [data-test="staggered-brain-row"]');
    }

    get staggeredDialogueRows(): Locator {
        return this.page.locator('[data-testid="staggered-dialogue-row"], [data-test="staggered-dialogue-row"]');
    }

    get dialogueColumn(): Locator {
        return this.page.locator('[data-testid="dialogue-column"], [data-test="dialogue-column"]');
    }

    get brainColumn(): Locator {
        return this.page.locator('[data-testid="brain-column"], [data-test="brain-column"]');
    }

    get dialogueColumnHeader(): Locator {
        return this.page.locator('[data-testid="dialogue-column-header"], [data-test="dialogue-column-header"]');
    }

    get brainColumnHeader(): Locator {
        return this.page.locator('[data-testid="brain-column-header"], [data-test="brain-column-header"]');
    }

    get unifiedColumn(): Locator {
        return this.page.locator('[data-testid="unified-column"], [data-test="unified-column"]');
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

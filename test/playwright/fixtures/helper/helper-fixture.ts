import { test as base } from '@playwright/test';

/**
 * Helper fixtures for test lifecycle management.
 */
export type HelperFixtures = Record<string, never>;

export const test = base.extend<HelperFixtures>({});

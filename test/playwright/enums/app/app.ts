/**
 * Application-specific constants for TokenTelemetry.
 */

/** Common UI messages & titles */
export enum Messages {
    APP_TITLE = 'TokenTelemetry',
    LIVE_TELEMETRY = 'Live Telemetry',
    CONNECTING = 'Connecting...',
    NO_SESSIONS = 'No sessions found',
    ACTIVE_AGENTS_DETECTED = 'Active Agents Detected',
}

/** UI route paths */
export enum AppRoutes {
    OVERVIEW = '/',
    SESSIONS = '/sessions',
    PROJECTS = '/projects',
    ANALYTICS = '/analytics',
    SETTINGS = '/settings',
}

/** API endpoint paths */
export enum ApiEndpoints {
    HEALTH = '/healthz',
    SESSIONS = '/api/sessions',
    RECENT = '/api/recent',
    STATS = '/api/stats',
    DAILY_STATS = '/api/stats/daily',
    LEADERBOARD = '/api/leaderboard',
    PROJECTS = '/api/projects',
    PRICING = '/api/pricing',
    PRICING_OVERRIDE = '/api/pricing/override',
    EVENTS = '/events',
}

/** Storage state file paths */
export enum StorageStatePaths {
    APP = '.auth/app/appStorageState.json',
}

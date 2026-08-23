import path from 'path';
import os from 'os';

export const appConfig = {
    /** Frontend application URL */
    appUrl: process.env.APP_URL || 'http://127.0.0.1:8000',
    /** Backend API URL */
    apiUrl: process.env.API_URL || process.env.APP_URL || 'http://127.0.0.1:8000',
    /** Ephemeral scan directory monitored by the Go server */
    scanDir:
        process.env.E2E_SCAN_DIR ||
        path.join(os.tmpdir(), 'tokentelemetry-e2e-run', 'logs'),
    /** Ephemeral DB file path */
    dbPath:
        process.env.E2E_DB_PATH ||
        path.join(os.tmpdir(), 'tokentelemetry-e2e-run', 'test.db'),
};

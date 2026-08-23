import { test as base } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { appConfig } from '../../config/app';
import {
    CursorTurnOptions,
    createCursorTranscript,
} from '../../helpers/factories/cursor.factory';
import {
    OpenCodeTurnOptions,
    createOpenCodeTranscript,
} from '../../helpers/factories/opencode.factory';
import {
    CopilotSessionOptions,
    createCopilotChatDocument,
} from '../../helpers/factories/copilot.factory';

export interface TranscriptFixture {
    readonly scanDir: string;
    writeCursorSession(
        projectName: string,
        sessionId: string,
        turns?: CursorTurnOptions[]
    ): Promise<string>;
    writeOpenCodeSession(
        projectName: string,
        sessionId: string,
        turns?: OpenCodeTurnOptions[]
    ): Promise<string>;
    writeCopilotSession(
        projectName: string,
        sessionId: string,
        options?: CopilotSessionOptions
    ): Promise<string>;
    writeRawTranscript(
        relativePath: string,
        content: string
    ): Promise<string>;
    appendTranscript(
        relativePath: string,
        additionalContent: string
    ): Promise<void>;
    cleanup(): Promise<void>;
}

export class FileTranscriptManager implements TranscriptFixture {
    private createdFiles: string[] = [];

    get scanDir(): string {
        const dir = appConfig.scanDir || process.env.E2E_SCAN_DIR;
        if (!dir) {
            throw new Error('E2E_SCAN_DIR is not configured');
        }
        return dir;
    }

    async writeRawTranscript(
        relativePath: string,
        content: string
    ): Promise<string> {
        const fullPath = path.join(this.scanDir, relativePath);
        fs.mkdirSync(path.dirname(fullPath), { recursive: true });
        fs.writeFileSync(fullPath, content, 'utf-8');
        this.createdFiles.push(fullPath);
        return fullPath;
    }

    async appendTranscript(
        relativePath: string,
        additionalContent: string
    ): Promise<void> {
        const fullPath = path.join(this.scanDir, relativePath);
        fs.appendFileSync(fullPath, additionalContent, 'utf-8');
    }

    async writeCursorSession(
        projectName: string,
        sessionId: string,
        turns?: CursorTurnOptions[]
    ): Promise<string> {
        const relPath = path.join('.cursor', 'projects', projectName, `${sessionId}.jsonl`);
        const content = createCursorTranscript(
            turns ?? [
                {
                    role: 'assistant',
                    model: 'claude-3-5-sonnet',
                    inputTokens: 1200,
                    outputTokens: 350,
                    cacheReadTokens: 400,
                    tools: ['read_file'],
                },
            ]
        );
        return this.writeRawTranscript(relPath, content);
    }

    async writeOpenCodeSession(
        projectName: string,
        sessionId: string,
        turns?: OpenCodeTurnOptions[]
    ): Promise<string> {
        const relPath = path.join('opencode', projectName, `${sessionId}.jsonl`);
        const content = createOpenCodeTranscript(
            turns ?? [
                {
                    role: 'assistant',
                    model: 'claude-3-7-sonnet',
                    inputTokens: 2500,
                    outputTokens: 600,
                    cacheReadTokens: 800,
                    cacheWriteTokens: 150,
                    toolName: 'ast_grep_search',
                },
            ]
        );
        return this.writeRawTranscript(relPath, content);
    }

    async writeCopilotSession(
        projectName: string,
        sessionId: string,
        options?: CopilotSessionOptions
    ): Promise<string> {
        const relPath = path.join('copilot', projectName, `${sessionId}.json`);
        const content = createCopilotChatDocument(options);
        return this.writeRawTranscript(relPath, content);
    }

    async cleanup(): Promise<void> {
        for (const file of this.createdFiles) {
            try {
                if (fs.existsSync(file)) {
                    fs.unlinkSync(file);
                }
            } catch {
                // Ignore cleanup errors
            }
        }
        this.createdFiles = [];
    }
}

export const test = base.extend<{
    transcriptFixture: TranscriptFixture;
}>({
    transcriptFixture: async ({}, use) => {
        const manager = new FileTranscriptManager();
        try {
            await use(manager);
        } finally {
            await manager.cleanup();
        }
    },
});

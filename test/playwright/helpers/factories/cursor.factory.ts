export interface CursorTurnOptions {
    role?: 'user' | 'assistant';
    model?: string;
    inputTokens?: number;
    outputTokens?: number;
    cacheReadTokens?: number;
    cacheCreationTokens?: number;
    tools?: string[];
    timestamp?: string;
}

export function createCursorTurnLine(options: CursorTurnOptions = {}): string {
    const role = options.role ?? 'assistant';
    const model = options.model ?? 'claude-3-5-sonnet';
    const timestamp = options.timestamp ?? new Date().toISOString();
    const inputTokens = options.inputTokens ?? 150;
    const outputTokens = options.outputTokens ?? 50;
    const cacheReadTokens = options.cacheReadTokens ?? 0;
    const cacheCreationTokens = options.cacheCreationTokens ?? 0;
    const tools = options.tools ?? [];

    const obj = {
        role,
        timestamp,
        message: {
            model,
            usage: {
                input_tokens: inputTokens,
                output_tokens: outputTokens,
                cache_read_input_tokens: cacheReadTokens,
                cache_creation_input_tokens: cacheCreationTokens,
            },
            content: tools.map((name) => ({
                type: 'tool_call',
                name,
            })),
        },
    };

    return JSON.stringify(obj);
}

export function createCursorTranscript(turns: CursorTurnOptions[]): string {
    return turns.map((t) => createCursorTurnLine(t)).join('\n') + '\n';
}

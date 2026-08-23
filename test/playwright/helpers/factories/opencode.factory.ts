export interface OpenCodeTurnOptions {
    role?: 'assistant' | 'user';
    model?: string;
    inputTokens?: number;
    outputTokens?: number;
    cacheReadTokens?: number;
    cacheWriteTokens?: number;
    timestamp?: number;
    toolName?: string;
}

export function createOpenCodeTurnLine(options: OpenCodeTurnOptions = {}): string {
    const role = options.role ?? 'assistant';
    const model = options.model ?? 'claude-3-7-sonnet';
    const timestamp = options.timestamp ?? Date.now();
    const input = options.inputTokens ?? 200;
    const output = options.outputTokens ?? 80;
    const read = options.cacheReadTokens ?? 0;
    const write = options.cacheWriteTokens ?? 0;

    const obj = {
        type: 'turn',
        role,
        model,
        timestamp,
        tool: options.toolName,
        name: options.toolName,
        tokens: {
            input,
            output,
            cache: {
                read,
                write,
            },
        },
        data: options.toolName
            ? {
                  type: 'tool_call',
                  name: options.toolName,
                  tool: options.toolName,
                  tokens: {
                      input,
                      output,
                      cache: {
                          read,
                          write,
                      },
                  },
              }
            : undefined,
    };

    return JSON.stringify(obj);
}

export function createOpenCodeTranscript(turns: OpenCodeTurnOptions[]): string {
    return turns.map((t) => createOpenCodeTurnLine(t)).join('\n') + '\n';
}

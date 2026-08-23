export interface CopilotRequestOptions {
    modelId?: string;
    completionTokens?: number;
    userPromptText?: string;
    timestamp?: number;
}

export interface CopilotSessionOptions {
    version?: number;
    creationDate?: number;
    requests?: CopilotRequestOptions[];
}

export function createCopilotChatDocument(options: CopilotSessionOptions = {}): string {
    const version = options.version ?? 1;
    const creationDate = options.creationDate ?? Date.now();
    const requests = options.requests ?? [
        {
            modelId: 'gpt-4o',
            completionTokens: 250,
            userPromptText: 'Help me optimize this Go SQLite transaction loop.',
            timestamp: creationDate,
        },
    ];

    const doc = {
        version,
        creationDate,
        requests: requests.map((r) => ({
            modelId: r.modelId ?? 'gpt-4o',
            timestamp: r.timestamp ?? Date.now(),
            completionTokens: r.completionTokens ?? 100,
            message: {
                text: r.userPromptText ?? 'Hello Copilot',
            },
        })),
    };

    return JSON.stringify(doc, null, 2);
}

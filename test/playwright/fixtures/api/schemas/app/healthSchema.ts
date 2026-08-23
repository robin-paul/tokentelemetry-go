import { z } from 'zod';

export const healthResponseSchema = z.strictObject({
    status: z.string(),
    version: z.string(),
});

export type HealthResponse = z.infer<typeof healthResponseSchema>;

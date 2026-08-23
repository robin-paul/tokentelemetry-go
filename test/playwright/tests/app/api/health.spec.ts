import { expect, test } from '../../../fixtures/pom/test-options';
import { ApiEndpoints } from '../../../enums/app/app';
import { healthResponseSchema } from '../../../fixtures/api/schemas/app/healthSchema';

test.describe('health api', () => {
    test(
        'should return 200 OK and valid schema on /healthz',
        { tag: '@api' },
        async ({ apiRequest }) => {
            const response = await apiRequest({
                method: 'GET',
                url: ApiEndpoints.HEALTH,
                baseUrl: process.env.API_URL,
            });

            expect(response.status).toBe(200);
            const data = healthResponseSchema.parse(response.body);
            expect(data.status).toBe('ok');
            expect(data.version).toBeDefined();
        }
    );
});

import { expect, test } from '../../../fixtures/pom/test-options';
import { ApiEndpoints } from '../../../enums/app/app';
import { healthResponseSchema } from '../../../fixtures/api/schemas/app/healthSchema';
import { appConfig } from '../../../config/app';

test.describe('health api', () => {
    test(
        'should return 200 OK and valid schema on /healthz',
        { tag: '@api' },
        async ({ apiRequest }) => {
            const response = await apiRequest({
                method: 'GET',
                url: ApiEndpoints.HEALTH,
                baseUrl: appConfig.apiUrl,
            });

            expect(response.status).toBe(200);
            expect(healthResponseSchema.parse(response.body)).toBeTruthy();
        }
    );
});

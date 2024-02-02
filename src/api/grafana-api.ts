import { getBackendSrv } from '@grafana/runtime';

import { lastValueFrom } from 'rxjs';

export interface ApiKey {
  id: number;
  name: string;
  key: string;
  role: string;
}

interface CreateApiKeyResult {
  name: string;
  key: string;
}

export const createApiKey = async (
  name: string,
  role: string,
  secondsToLive = 0
): Promise<CreateApiKeyResult | undefined> => {
  const response = await lastValueFrom(
    getBackendSrv().fetch({
      url: `/api/auth/keys`,
      method: 'POST',
      data: { name, role, secondsToLive },
    })
  );

  return response.data as CreateApiKeyResult | undefined;
};

export const deleteApiKey = async (id: number): Promise<void> => {
  await lastValueFrom(
    getBackendSrv().fetch({
      url: `/api/auth/keys/${id}`,
      method: `DELETE`,
    })
  );
};

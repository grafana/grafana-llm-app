import { BackendSrvRequest, FetchResponse, getBackendSrv } from '@grafana/runtime';

import { lastValueFrom } from 'rxjs';

import pluginJson from '../plugin.json';

export const getAdminApiUrl = (path: string): string => {
  const url = `/api/plugins/${pluginJson.id}/resources/${path}`;

  return url.replace(/([^:]\/)\/+/g, '$1');
};

const apiRequest = async (url: string, options?: Omit<BackendSrvRequest, 'url'>): Promise<FetchResponse> => {
  const fetch = getBackendSrv().fetch({
    ...options,
    url,
  });
  return lastValueFrom(fetch);
};

const apiPost = (url: string, options?: Omit<BackendSrvRequest, 'url'>): Promise<FetchResponse> =>
  apiRequest(url, {
    ...options,
    method: 'POST',
  });

const apiPut = (url: string, options?: Omit<BackendSrvRequest, 'url'>): Promise<FetchResponse> =>
  apiRequest(url, {
    ...options,
    method: 'PUT',
  });

const apiGet = (url: string, options?: Omit<BackendSrvRequest, 'url'>): Promise<FetchResponse> =>
  apiRequest(url, {
    ...options,
    method: 'GET',
  });

const apiDelete = (url: string, options?: Omit<BackendSrvRequest, 'url'>): Promise<FetchResponse> =>
  apiRequest(url, {
    ...options,
    method: 'DELETE',
  });

export const adminApiGet = <T = any>(
  url: string,
  options?: Omit<BackendSrvRequest, 'url'>
): Promise<FetchResponse<T>> => apiGet(getAdminApiUrl(url), options);

export const adminApiPost = <T = any>(
  url: string,
  options?: Omit<BackendSrvRequest, 'url'>
): Promise<FetchResponse<T>> => apiPost(getAdminApiUrl(url), options);

export const adminApiPut = <T = any>(
  url: string,
  options?: Omit<BackendSrvRequest, 'url'>
): Promise<FetchResponse<T>> => apiPut(getAdminApiUrl(url), options);

export const adminApiDelete = (url: string, options?: Omit<BackendSrvRequest, 'url'>): Promise<FetchResponse> =>
  apiDelete(getAdminApiUrl(url), options);

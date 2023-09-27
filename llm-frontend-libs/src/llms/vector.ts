/**
 * Vector search API.
 *
 * This module can be used to interact with the vector database configured
 * in the Grafana LLM app plugin. That plugin must be installed, enabled and configured
 * in order for these functions to work.
 *
 * The {@link enabled} function can be used to check if the plugin is enabled and configured.
 */

import { getBackendSrv, logDebug } from "@grafana/runtime";
import { LLM_PLUGIN_ROUTE, setLLMPluginVersion } from "./constants";
import { LLMAppHealthCheck } from "./types";

interface SearchResultPayload extends Record<string, any> { }

/**
 * A request to search for resources in the vector database.
 **/
export interface SearchRequest {
  /**
   * The name of the collection to search in.
   **/
  collection: string;

  /** The query to search for. */
  query: string;

  /**
   * Limit the number of results returned to the top `topK` results.
   * 
   * Defaults to 10.
   **/
  topK?: number;
}

/**
 * The results of a vector search.
 *
 * Results will be ordered by score, descending.
 */
export interface SearchResult<T extends SearchResultPayload> {
  /**
   * The payload of the result.
   *
   * The type of this payload depends on the collection that was searched in.
   * Grafana core types will be added to the same module as this type as they
   * are implemented.
   **/
  payload: T;

  /**
   * The score of the result.
   *
   * This is a number between 0 and 1, where 1 is the best possible match.
   */
  score: number;
}

interface SearchResultResponse<T extends SearchResultPayload> {
  results: Array<SearchResult<T>>;
}

/**
 * Search for resources in the configured vector database.
 */
export async function search<T extends SearchResultPayload>(request: SearchRequest): Promise<Array<SearchResult<T>>> {
  const response = await getBackendSrv().post<SearchResultResponse<T>>('/api/plugins/grafana-llm-app/resources/vector/search', request, {
    headers: { 'Content-Type': 'application/json' }
  });
  return response.results;
}

let loggedWarning = false;

/** Check if the vector API is enabled and configured via the LLM plugin. */
export const enabled = async () => {
  // Run a health check to see if the plugin is installed.
  let response: LLMAppHealthCheck;
  try {
    response = await getBackendSrv().get(`${LLM_PLUGIN_ROUTE}/health`, undefined, undefined, {
      showSuccessAlert: false, showErrorAlert: false,
    });
  } catch (e) {
    if (!loggedWarning) {
      logDebug(String(e));
      logDebug('Failed to check if vector service is enabled. This is expected if the Grafana LLM plugin is not installed, and the above error can be ignored.');
      loggedWarning = true;
    }
    return false;
  }
  const { details } = response;
  // Update the version if it's present on the response.
  if (details.version !== undefined) {
    setLLMPluginVersion(details.version);
  }
  // If the plugin is installed then check if it is configured.
  return details.vector ?? false;
};

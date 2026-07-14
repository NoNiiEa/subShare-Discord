import { BackendClient } from "../api/index.js";
import { config } from "../config.js";

/**
 * Shared backend client. Import this everywhere instead of constructing a new
 * BackendClient per handler.
 */
export const backend = new BackendClient({
  baseUrl: config.BACKEND_BASE_URL,
  apiKey: config.BACKEND_API_KEY,
});

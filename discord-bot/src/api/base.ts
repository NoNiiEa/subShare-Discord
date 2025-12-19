import { BackendConfig } from "./types.js";

export class BaseAPI {
  private baseUrl: string;
  private apiKey: string;

  constructor(config: BackendConfig) {
    this.baseUrl = config.baseUrl;
    this.apiKey = config.apiKey;
  }

  protected async request<T>(
    endpoint: string,
    method: "GET" | "POST" | "PUT" | "DELETE" = "GET",
    body?: unknown
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const headers: HeadersInit = {
      "Content-Type": "application/json",
    };

    if (this.apiKey) {
      headers["Authorization"] = `Bearer ${this.apiKey}`;
    }

    try {
      const response = await fetch(url, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });

      if (!response.ok) {
        const errorBody = await response.json().catch(() => null);
        const error: any = new Error(
          `API Error [${response.status}]: ${errorBody?.error || errorBody?.message || response.statusText}`
        );
        // Attach response data for better error handling
        error.response = {
          status: response.status,
          statusText: response.statusText,
          data: errorBody
        };
        throw error;
      }

      return (await response.json()) as T;
    } catch (error) {
        console.error(`[BackendClient] Request failed: ${url}`, error);
        throw error;
    }
  }
}
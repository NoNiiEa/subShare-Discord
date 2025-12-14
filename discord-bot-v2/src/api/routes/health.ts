import { BaseAPI } from "../base.js";
import { HealthResponse } from "../types.js";

export class Health extends BaseAPI {
    
    async ping(): Promise<HealthResponse> {
        return await this.request<HealthResponse>(`/health`)
    }
}
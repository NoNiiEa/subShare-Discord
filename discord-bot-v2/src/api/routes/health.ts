import { BaseAPI } from "../base";
import { HealthResponse } from "../types";

export class Health extends BaseAPI {
    
    async ping(): Promise<HealthResponse> {
        return await this.request<HealthResponse>(`/health`)
    }
}
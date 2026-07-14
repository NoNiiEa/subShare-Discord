import { BaseAPI } from "../base.js";
import { BillResponse, PayRequest, PayMultipleRequest, PayMultipleResponse } from "../types.js";

export class Bill extends BaseAPI {
    async GetUnpaidByGuild(guildId: string): Promise<BillResponse[]> {
        return await this.request<BillResponse[]>(`/guild/${guildId}/unpaid-bills`)
    }

    async GetUnpaidByUserAndGuild(userId: string, guildId: string): Promise<BillResponse[]> {
        return await this.request<BillResponse[]>(`/member/${userId}/guild/${guildId}/unpaid-bills`)
    }

    async Pay(payload: PayRequest): Promise<BillResponse> {
        return await this.request<BillResponse>(`/bill/pay`, "POST", payload)
    }

    async PayMultiple(payload: PayMultipleRequest): Promise<PayMultipleResponse> {
        return await this.request<PayMultipleResponse>(`/bill/pay-multiple`, "POST", payload)
    }
}
import { BaseAPI } from "../base.js";
import { BillResponse } from "../types.js";

export class Bill extends BaseAPI {
    async GetUnpaidByGuild(guildId: string): Promise<BillResponse[]> {
        return await this.request<BillResponse[]>(`/guild/${guildId}/bills`)
    }

    async GetUnpaidByUserAndGuild(userId: string, guildId: string): Promise<BillResponse[]> {
        return await this.request<BillResponse[]>(`/member/${userId}/guild/${guildId}/bills`)
    }
}
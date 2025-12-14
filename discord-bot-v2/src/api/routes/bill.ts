import { BaseAPI } from "../base.js";
import { BillResponse } from "../types.js";

export class Bill extends BaseAPI {
    async GetUnpaidByGuild(guildId: string): Promise<BillResponse[]> {
        return await this.request<BillResponse[]>(`/guild/${guildId}/bills`)
    }
}
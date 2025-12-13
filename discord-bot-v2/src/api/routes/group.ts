import { CreateGroupRequest, GroupResponse, InviteMemberRequest } from "../types";
import { BaseAPI } from "../base";

export class Group extends BaseAPI {

    async create(payload: CreateGroupRequest): Promise<GroupResponse> {   
        return await this.request<GroupResponse>(`/groups`, "POST", payload);
    }

    async view(userId: string, guildId: string): Promise<GroupResponse[]> {
        return await this.request<GroupResponse[]>(`/member/${userId}/guild/${guildId}/groups`)
    }

    async invite(payload: InviteMemberRequest, groupId: number): Promise<GroupResponse> {
        return await this.request<GroupResponse>(`/groups/${groupId.toString()}/invite`, "POST", payload)
    }

    async viewOwn(userId: string, guildId: string): Promise<GroupResponse[]> {
        return await this.request<GroupResponse[]>(`/member/${userId}/guild/${guildId}/own-groups`)
    }
}
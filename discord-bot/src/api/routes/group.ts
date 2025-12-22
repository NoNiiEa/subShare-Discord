import { CreateGroupRequest, GroupResponse, InviteMemberRequest, AcceptInviteRequest, UpdateGroupRequest } from "../types.js";
import { BaseAPI } from "../base.js";

export class Group extends BaseAPI {

    async create(payload: CreateGroupRequest): Promise<GroupResponse> {   
        return await this.request<GroupResponse>(`/groups`, "POST", payload);
    }

    async get(groupId: number): Promise<GroupResponse> {
        return await this.request<GroupResponse>(`/groups/${groupId}`);
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

    async pendingInvite(userId: string, guildId: string): Promise<GroupResponse[]> {
        return await this.request<GroupResponse[]>(`/member/${userId}/guild/${guildId}/pending-invite`)
    }

    async acceptInvite(payload: AcceptInviteRequest, groupId: number): Promise<GroupResponse> {
        return await this.request<GroupResponse>(`/groups/${groupId}/accept-invite`, "POST", payload);
    }

    async DeclineInvite(payload: AcceptInviteRequest, groupId: number): Promise<GroupResponse> {
        return await this.request<GroupResponse>(`/groups/${groupId}/decline-invite`, "POST", payload);
    }

    async delete(groupId: number, userId: string): Promise<void> {
        return await this.request<void>(`/groups/${groupId}/member/${userId}`, "DELETE");
    }

    async update(payload: UpdateGroupRequest, groupId: number): Promise<void> {
        return await this.request<void>(`/groups/${groupId}`, "PATCH", payload)
    }
}
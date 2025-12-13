import axios, { AxiosInstance } from "axios";
import { 
    HealthResponse,
    CreateGroupRequest,
    GroupResponse,
    InviteGroupRequest
 } from "./schemas";

export class BackendClient {
    private baseURL: string;

    constructor(baseURL: string) {
        this.baseURL = baseURL;
    }

    async health(): Promise<HealthResponse> {
        const res = await axios.get(`${this.baseURL}/health`);
        return res.data;
    }

    async createGroup(payload: CreateGroupRequest): Promise<GroupResponse> {
        const res = await axios.post<GroupResponse>(
            `${this.baseURL}/groups`,
            payload
        );

        return res.data;
    }

    async inviteGroup(payload: InviteGroupRequest, groupID: number): Promise<GroupResponse> {
        const res = await axios.post<GroupResponse>(
            `${this.baseURL}/group/${groupID}/invite`,
            payload
        );

        return res.data
    }

    async viewGroups(memberId: string, guildId: string): Promise<GroupResponse[]> {
        const res = await axios.get<GroupResponse[]>(
            `${this.baseURL}/member/${memberId}/guild/${guildId}/groups`
        );

        return res.data
    }
}
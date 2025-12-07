export interface HealthResponse {
    status: string;
}

export interface CreateGroupRequest {
  name: string;
  amount: number;
  due_day: number;
  discord_guild_id: string;
  owner_discord_id: string;
  payment: {
    method: string;
    account: string;
  };
}

export interface GroupMember {
  member_id: string;
  dept: number;
  status: string;
  payment_status: string;
}

export interface GroupResponse {
  id: number;
  name: string;
  amount: number;
  amount_per_person: number;
  due_day: number;
  members: GroupMember[];
  discord_guild_id: string;
  owner_discord_id: string;
  payment: {
    method: string;
    account: string;
  };
  create_at: string;
}

export interface InviteGroupRequest {
    owner_id: string;
    member_ids: string[];
}


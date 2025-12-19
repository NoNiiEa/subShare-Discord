export interface HealthResponse {
    status: string;
}

export interface BackendConfig {
  baseUrl: string;
  apiKey: string;
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

export interface InviteMemberRequest {
    owner_id: string
    member_ids: string[]
}

export interface AcceptInviteRequest {
  user_id: string
}

export enum BillStatus {
    PENDING = "pending",
    SUBMITTED = "submitted",
    VERIFIED = "verified",
    REJECTED = "rejected"
}

export interface BillResponse {
    id: number;
    
    group_id: number;
    member_id: string;

    year: number;
    month: number;

    amount_due: number;
    currency: string;
    status: BillStatus | string; 
    description?: string; 

    proof?: Record<string, any> | null; 

    created_at: string;
    updated_at: string;
    submitted_at?: string | null;
    verified_at?: string | null;
    rejected_at?: string | null;
}

export interface PayRequest {
    user_id: string;
    guild_id: string;
    bill_id: number; // Backend expects int, not string
    proof_url: string;
}
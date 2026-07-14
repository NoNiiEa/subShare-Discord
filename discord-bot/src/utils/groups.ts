import { backend } from "./backend.js";
import { GroupResponse } from "../api/types.js";

/**
 * Fetches the given groups, keyed by id.
 *
 * Groups that fail to load are omitted rather than substituted with a
 * placeholder: without a group we cannot know who a bill is paid to, and a bill
 * with an unknown payee must not be batched with others.
 */
export async function fetchGroupMap(groupIds: number[]): Promise<Map<number, GroupResponse>> {
    const groups = new Map<number, GroupResponse>();

    await Promise.all(
        [...new Set(groupIds)].map(async (groupId) => {
            try {
                groups.set(groupId, await backend.group.get(groupId));
            } catch (err) {
                console.error(`Failed to fetch group ${groupId}:`, err);
            }
        })
    );

    return groups;
}

import { AutocompleteInteraction } from "discord.js";
import { GroupResponse } from "../api/types.js";

type GroupFetcher = (userId: string, guildId: string) => Promise<GroupResponse[]>;

/**
 * Shared autocomplete for group-selecting options. Fetches the user's groups
 * with the given fetcher, filters by the focused value (name or id), and
 * responds with up to 25 choices. Never throws — responds with [] on failure.
 */
export async function respondGroupAutocomplete(
  interaction: AutocompleteInteraction,
  fetchGroups: GroupFetcher
) {
  const guildId = interaction.guildId;
  if (!guildId) {
    await interaction.respond([]);
    return;
  }
  if (interaction.responded) return;

  try {
    const searchTerm = interaction.options.getFocused().toLowerCase();
    const groups = await fetchGroups(interaction.user.id, guildId);

    const choices = groups
      .filter(
        (group) =>
          group.name.toLowerCase().includes(searchTerm) ||
          String(group.id).includes(searchTerm)
      )
      .slice(0, 25)
      .map((group) => ({
        name: `${group.name} (ID: ${group.id})`,
        value: String(group.id),
      }));

    await interaction.respond(choices);
  } catch (err) {
    console.error("Group Autocomplete Error:", err);
    if (!interaction.responded) {
      await interaction.respond([]).catch(() => {});
    }
  }
}

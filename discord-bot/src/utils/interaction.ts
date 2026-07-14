import { ChatInputCommandInteraction, MessageFlags } from "discord.js";

/**
 * Ensures the command was used inside a guild. Returns the guild id, or replies
 * with a standard ephemeral message and returns null if used in DMs.
 *
 * Call this before deferring/replying.
 */
export async function requireGuild(
  interaction: ChatInputCommandInteraction
): Promise<string | null> {
  if (interaction.guildId) return interaction.guildId;

  await interaction.reply({
    content: "This command can only be used inside a server (not in DMs).",
    flags: MessageFlags.Ephemeral,
  });
  return null;
}

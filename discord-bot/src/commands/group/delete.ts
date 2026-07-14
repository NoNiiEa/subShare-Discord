import { ChatInputCommandInteraction, AutocompleteInteraction, MessageFlags, EmbedBuilder } from "discord.js";
import { backend } from "../../utils/backend.js";
import { requireGuild } from "../../utils/interaction.js";
import { toUserMessage, ErrorRule } from "../../utils/errors.js";
import { respondGroupAutocomplete } from "../../utils/groupAutocomplete.js";

const DELETE_ERROR_RULES: ErrorRule[] = [
    { match: ["not found"], status: 404, message: "❌ Group not found." },
    { status: 403, message: "❌ You don't have permission to delete this group." },
];

export function autocompleteDelete(interaction: AutocompleteInteraction) {
    // Only groups owned by the user can be deleted.
    return respondGroupAutocomplete(interaction, (userId, guildId) =>
        backend.group.viewOwn(userId, guildId)
    );
}

export async function executeDelete(interaction: ChatInputCommandInteraction) {
    const guildId = await requireGuild(interaction);
    if (!guildId) return;

    const userId = interaction.user.id;
    const groupIdStr = interaction.options.getString("group", true);

    const groupId = parseInt(groupIdStr, 10);
    if (isNaN(groupId)) {
        await interaction.reply({
            content: "❌ Invalid group ID. Please select a group from the dropdown.",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    await interaction.deferReply({
        flags: MessageFlags.Ephemeral,
    });

    try {
        // First, verify the group exists and user is the owner
        const group = await backend.group.get(groupId);

        if (!group) {
            await interaction.editReply({ content: "❌ Group not found." });
            return;
        }

        if (group.owner_discord_id !== userId) {
            await interaction.editReply({ content: "❌ You can only delete groups that you own." });
            return;
        }

        await backend.group.delete(groupId, userId);

        const embed = new EmbedBuilder()
            .setTitle("✅ Group Deleted Successfully")
            .setDescription(`**${group.name}** has been permanently deleted.`)
            .setColor(0xED4245) // Discord red
            .addFields(
                { name: "Group ID", value: `\`${group.id}\``, inline: true },
                { name: "Deleted by", value: `<@${userId}>`, inline: true }
            )
            .setFooter({ text: "All associated bills have also been deleted." });

        await interaction.editReply({ embeds: [embed] });
    } catch (err: any) {
        console.error("Delete Group Error:", err);
        await interaction.editReply({
            content: `❌ **Failed to delete group**\n> ${toUserMessage(err, DELETE_ERROR_RULES)}`,
        });
    }
}

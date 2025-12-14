import { ChatInputCommandInteraction, MessageFlags, EmbedBuilder, } from "discord.js";
import { BackendClient } from "../../api/index.js";
import { config } from "../../config.js";

export async function executeView(interaction: ChatInputCommandInteraction) {
    const userId = interaction.user.id
    const guildId = interaction.guildId

    if (!guildId) {
        await interaction.reply({
            content: "This command can only be used inside a server (not in DMs).",
            ephemeral: true,
        });
        return;
    }

    const backend = new BackendClient({
        baseUrl: config.BACKEND_BASE_URL || "http://localhost:8000",
    });

    await interaction.deferReply({
        flags: MessageFlags.Ephemeral,
    });

    try {
        const groups = await backend.group.view(userId, guildId);

        const embed = new EmbedBuilder()
            .setTitle(`📂 Active Groups (${groups.length})`)
            .setColor(0x5865F2)
            .setTimestamp();
        
        if (groups.length === 0) {
            embed.setDescription("No active groups found.");
        } else {
            const groupList = groups.map((group, index) => {
                return `**${index + 1}. ${group.name}**
                    💰 **${group.amount}** (Per person: ${group.amount_per_person})
                    📅 Due: **${group.due_day}** • 👑 <@${group.owner_discord_id}>
                    `;
                }).join("\n");

            embed.setDescription(groupList.length > 4096 ? groupList.substring(0, 4093) + "..." : groupList);
        }

        await interaction.editReply({ embeds: [embed] });
    } catch (err: any) {
        const detail =
            err?.response?.data?.error ||
            err?.response?.data?.message ||
            JSON.stringify(err?.response?.data ?? {}, null, 2);

        await interaction.editReply(
            `Failed to create group.\n\`\`\`json\n${detail}\n\`\`\``
        );
    }
}
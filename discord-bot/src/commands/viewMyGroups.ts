import {
    SlashCommandBuilder,
    ChatInputCommandInteraction,
    EmbedBuilder,
    MessageFlags,
} from "discord.js";
import { BackendClient } from "../backendClient/index.js";

export const data = new SlashCommandBuilder()
.setName("my-group")
.setDescription("View all of my group")

export async function execute(interaction: ChatInputCommandInteraction) {
    const guildId = interaction.guildId;
    if (!guildId) {
        await interaction.reply({
            content: "❌ This command can only be used inside a server (not in DMs).",
            ephemeral: true,
        });
        return;
    }

    const backendBaseUrl = process.env.BACKEND_BASE_URL;
    if (!backendBaseUrl) {
        await interaction.reply({
            content: "❌ BACKEND_BASE_URL is not configured on the bot.",
            ephemeral: true,
        });
        return;
    }

    const userId = interaction.user.id;
    const backend = new BackendClient(backendBaseUrl);

    await interaction.deferReply({
        flags: MessageFlags.Ephemeral,
    });

    try {
        const groups = await backend.viewGroups(userId, guildId);

        if (!groups || groups.length === 0) {
            await interaction.editReply({
                content: "You are not in any group.",
            });
            return;
        }

        const lines = groups.map((g: any) => {
            const amount = typeof g.amount === "number" ? g.amount.toFixed(2) : g.amount;
            const due = g.dueDay ?? g.due_day ?? "?";

            return `• **${g.name}** (ID: \`${g.id}\`)\n  💰 ${amount} / คน  •  📅 จ่ายทุกวันที่ ${due}`;
        });

        const embed = new EmbedBuilder()
            .setTitle("📋 Your groups in this server.")
            .setDescription(lines.join("\n\n"))
            .setFooter({ text: `${groups.length} group` })
            .setColor(0x00bcd4);

        await interaction.editReply({
            content: "",
            embeds: [embed],
        });
    } catch (err) {
        console.error("viewGroup error:", err);
        await interaction.editReply({
            content: "❌ error",
        });
    }
}
import {
    SlashCommandBuilder,
    ChatInputCommandInteraction,
    MessageFlags
} from "discord.js";
import { backend } from "../../utils/backend.js";

export default {
    data: new SlashCommandBuilder()
        .setName("health")
        .setDescription("Provides information about the health of the server"),

    async execute(interaction: ChatInputCommandInteraction) {
        await interaction.deferReply({ flags: MessageFlags.Ephemeral });

        try {
            const res = await backend.health.ping();
            await interaction.editReply(`✅ Backend status: ${res.status}`);
        } catch (err) {
            console.error("Health Check Error:", err);
            await interaction.editReply("❌ Backend is unreachable. Please try again later.");
        }
    },
};

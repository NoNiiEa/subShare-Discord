import {
    SlashCommandBuilder,
    ChatInputCommandInteraction,
    EmbedBuilder,
    MessageFlags
} from "discord.js";
import { BackendClient } from "../../api/index.js";
import { config } from "../../config.js";

export default {
    data: new SlashCommandBuilder()
        .setName("health")
        .setDescription("Provides information about the health of the server"),
        
    async execute(interaction: ChatInputCommandInteraction) {

        const backend = new BackendClient({
            baseUrl: config.BACKEND_BASE_URL || "http://localhost:8000",
        });

        const res = await backend.health.ping();

        await interaction.reply(
            res.status
        );
    },
};
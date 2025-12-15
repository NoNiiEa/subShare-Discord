import {
    SlashCommandBuilder,
    ChatInputCommandInteraction,
    EmbedBuilder,
    MessageFlags
} from "discord.js";

export default {
    data: new SlashCommandBuilder()
        .setName("user")
        .setDescription("Provides information about the user."),
        
    async execute(interaction: ChatInputCommandInteraction) {

        await interaction.reply(
            `This command was run by ${interaction.user.username}.`
        );
    
    },
};
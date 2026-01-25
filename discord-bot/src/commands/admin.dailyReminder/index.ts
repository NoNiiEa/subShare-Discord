import { SlashCommandBuilder, ChatInputCommandInteraction, PermissionFlagsBits } from "discord.js";
import { checkUnpaidBills } from "../../services/reminder.js";
import { config } from "../../config.js";

export default {
    data: new SlashCommandBuilder()
        .setName("force-check-bills")
        .setDescription("ADMIN ONLY: Manually triggers the daily bill check.")
        .setDefaultMemberPermissions(PermissionFlagsBits.Administrator),
    
    async execute(interaction: ChatInputCommandInteraction) {
        if (interaction.user.id !== config.BOT_OWNER_ID) {
            return interaction.reply({ content: "⛔ You are not the bot owner.", ephemeral: true });
        }

        await interaction.deferReply({ ephemeral: true });

        try {
            await interaction.editReply("⏳ Starting manual bill check...");
            
            const count = await checkUnpaidBills(interaction.client);
            
            await interaction.editReply(`✅ Manual check complete. Sent **${count}** reminders.`);
        } catch (error) {
            console.error(error);
            await interaction.editReply("❌ An error occurred while running the check.");
        }
    }
};
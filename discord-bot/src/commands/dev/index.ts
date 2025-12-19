import { ChatInputCommandInteraction, SlashCommandBuilder, PermissionFlagsBits } from "discord.js";
import { checkUnpaidBills } from "../../services/reminder.js"; // Import the function

export default {
    data: new SlashCommandBuilder()
        .setName("test-reminders")
        .setDescription(" [ADMIN] Manually trigger the daily bill check")
        .setDefaultMemberPermissions(PermissionFlagsBits.Administrator), // Only admins can run this

    async execute(interaction: ChatInputCommandInteraction) {
        await interaction.deferReply({ ephemeral: true });

        try {
            await interaction.editReply("⏳ **Starting manual bill check...**");

            // Manually trigger the logic
            const count = await checkUnpaidBills(interaction.client);

            await interaction.editReply(`✅ **Check Complete!**\n> Sent reminders to **${count}** users.`);
            
        } catch (error) {
            console.error(error);
            await interaction.editReply("❌ Something went wrong while testing.");
        }
    }
}
import { SlashCommandBuilder, ChatInputCommandInteraction, PermissionFlagsBits, MessageFlags } from "discord.js";
import { checkUnpaidBills } from "../../services/reminder.js";
import { config } from "../../config.js";

export default {
    data: new SlashCommandBuilder()
        .setName("force-check-bills")
        .setDescription("ADMIN ONLY: Manually triggers the weekly unpaid-bill reminder.")
        .setDefaultMemberPermissions(PermissionFlagsBits.Administrator),

    async execute(interaction: ChatInputCommandInteraction) {
        if (!config.BOT_OWNER_ID) {
            return interaction.reply({
                content: "⚠️ Bot owner is not configured (set BOT_OWNER_ID).",
                flags: MessageFlags.Ephemeral,
            });
        }

        if (interaction.user.id !== config.BOT_OWNER_ID) {
            return interaction.reply({
                content: "⛔ You are not the bot owner.",
                flags: MessageFlags.Ephemeral,
            });
        }

        await interaction.deferReply({ flags: MessageFlags.Ephemeral });

        try {
            await interaction.editReply("⏳ Starting manual bill check...");

            const count = await checkUnpaidBills(interaction.client);

            await interaction.editReply(`✅ Manual check complete. Tagged **${count}** member(s) with unpaid bills.`);
        } catch (error) {
            console.error("Force-check-bills Error:", error);
            await interaction.editReply("❌ An error occurred while running the check.");
        }
    }
};

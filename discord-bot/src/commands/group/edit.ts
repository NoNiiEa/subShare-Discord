import {
    ChatInputCommandInteraction,
    MessageFlags,
    EmbedBuilder,
    AutocompleteInteraction,
} from "discord.js";
import { backend } from "../../utils/backend.js";
import { toUserMessage } from "../../utils/errors.js";
import { respondGroupAutocomplete } from "../../utils/groupAutocomplete.js";

export function autocompleteEdit(interaction: AutocompleteInteraction) {
    // Only groups owned by the user can be edited.
    return respondGroupAutocomplete(interaction, (userId, guildId) =>
        backend.group.viewOwn(userId, guildId)
    );
}

export async function executeEdit(interaction: ChatInputCommandInteraction) {
    const groupIdStr = interaction.options.getString("group", true);
    const name = interaction.options.getString("name", false);
    const amount = interaction.options.getInteger("amount", false);
    const dueDay = interaction.options.getInteger("due_day", false);
    const paymentMethod = interaction.options.getString("payment_method", false);
    const account = interaction.options.getString("account", false);

    const groupId = parseInt(groupIdStr, 10);
    if (isNaN(groupId)) {
        await interaction.reply({
            content: "❌ Invalid group ID. Please select a group from the dropdown.",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    if ((paymentMethod && !account) || (!paymentMethod && account)) {
        await interaction.reply({
            content: "⚠️ **Payment Update Error**: When changing payment details, you must provide both the **Method** and the **Account Number**.",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    await interaction.deferReply({
        flags: MessageFlags.Ephemeral,
    });

    try {
        const payload = {
            name: name ?? "",
            amount: amount ?? null,
            due_day: dueDay ?? null,
            payment: {
                method: paymentMethod ?? "",
                account: account ?? ""
            }
        };

        await backend.group.update(payload, groupId);

        const embed = new EmbedBuilder()
            .setTitle("📝 Group Updated Successfully")
            .setDescription(`Changes applied to Group ID: \`${groupId}\``)
            .setColor(0x5865F2)
            .addFields(
                { name: "🏷️ Name", value: name || "*(Unchanged)*", inline: true },
                { name: "💰 Total Amount", value: amount ? `${amount} THB` : "*(Unchanged)*", inline: true },
                { name: "📅 Due Day", value: dueDay ? `Day ${dueDay}` : "*(Unchanged)*", inline: true },
                { name: "💳 Payment Method", value: paymentMethod || "*(Unchanged)*", inline: true },
                { name: "🔢 Account Number", value: account ? `\`${account}\`` : "*(Unchanged)*", inline: true }
            )
            .setTimestamp()

        await interaction.editReply({ embeds: [embed] });

    } catch (err: any) {
        console.error("Edit Group Error:", err);
        await interaction.editReply({
            content: `❌ **Failed to update group**\n> ${toUserMessage(err)}`,
        });
    }
}

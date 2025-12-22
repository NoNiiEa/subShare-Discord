import { ChatInputCommandInteraction, MessageFlags, EmbedBuilder, AutocompleteInteraction, } from "discord.js";
import { BackendClient } from "../../api/index.js";
import { config } from "../../config.js";

export async function autocompleteEdit(interaction: AutocompleteInteraction) {
    const focusedValue = interaction.options.getFocused();
    const userId = interaction.user.id;
    const guildId = interaction.guildId;

    if (!guildId) {
        await interaction.respond([]);
        return;
    }

    if (interaction.responded) {
        return;
    }

    try {
        const backend = new BackendClient({
            baseUrl: config.BACKEND_BASE_URL || "http://localhost:8000",
            apiKey: config.BACKEND_API_KEY
        });

        // Get only groups owned by the user
        const groups = await backend.group.viewOwn(userId, guildId);

        const searchTerm = focusedValue.toLowerCase();
        const filtered = groups.filter((group: any) => 
            group.name.toLowerCase().includes(searchTerm) ||
            String(group.id).includes(searchTerm)
        );

        const choices = filtered.slice(0, 25).map((group: any) => ({
            name: `${group.name} (ID: ${group.id})`,
            value: String(group.id)
        }));

        await interaction.respond(choices);

    } catch (err) {
        console.error("Autocomplete Error:", err);
        if (!interaction.responded) {
            await interaction.respond([]);
        }
    }
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

    const backend = new BackendClient({
        baseUrl: config.BACKEND_BASE_URL || "http://localhost:8000",
        apiKey: config.BACKEND_API_KEY
    });

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
        console.error("Edit Error:", err);
        
        const detail =
            err?.response?.data?.error ||
            err?.response?.data?.message ||
            err?.message ||
            "Unknown error occurred";

        await interaction.editReply({
            content: `❌ **Failed to update group**\n> ${detail}`,
        });
    }
}
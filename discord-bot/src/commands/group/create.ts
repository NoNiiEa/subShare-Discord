import { ChatInputCommandInteraction, MessageFlags, EmbedBuilder, } from "discord.js";
import { BackendClient } from "../../api/index.js";
import { config } from "../../config.js";

export async function executeCreate(interaction: ChatInputCommandInteraction) {
    const name = interaction.options.getString("name", true);
    const amount = interaction.options.getInteger("amount", true);
    const dueDay = interaction.options.getInteger("due_day", true);
    const paymentMethod = interaction.options.getString("payment_method", true);
    const account = interaction.options.getString("account", true);
    const ownerId = interaction.user.id;
    const guildId = interaction.guildId

    if (!guildId) {
        await interaction.reply({
            content: "This command can only be used inside a server (not in DMs).",
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
        const group = await backend.group.create({
            name,
            amount,
            due_day: dueDay,
            discord_guild_id: guildId,
            owner_discord_id: ownerId,
            payment: {
                method: paymentMethod,
                account,
            }
        });

        const createdAt = new Date(group.create_at).toLocaleString("th-TH", {
            timeZone: "Asia/Bangkok",
        });

        // Format payment method: show PromptPay instead of MSISDN
        const methodLower = (group.payment.method || '').toLowerCase();
        let displayMethod = group.payment.method || 'N/A';
        if (methodLower === 'msisdn') displayMethod = 'PromptPay';
        else if (methodLower === 'promptpay') displayMethod = 'PromptPay';
        else if (methodLower === 'bank_account' || methodLower === 'bank' || methodLower === 'bankac') displayMethod = 'Bank';

        const embed = new EmbedBuilder()
            .setTitle("✅ Group Created Successfully!")
            .setDescription(`**${group.name}**`)
            .setColor(0x57f287) // Discord success green
            .addFields(
                {
                    name: "Amount",
                    value: `**${group.amount}** (per person: ${group.amount_per_person})`,
                    inline: true,
                },
                {
                name: "Due Day",
                value: `**${group.due_day}**`,
                inline: true,
                },
                {
                name: "Owner",
                value: `<@${group.owner_discord_id}>`,
                inline: true,
                },
                {
                name: "Payment",
                value: `**${displayMethod}** → \`${group.payment.account}\``,
                inline: false,
                },
                {
                name: "Group ID",
                value: `\`${group.id}\``,
                inline: true,
                },
                {
                name: "Created At",
                value: createdAt,
                inline: true,
                }
            )
            .setFooter({
                text: `Guild: ${group.discord_guild_id}`,
            });

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
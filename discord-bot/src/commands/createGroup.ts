import {
    SlashCommandBuilder,
    ChatInputCommandInteraction,
    EmbedBuilder,
    MessageFlags
} from "discord.js";
import { BackendClient } from "../backendClient/index.js";

export const data = new SlashCommandBuilder()
    .setName("create-group")
    .setDescription("Create a new subscription group")
    .addStringOption((option) =>
        option
            .setName("name")
            .setDescription("Group name (e.g. Netflix Squad)")
            .setRequired(true)
    )
    .addIntegerOption((option) =>
        option
            .setName("amount")
            .setDescription("Total amount (e.g. 399)")
            .setRequired(true)
            .setMinValue(1)
    )
    .addIntegerOption((option) =>
        option
            .setName("due_day")
            .setDescription("Due day of month (1–31)")
            .setRequired(true)
            .setMinValue(1)
            .setMaxValue(31)
    )
    .addStringOption((option) =>
        option
            .setName("payment_method")
            .setDescription("Payment method")
            .setRequired(true)
            .addChoices(
                { name: "Bank account", value: "BANKAC" },
                { name: "PromptPay (MSISDN)", value: "MSISDN" },
            )
    )
    .addStringOption((option) =>
        option
            .setName("account")
            .setDescription("Account / phone number")
            .setRequired(true)
    );

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

    const name = interaction.options.getString("name", true);
    const amount = interaction.options.getInteger("amount", true);
    const dueDay = interaction.options.getInteger("due_day", true);
    const paymentMethod = interaction.options.getString("payment_method", true);
    const account = interaction.options.getString("account", true);
    const ownerId = interaction.user.id;

    const backend = new BackendClient(backendBaseUrl);

    await interaction.deferReply({
        flags: MessageFlags.Ephemeral,
    });


    try {
        const group = await backend.createGroup({
            name,
            amount,
            due_day: dueDay,
            discord_guild_id: guildId,
            owner_discord_id: ownerId,
            payment: {
                method: paymentMethod, // "BANKAC" or "MSISDN"
                account,
            },
        });

        const createdAt = new Date(group.create_at).toLocaleString("th-TH", {
            timeZone: "Asia/Bangkok",
        });

        const embed = new EmbedBuilder()
            .setTitle("✅ Group Created Successfully!")
            .setDescription(`**${group.name}**`)
            .setColor(0x57f287) // Discord success green
            .addFields(
                {
                    name: "💰 Amount",
                    value: `**${group.amount}** (per person: ${group.amount_per_person})`,
                    inline: true,
                },
                {
                name: "📅 Due Day",
                value: `**${group.due_day}**`,
                inline: true,
                },
                {
                name: "👑 Owner",
                value: `<@${group.owner_discord_id}>`,
                inline: true,
                },
                {
                name: "🏦 Payment",
                value: `**${group.payment.method}** → \`${group.payment.account}\``,
                inline: false,
                },
                {
                name: "🆔 Group ID",
                value: `\`${group.id}\``,
                inline: true,
                },
                {
                name: "🕒 Created At",
                value: createdAt,
                inline: true,
                }
            )
            .setFooter({
                text: `Guild: ${group.discord_guild_id}`,
            });

        await interaction.editReply({ embeds: [embed] });
    } catch (err: any) {
        console.error("create-group error:", err?.response?.data ?? err);

        const detail =
            err?.response?.data?.error ||
            err?.response?.data?.message ||
            JSON.stringify(err?.response?.data ?? {}, null, 2);

        await interaction.editReply(
            `❌ Failed to create group.\n\`\`\`json\n${detail}\n\`\`\``
        );
    }
}

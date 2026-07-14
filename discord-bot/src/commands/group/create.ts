import { ChatInputCommandInteraction, MessageFlags, EmbedBuilder } from "discord.js";
import { backend } from "../../utils/backend.js";
import { requireGuild } from "../../utils/interaction.js";
import { toUserMessage } from "../../utils/errors.js";
import { formatPaymentMethod } from "../../utils/format.js";

export async function executeCreate(interaction: ChatInputCommandInteraction) {
    const guildId = await requireGuild(interaction);
    if (!guildId) return;

    const name = interaction.options.getString("name", true);
    const amount = interaction.options.getInteger("amount", true);
    const dueDay = interaction.options.getInteger("due_day", true);
    const paymentMethod = interaction.options.getString("payment_method", true);
    const account = interaction.options.getString("account", true);
    const ownerId = interaction.user.id;

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
                value: `**${formatPaymentMethod(group.payment.method)}** → \`${group.payment.account}\``,
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
        console.error("Create Group Error:", err);
        await interaction.editReply({
            content: `❌ **Failed to create group**\n> ${toUserMessage(err)}`,
        });
    }
}

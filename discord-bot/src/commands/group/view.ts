import {
    ChatInputCommandInteraction,
    AutocompleteInteraction,
    MessageFlags,
    EmbedBuilder,
    User,
} from "discord.js";
import { backend } from "../../utils/backend.js";
import { toUserMessage } from "../../utils/errors.js";
import { formatPaymentMethod } from "../../utils/format.js";
import { respondGroupAutocomplete } from "../../utils/groupAutocomplete.js";
import { GroupResponse } from "../../api/types.js";

export function autocompleteView(interaction: AutocompleteInteraction) {
    return respondGroupAutocomplete(interaction, (userId, guildId) =>
        backend.group.view(userId, guildId)
    );
}

export async function executeView(interaction: ChatInputCommandInteraction) {
    const userId = interaction.user.id;
    const guildId = interaction.guildId;
    const selectedGroupId = interaction.options.getString("group");

    if (!guildId) {
        await interaction.reply({
            content: "This command can only be used inside a server (not in DMs).",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    await interaction.deferReply({
        flags: MessageFlags.Ephemeral,
    });

    try {
        // If a specific group is selected, show detailed view
        if (selectedGroupId) {
            const groupId = parseInt(selectedGroupId, 10);
            if (isNaN(groupId)) {
                await interaction.editReply({
                    content: "❌ Invalid group ID. Please select a group from the dropdown.",
                });
                return;
            }

            const group = await backend.group.get(groupId);

            const isMember = group.members.some((m) => m.member_id === userId);
            if (!isMember) {
                await interaction.editReply({
                    content: "❌ You are not a member of this group.",
                });
                return;
            }

            await interaction.editReply({ embeds: [buildGroupDetailEmbed(group, userId)] });
            return;
        }

        const groups = await backend.group.view(userId, guildId);

        if (!groups || groups.length === 0) {
            const emptyEmbed = new EmbedBuilder()
                .setTitle("📂 My Groups")
                .setDescription("You are not part of any subscription groups yet.")
                .setColor(0x95A5A6)
                .setThumbnail("https://cdn-icons-png.flaticon.com/512/7486/7486744.png");

            await interaction.editReply({ embeds: [emptyEmbed] });
            return;
        }

        await interaction.editReply({
            embeds: [buildDashboardEmbed(groups, userId, interaction.user)],
        });
    } catch (err: any) {
        console.error("View Group Error:", err);
        await interaction.editReply({
            content: `❌ **Failed to load groups**\n> ${toUserMessage(err)}`,
        });
    }
}

/** Summary dashboard embed across all of a user's groups. */
function buildDashboardEmbed(groups: GroupResponse[], userId: string, user: User): EmbedBuilder {
    const totalCost = groups.reduce((acc, curr) => acc + (Number(curr.amount_per_person) || 0), 0);
    const ownedGroups = groups.filter((g) => g.owner_discord_id === userId).length;

    const embed = new EmbedBuilder()
        .setTitle(`📊 Subscription Dashboard`)
        .setColor(0x5865F2)
        .setThumbnail(user.displayAvatarURL())
        .setDescription(`Here is a summary of your active subscriptions in this server.`)
        .addFields(
            { name: "💰 Monthly Total", value: `\`${totalCost.toLocaleString()} THB\``, inline: true },
            { name: "📂 Total Groups", value: `\`${groups.length} Active\``, inline: true },
            { name: "👑 Owned by You", value: `\`${ownedGroups} Groups\``, inline: true }
        );

    const maxDisplay = 10;

    groups.slice(0, maxDisplay).forEach((group, index) => {
        const isOwner = group.owner_discord_id === userId;
        const roleTag = isOwner ? "👑 Owner" : "👤 Member";
        const priceTag = isOwner
            ? `Total: **${group.amount}** THB`
            : `Share: **${group.amount_per_person}** THB`;

        embed.addFields({
            name: `${index + 1}. ${group.name}`,
            value: [
                `> ${roleTag} • 📅 Due: Day **${group.due_day}**`,
                `> 💸 ${priceTag}`,
                `> 💳 Method: \`${formatPaymentMethod(group.payment.method)}\``,
            ].join("\n"),
            inline: false,
        });
    });

    if (groups.length > maxDisplay) {
        embed.setFooter({ text: `...and ${groups.length - maxDisplay} more groups.` });
    } else {
        embed.setFooter({ text: "Use /group view to check your status anytime." });
    }

    return embed;
}

/** Detailed embed for a single group, including its member list. */
function buildGroupDetailEmbed(group: GroupResponse, userId: string): EmbedBuilder {
    const isOwner = group.owner_discord_id === userId;
    const createdAt = new Date(group.create_at).toLocaleDateString("en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
    });

    const embed = new EmbedBuilder()
        .setTitle(`📋 ${group.name}`)
        .setColor(isOwner ? 0xFEE75C : 0x5865F2)
        .setDescription(`Detailed information about this subscription group.`)
        .addFields(
            { name: "👑 Owner", value: `<@${group.owner_discord_id}>`, inline: true },
            { name: "💰 Total Amount", value: `**${group.amount}** THB`, inline: true },
            { name: "💵 Per Person", value: `**${group.amount_per_person}** THB`, inline: true },
            { name: "📅 Due Day", value: `Day **${group.due_day}** of each month`, inline: true },
            {
                name: "💳 Payment Method",
                value: `**${formatPaymentMethod(group.payment.method)}** → \`${group.payment.account}\``,
                inline: true,
            },
            { name: "📆 Created", value: createdAt, inline: true }
        );

    if (group.members && group.members.length > 0) {
        const memberList = group.members.map((member, index) => {
            const memberMention = `<@${member.member_id}>`;
            const isMemberOwner = member.member_id === group.owner_discord_id;
            const roleIcon = isMemberOwner ? "👑" : "👤";
            const statusIcon = member.status?.toLowerCase() === "active" ? "✅" : "⏳";

            const isPaid = (member.payment_status || "").toLowerCase() === "paid";
            const paymentStatus = isPaid ? "✅ Paid" : "❌ Unpaid";
            const debt = member.dept > 0 ? ` (Debt: ${member.dept} THB)` : "";

            return `${index + 1}. ${roleIcon} ${memberMention} ${statusIcon}\n   └─ ${paymentStatus}${debt}`;
        }).join("\n\n");

        embed.addFields({
            name: `👥 Members (${group.members.length})`,
            value: memberList.length > 1024 ? memberList.substring(0, 1020) + "..." : memberList,
            inline: false,
        });
    } else {
        embed.addFields({ name: "👥 Members", value: "No members found.", inline: false });
    }

    embed.setFooter({
        text: isOwner ? "You are the owner of this group" : "You are a member of this group",
    });

    return embed;
}

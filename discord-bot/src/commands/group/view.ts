import { ChatInputCommandInteraction, AutocompleteInteraction, MessageFlags, EmbedBuilder } from "discord.js";
import { BackendClient } from "../../api/index.js";
import { config } from "../../config.js";

export async function autocompleteView(interaction: AutocompleteInteraction) {
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

        const groups = await backend.group.view(userId, guildId);

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

export async function executeView(interaction: ChatInputCommandInteraction) {
    const userId = interaction.user.id
    const guildId = interaction.guildId
    const selectedGroupId = interaction.options.getString("group");

    if (!guildId) {
        await interaction.reply({
            content: "This command can only be used inside a server.",
            ephemeral: true,
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
            
            const isMember = group.members.some((m: any) => m.member_id === userId);
            if (!isMember) {
                await interaction.editReply({
                    content: "❌ You are not a member of this group.",
                });
                return;
            }

            await showGroupDetails(interaction, group, userId);
            return;
        }

        const groups = await backend.group.view(userId, guildId);

        if (groups.length === 0) {
            const emptyEmbed = new EmbedBuilder()
                .setTitle("📂 My Groups")
                .setDescription("You are not part of any subscription groups yet.")
                .setColor(0x95A5A6)
                .setThumbnail("https://cdn-icons-png.flaticon.com/512/7486/7486744.png");

            await interaction.editReply({ embeds: [emptyEmbed] });
            return;
        }

        const totalCost = groups.reduce((acc, curr) => acc + (Number(curr.amount_per_person) || 0), 0);
        const ownedGroups = groups.filter(g => g.owner_discord_id === userId).length;

        const embed = new EmbedBuilder()
            .setTitle(`📊 Subscription Dashboard`)
            .setColor(0x5865F2) 
            .setThumbnail(interaction.user.displayAvatarURL())
            .setDescription(`Here is a summary of your active subscriptions in this server.`)
            .addFields(
                { 
                    name: "💰 Monthly Total", 
                    value: `\`${totalCost.toLocaleString()} THB\``, 
                    inline: true 
                },
                { 
                    name: "📂 Total Groups", 
                    value: `\`${groups.length} Active\``, 
                    inline: true 
                },
                { 
                    name: "👑 Owned by You", 
                    value: `\`${ownedGroups} Groups\``, 
                    inline: true 
                }
            );

        const maxDisplay = 10; 
        
        groups.slice(0, maxDisplay).forEach((group, index) => {
            const isOwner = group.owner_discord_id === userId;
            const roleTag = isOwner ? "👑 Owner" : "👤 Member";
            const priceTag = isOwner 
                ? `Total: **${group.amount}** THB` 
                : `Share: **${group.amount_per_person}** THB`;

            const paymentMethod = (group.payment.method || '').toLowerCase();
            let displayMethod = group.payment.method || 'N/A';
            if (paymentMethod === 'msisdn') displayMethod = 'PromptPay';
            else if (paymentMethod === 'promptpay') displayMethod = 'PromptPay';
            else if (paymentMethod === 'bank_account' || paymentMethod === 'bank' || paymentMethod === 'bankac') displayMethod = 'Bank';

            embed.addFields({
                name: `${index + 1}. ${group.name}`, 
                value: [
                    `> ${roleTag} • 📅 Due: Day **${group.due_day}**`,
                    `> 💸 ${priceTag}`,
                    `> 💳 Method: \`${displayMethod}\``
                ].join("\n"),
                inline: false
            });
        });

        if (groups.length > maxDisplay) {
            embed.setFooter({ text: `...and ${groups.length - maxDisplay} more groups.` });
        } else {
            embed.setFooter({ text: "Use /group view to check your status anytime." });
        }

        await interaction.editReply({ embeds: [embed] });

    } catch (err: any) {
        const detail =
            err?.response?.data?.error ||
            err?.response?.data?.message ||
            "Unknown error occurred";

        await interaction.editReply({
            content: `❌ **Failed to load groups**\n> ${detail}`,
        });
    }
}

/**
 * Show detailed view of a specific group
 */
async function showGroupDetails(
    interaction: ChatInputCommandInteraction,
    group: any,
    userId: string
) {
    const isOwner = group.owner_discord_id === userId;
    const createdAt = new Date(group.create_at).toLocaleDateString("en-US", {
        year: "numeric",
        month: "short",
        day: "numeric"
    });

    // Format payment method
    const paymentMethod = (group.payment.method || '').toLowerCase();
    let displayMethod = group.payment.method || 'N/A';
    if (paymentMethod === 'msisdn') displayMethod = 'PromptPay';
    else if (paymentMethod === 'promptpay') displayMethod = 'PromptPay';
    else if (paymentMethod === 'bank_account' || paymentMethod === 'bank' || paymentMethod === 'bankac') displayMethod = 'Bank';

    const embed = new EmbedBuilder()
        .setTitle(`📋 ${group.name}`)
        .setColor(isOwner ? 0xFEE75C : 0x5865F2)
        .setDescription(`Detailed information about this subscription group.`)
        .addFields(
            {
                name: "👑 Owner",
                value: `<@${group.owner_discord_id}>`,
                inline: true
            },
            {
                name: "💰 Total Amount",
                value: `**${group.amount}** THB`,
                inline: true
            },
            {
                name: "💵 Per Person",
                value: `**${group.amount_per_person}** THB`,
                inline: true
            },
            {
                name: "📅 Due Day",
                value: `Day **${group.due_day}** of each month`,
                inline: true
            },
            {
                name: "💳 Payment Method",
                value: `**${displayMethod}** → \`${group.payment.account}\``,
                inline: true
            },
            {
                name: "📆 Created",
                value: createdAt,
                inline: true
            }
        );

    // Add members section
    if (group.members && group.members.length > 0) {
        const memberList = group.members.map((member: any, index: number) => {
            const memberMention = `<@${member.member_id}>`;
            const isMemberOwner = member.member_id === group.owner_discord_id;
            const roleIcon = isMemberOwner ? "👑" : "👤";
            const statusIcon = member.status === "active" || member.status === "Active" ? "✅" : "⏳";
            
            const paymentStatusLower = (member.payment_status || "").toLowerCase();
            const isPaid = paymentStatusLower === "paid";
            const paymentStatus = isPaid ? "✅ Paid" : "❌ Unpaid";
            const debt = member.dept > 0 ? ` (Debt: ${member.dept} THB)` : "";
            
            return `${index + 1}. ${roleIcon} ${memberMention} ${statusIcon}\n   └─ ${paymentStatus}${debt}`;
        }).join("\n\n");

        embed.addFields({
            name: `👥 Members (${group.members.length})`,
            value: memberList.length > 1024 
                ? memberList.substring(0, 1020) + "..." 
                : memberList,
            inline: false
        });
    } else {
        embed.addFields({
            name: "👥 Members",
            value: "No members found.",
            inline: false
        });
    }

    embed.setFooter({
        text: isOwner 
            ? "You are the owner of this group" 
            : "You are a member of this group"
    });

    await interaction.editReply({ embeds: [embed] });
}
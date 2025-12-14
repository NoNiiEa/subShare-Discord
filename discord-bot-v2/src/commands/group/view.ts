import { ChatInputCommandInteraction, MessageFlags, EmbedBuilder } from "discord.js";
import { BackendClient } from "../../api/index.js";
import { config } from "../../config.js";

export async function executeView(interaction: ChatInputCommandInteraction) {
    const userId = interaction.user.id
    const guildId = interaction.guildId

    if (!guildId) {
        await interaction.reply({
            content: "This command can only be used inside a server.",
            ephemeral: true,
        });
        return;
    }

    const backend = new BackendClient({
        baseUrl: config.BACKEND_BASE_URL || "http://localhost:8000",
    });

    await interaction.deferReply({
        flags: MessageFlags.Ephemeral,
    });

    try {
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

            embed.addFields({
                name: `${index + 1}. ${group.name}`, 
                value: [
                    `> ${roleTag} • 📅 Due: Day **${group.due_day}**`,
                    `> 💸 ${priceTag}`,
                    `> 💳 Method: \`${group.payment.method || 'N/A'}\``
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
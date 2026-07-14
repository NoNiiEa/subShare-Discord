import {
    ChatInputCommandInteraction,
    ActionRowBuilder,
    ButtonBuilder,
    ButtonStyle,
    EmbedBuilder,
    ComponentType,
    MessageFlags
} from "discord.js"
import { backend } from "../../utils/backend.js";
import { requireGuild } from "../../utils/interaction.js";

export async function executeAccept(interaction: ChatInputCommandInteraction) {
    const userId = interaction.user.id;

    const guildId = await requireGuild(interaction);
    if (!guildId) return;

    try {
        await interaction.deferReply({ flags: MessageFlags.Ephemeral });

        const invites = await backend.group.pendingInvite(userId, guildId);

        if (!invites || invites.length === 0) {
            await interaction.editReply("You have no pending invite.");
            return;
        }

        const invite = invites[0];

        const embed = new EmbedBuilder()
            .setTitle("📬 Group Invitation")
            .setDescription(`You have been invited to join **${invite.name}**`)
            .addFields(
                { name: "Invited by", value: `<@${invite.owner_discord_id}>`, inline: true },
                { name: "Price", value: `${invite.amount} THB`, inline: true }
            )
            .setColor("Blue");

        const acceptBtn = new ButtonBuilder()
            .setCustomId('accept')
            .setLabel('Accept')
            .setStyle(ButtonStyle.Success);

        const declineBtn = new ButtonBuilder()
            .setCustomId('decline')
            .setLabel('Decline')
            .setStyle(ButtonStyle.Danger);

        const row = new ActionRowBuilder<ButtonBuilder>().addComponents(acceptBtn, declineBtn);

        const message = await interaction.editReply({ embeds: [embed], components: [row] });

        const collector = message.createMessageComponentCollector({
            componentType: ComponentType.Button,
            time: 60000
        });

        collector.on('collect', async (i) => {
            if (i.user.id !== interaction.user.id) {
                await i.reply({ content: "This is not your invite!", flags: MessageFlags.Ephemeral });
                return;
            }

            try {
                if (i.customId === 'accept') {
                    await backend.group.acceptInvite({ user_id: interaction.user.id }, invite.id);
                    await i.update({ content: `🎉 You successfully joined **${invite.name}**!`, components: [], embeds: [] });
                } else if (i.customId === 'decline') {
                    await backend.group.DeclineInvite({ user_id: interaction.user.id }, invite.id);
                    await i.update({ content: `❌ You declined the invite to **${invite.name}**.`, components: [], embeds: [] });
                }
            } catch (err) {
                console.error("Accept/Decline Invite Error:", err);
                await i.update({ content: "❌ Something went wrong. Please try again later.", components: [], embeds: [] });
            } finally {
                collector.stop();
            }
        });

        collector.on('end', async (collected) => {
            if (collected.size === 0) {
                await interaction.editReply({ content: '⚠️ Invite action timed out.', components: [] }).catch(() => {});
            }
        });
    } catch (error) {
        console.error("Accept Invite Error:", error);
        await interaction.editReply("❌ Something went wrong fetching your invites.").catch(() => {});
    }
}

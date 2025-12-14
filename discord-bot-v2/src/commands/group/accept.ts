import {
    ChatInputCommandInteraction,
    ActionRowBuilder,
    ButtonBuilder,
    ButtonStyle,
    EmbedBuilder,
    ComponentType,
    MessageFlags
} from "discord.js"
import { BackendClient } from "../../api/index.js";
import { config } from "../../config.js";

export async function executeAccept(interaction: ChatInputCommandInteraction) {
    const userId = interaction.user.id
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
    });

    try {
        await interaction.deferReply({ flags: MessageFlags.Ephemeral })

        const invites = await backend.group.pendingInvite(userId, guildId)

        if (!invites || invites.length == 0) {
            await interaction.editReply("You have no pending invite.")
            return;
        }

        const invite = invites[0]

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

        const row = new ActionRowBuilder<ButtonBuilder>()
            .addComponents(acceptBtn, declineBtn);

        const message = await interaction.editReply({
            embeds: [embed],
            components: [row]
        });

        const collector = message.createMessageComponentCollector({ 
            componentType: ComponentType.Button, 
            time: 60000
        });

        collector.on('collect', async (i) => {
            if (i.user.id !== interaction.user.id) {
                await i.reply({ content: "This is not your invite!", flags: MessageFlags.Ephemeral });
                return;
            }

            if (i.customId === 'accept') {
                try {
                    await backend.group.acceptInvite({
                        user_id: interaction.user.id
                    }, invite.id)
                    await i.update({ content: `🎉 You successfully joined **${invite.name}**!`, components: [], embeds: [] });
                } catch (err) {
                    await i.reply({ content: "❌ Error accepting invite.", flags: MessageFlags.Ephemeral });
                }
            } else if (i.customId === 'decline') {
                try {
                    await backend.group.DeclineInvite({
                        user_id: interaction.user.id
                    }, invite.id)
                    await i.update({ content: `❌ You declined the invite to **${invite.name}**.`, components: [], embeds: [] });
                } catch (err) {
                    await i.reply({ content: "❌ Error declining invite.", flags: MessageFlags.Ephemeral });
                }
            }
        });

        collector.on('end', collected => {
            if (collected.size === 0) {
                interaction.editReply({ content: '⚠️ Invite action timed out.', components: [] });
            }
        });
    } catch (error) {
        console.log(error)
        await interaction.editReply("❌ Something went wrong fetching your invites.");
    }
}
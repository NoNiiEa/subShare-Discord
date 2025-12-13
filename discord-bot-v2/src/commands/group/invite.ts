import { 
    ChatInputCommandInteraction, 
    ActionRowBuilder, 
    StringSelectMenuBuilder, 
    StringSelectMenuOptionBuilder, 
    ComponentType,
    MessageFlags
} from "discord.js";
import { BackendClient } from "../../api";
import { config } from "../../config";

export async function executeInvite(interaction: ChatInputCommandInteraction) {
    const targetUser = interaction.options.getUser("invite-user", true);
    const userId = interaction.user.id;
    const guildId = interaction.guildId

    if (userId === targetUser.id) {
        await interaction.reply({
            content: "You can't invite yourself you your own group.",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

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

    const groups = await backend.group.viewOwn(userId, guildId);

    if (groups.length === 0) {
        return interaction.reply({ 
            content: "❌ You don't own any groups to invite people to.", 
            flags: MessageFlags.Ephemeral,
        });
    }

    const selectMenu = new StringSelectMenuBuilder()
        .setCustomId("group_select")
        .setPlaceholder("Select a group to invite them to...")
        .addOptions(
            groups.map((group) => 
                new StringSelectMenuOptionBuilder()
                    .setLabel(group.name)
                    .setDescription(`Price: ${group.amount}`)
                    .setValue(group.id.toString())
            )
        );

    const row = new ActionRowBuilder<StringSelectMenuBuilder>()
        .addComponents(selectMenu);

    const response = await interaction.reply({
        content: `Select which group you want to invite **${targetUser.username}** to:`,
        components: [row],
        flags: MessageFlags.Ephemeral,
    });

    try {
        const confirmation = await response.awaitMessageComponent({ 
            filter: (i) => i.user.id === interaction.user.id, // Ensure only the original user can click
            componentType: ComponentType.StringSelect,
            time: 60000 // 60 seconds timeout
        });

        const selectedGroupId = confirmation.values[0];
        const selectedGroup = groups.find(g => g.id.toString() === selectedGroupId) || { id: 0, name: "unknow"};

        const memberIds: string[] = [targetUser.id]
        
        
        await backend.group.invite({
            owner_id: userId,
            member_ids: memberIds
        }, selectedGroup.id)

        await confirmation.update({
            content: `Successfully invited **${targetUser.username}** to **${selectedGroup.name}**!`,
            components: []
        });
    } catch (err: any) {
        if (err.message && err.message.includes('[400]')) {
            await interaction.editReply({
                content: "User is already invite to this group.",
                components: [] 
            });
            return;
        } else if (err.message && err.message.includes('[500]')) {
            await interaction.editReply({
                content: "❌ Something went wrong while inviting.",
                components: [] 
            });
            return;
        }

        await interaction.editReply({ 
            content: "⏳ Invite cancelled: You took too long to select a group.", 
            components: [] 
        });
    }
}
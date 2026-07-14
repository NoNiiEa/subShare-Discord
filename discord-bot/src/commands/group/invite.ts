import {
    ChatInputCommandInteraction,
    ActionRowBuilder,
    StringSelectMenuBuilder,
    StringSelectMenuOptionBuilder,
    ComponentType,
    MessageFlags
} from "discord.js";
import { backend } from "../../utils/backend.js";
import { requireGuild } from "../../utils/interaction.js";
import { toUserMessage, ErrorRule } from "../../utils/errors.js";

const INVITE_ERROR_RULES: ErrorRule[] = [
    { status: 400, match: ["already"], message: "This user has already been invited to this group." },
];

export async function executeInvite(interaction: ChatInputCommandInteraction) {
    const targetUser = interaction.options.getUser("invite-user", true);
    const userId = interaction.user.id;

    if (userId === targetUser.id) {
        await interaction.reply({
            content: "You can't invite yourself to your own group.",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    const guildId = await requireGuild(interaction);
    if (!guildId) return;

    let groups;
    try {
        groups = await backend.group.viewOwn(userId, guildId);
    } catch (err: any) {
        console.error("Invite (fetch groups) Error:", err);
        await interaction.reply({
            content: `❌ ${toUserMessage(err)}`,
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    if (groups.length === 0) {
        await interaction.reply({
            content: "❌ You don't own any groups to invite people to.",
            flags: MessageFlags.Ephemeral,
        });
        return;
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

    const row = new ActionRowBuilder<StringSelectMenuBuilder>().addComponents(selectMenu);

    const response = await interaction.reply({
        content: `Select which group you want to invite **${targetUser.username}** to:`,
        components: [row],
        flags: MessageFlags.Ephemeral,
    });

    // Wait for the user to pick a group. A timeout is the only expected failure here.
    let confirmation;
    try {
        confirmation = await response.awaitMessageComponent({
            filter: (i) => i.user.id === interaction.user.id,
            componentType: ComponentType.StringSelect,
            time: 60000,
        });
    } catch {
        await interaction.editReply({
            content: "⏳ Invite cancelled: You took too long to select a group.",
            components: [],
        }).catch(() => {});
        return;
    }

    const selectedGroup = groups.find((g) => g.id.toString() === confirmation.values[0]);
    if (!selectedGroup) {
        await confirmation.update({ content: "❌ Selected group not found.", components: [] });
        return;
    }

    try {
        await backend.group.invite({ owner_id: userId, member_ids: [targetUser.id] }, selectedGroup.id);
        await confirmation.update({
            content: `Successfully invited **${targetUser.username}** to **${selectedGroup.name}**!`,
            components: [],
        });
    } catch (err: any) {
        console.error("Invite Error:", err);
        await confirmation.update({
            content: `❌ ${toUserMessage(err, INVITE_ERROR_RULES)}`,
            components: [],
        });
    }
}

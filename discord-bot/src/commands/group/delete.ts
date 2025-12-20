import { ChatInputCommandInteraction, AutocompleteInteraction, MessageFlags, EmbedBuilder } from "discord.js";
import { BackendClient } from "../../api/index.js";
import { config } from "../../config.js";

export async function autocompleteDelete(interaction: AutocompleteInteraction) {
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

        // Get only groups owned by the user
        const groups = await backend.group.viewOwn(userId, guildId);

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

export async function executeDelete(interaction: ChatInputCommandInteraction) {
    const userId = interaction.user.id;
    const guildId = interaction.guildId;
    const groupIdStr = interaction.options.getString("group", true);
    
    // Parse the group ID from string
    const groupId = parseInt(groupIdStr, 10);
    if (isNaN(groupId)) {
        await interaction.reply({
            content: "❌ Invalid group ID. Please select a group from the dropdown.",
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
        apiKey: config.BACKEND_API_KEY
    });

    await interaction.deferReply({
        flags: MessageFlags.Ephemeral,
    });

    try {
        // First, verify the group exists and user is the owner
        const group = await backend.group.get(groupId);

        if (!group) {
            await interaction.editReply({
                content: "❌ Group not found.",
            });
            return;
        }

        // Check if user is the owner
        if (group.owner_discord_id !== userId) {
            await interaction.editReply({
                content: "❌ You can only delete groups that you own.",
            });
            return;
        }

        // Delete the group
        await backend.group.delete(groupId);

        const embed = new EmbedBuilder()
            .setTitle("✅ Group Deleted Successfully")
            .setDescription(`**${group.name}** has been permanently deleted.`)
            .setColor(0xED4245) // Discord red
            .addFields(
                {
                    name: "Group ID",
                    value: `\`${group.id}\``,
                    inline: true,
                },
                {
                    name: "Deleted by",
                    value: `<@${userId}>`,
                    inline: true,
                }
            )
            .setFooter({
                text: "All associated bills have also been deleted.",
            });

        await interaction.editReply({ embeds: [embed] });
    } catch (err: any) {
        const detail =
            err?.response?.data?.error ||
            err?.response?.data?.message ||
            err?.message ||
            "Unknown error occurred";

        // Handle specific error cases
        if (err?.response?.status === 404) {
            await interaction.editReply({
                content: "❌ Group not found.",
            });
            return;
        }

        if (err?.response?.status === 403) {
            await interaction.editReply({
                content: "❌ You don't have permission to delete this group.",
            });
            return;
        }

        await interaction.editReply({
            content: `❌ **Failed to delete group**\n> ${detail}`,
        });
    }
}

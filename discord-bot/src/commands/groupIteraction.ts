import {
    ChatInputCommandInteraction,
    SlashCommandBuilder,
    StringSelectMenuBuilder,
    ActionRowBuilder,
    ComponentType,
    EmbedBuilder,
} from "discord.js";
import { BackendClient } from "../backendClient";

export const data = new SlashCommandBuilder()
    .setName("group")
    .setDescription("Choose a group and perform an action")
    .addStringOption(option =>
        option
            .setName("action")
            .setDescription("What do you want to do with the group?")
            .setRequired(true)
            .addChoices(
                { name: "create", value: "create" },
                { name: "Invite", value: "invite" },
                { name: "View", value: "view" },
                { name: "Delete", value: "delete" },
            )
    );




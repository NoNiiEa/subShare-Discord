import {
    SlashCommandBuilder,
    SlashCommandSubcommandBuilder,
    ChatInputCommandInteraction,
    AutocompleteInteraction,
} from "discord.js";
import { executeCreate } from "./create.js";
import { executeView, autocompleteView } from "./view.js";
import { executeInvite } from "./invite.js";
import { executeAccept } from "./accept.js";
import { autocompleteEdit, executeEdit } from "./edit.js";
import { executeDelete, autocompleteDelete } from "./delete.js";

/**
 * Adds the shared group fields (name/amount/due_day/payment_method/account) to a
 * subcommand. Used by both `create` (required) and `edit` (optional).
 */
function addGroupOptions(
    subcommand: SlashCommandSubcommandBuilder,
    required: boolean
): SlashCommandSubcommandBuilder {
    return subcommand
        .addStringOption((option) =>
            option
                .setName("name")
                .setDescription("Group name (e.g. Netflix Squad)")
                .setRequired(required)
        )
        .addIntegerOption((option) =>
            option
                .setName("amount")
                .setDescription("Total amount (e.g. 399)")
                .setRequired(required)
                .setMinValue(1)
        )
        .addIntegerOption((option) =>
            option
                .setName("due_day")
                .setDescription("Due day of month (1–31)")
                .setRequired(required)
                .setMinValue(1)
                .setMaxValue(31)
        )
        .addStringOption((option) =>
            option
                .setName("payment_method")
                .setDescription("Payment method")
                .setRequired(required)
                .addChoices(
                    { name: "Bank account", value: "BANKAC" },
                    { name: "PromptPay", value: "MSISDN" }
                )
        )
        .addStringOption((option) =>
            option
                .setName("account")
                .setDescription("Account / phone number")
                .setRequired(required)
        );
}

export default {
    data: new SlashCommandBuilder()
        .setName("group")
        .setDescription("group interaction")

        .addSubcommand((subcommand) =>
            addGroupOptions(
                subcommand.setName("create").setDescription("create group"),
                true
            )
        )
        .addSubcommand((subcommand) =>
            subcommand
                .setName("view")
                .setDescription("view my group")
                .addStringOption((option) =>
                    option
                        .setName("group")
                        .setDescription("Select a specific group to view details (optional)")
                        .setRequired(false)
                        .setAutocomplete(true)
                )
        )
        .addSubcommand((subcommand) =>
            subcommand
                .setName("invite")
                .setDescription("invite member to your group")
                .addUserOption((option) =>
                    option
                        .setName("invite-user")
                        .setDescription("the person you want to invite")
                        .setRequired(true)
                )
        )
        .addSubcommand((subcommand) =>
            subcommand
                .setName("accept")
                .setDescription("accept invite to a group")
        )
        .addSubcommand((subcommand) =>
            subcommand
                .setName("delete")
                .setDescription("delete your group")
                .addStringOption((option) =>
                    option
                        .setName("group")
                        .setDescription("Select a group to delete")
                        .setRequired(true)
                        .setAutocomplete(true)
                )
        )
        .addSubcommand((subcommand) =>
            addGroupOptions(
                subcommand
                    .setName("edit")
                    .setDescription("Edit your group")
                    .addStringOption((option) =>
                        option
                            .setName("group")
                            .setDescription("Select a group to edit")
                            .setRequired(true)
                            .setAutocomplete(true)
                    ),
                false
            )
        ),

    async execute(interaction: ChatInputCommandInteraction) {
        const subcommand = interaction.options.getSubcommand();

        switch (subcommand) {
            case "create":
                await executeCreate(interaction);
                break;
            case "view":
                await executeView(interaction);
                break;
            case "invite":
                await executeInvite(interaction);
                break;
            case "accept":
                await executeAccept(interaction);
                break;
            case "delete":
                await executeDelete(interaction);
                break;
            case "edit":
                await executeEdit(interaction);
                break;
        }
    },

    async autocomplete(interaction: AutocompleteInteraction) {
        const subcommand = interaction.options.getSubcommand();

        switch (subcommand) {
            case "view":
                await autocompleteView(interaction);
                break;
            case "delete":
                await autocompleteDelete(interaction);
                break;
            case "edit":
                await autocompleteEdit(interaction);
                break;
        }
    }
}

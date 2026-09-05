import {
    SlashCommandBuilder,
    ChatInputCommandInteraction,
    AutocompleteInteraction,
    MessageFlags
} from "discord.js";
import { executePaid, autocompletePaid } from "./paid.js";
import { executePayAll } from "./payall.js";

export default {
    data: new SlashCommandBuilder()
        .setName("bill")
        .setDescription("Bill management commands")
        
        // Subcommand: /bill paid
        .addSubcommand((subcommand) =>
            subcommand
                .setName("paid")
                .setDescription("Submit a payment slip for a bill")
                
                // 1. Bill Selection (Autocomplete)
                // This allows the user to search/select specific unpaid bills
                .addStringOption((option) =>
                    option
                        .setName("target_bill")
                        .setDescription("Select the bill you are paying for")
                        .setRequired(true)
                        .setAutocomplete(true) // <--- Enables the dynamic dropdown
                )

                // 2. Slip Upload (Image)
                .addAttachmentOption((option) =>
                    option
                        .setName("slip")
                        .setDescription("Upload your transfer slip (JPG/PNG)")
                        .setRequired(true)
                )
        )

        // Subcommand: /bill payall
        // Takes no options at all: bills are chosen from a select menu and the
        // slip is collected mid-flow by a modal, so the payment is one command
        // and the bill selection can never be skipped.
        .addSubcommand((subcommand) =>
            subcommand
                .setName("payall")
                .setDescription("Select several bills and pay them with one slip")
        ),

    // MAIN EXECUTION HANDLER (User hits Enter)
    async execute(interaction: ChatInputCommandInteraction) {
        const subcommand = interaction.options.getSubcommand();

        switch (subcommand) {
            case "paid":
                await executePaid(interaction);
                break;
            case "payall":
                await executePayAll(interaction);
                break;
            default:
                await interaction.reply({
                    content: "Unknown subcommand.",
                    flags: MessageFlags.Ephemeral
                });
        }
    },

    // AUTOCOMPLETE HANDLER (User types in target_bill)
    // This function is called every time the user types a character in the dropdown
    async autocomplete(interaction: AutocompleteInteraction) {
        const subcommand = interaction.options.getSubcommand();

        switch (subcommand) {
            case "paid":
                await autocompletePaid(interaction);
                break;
        }
    }
};
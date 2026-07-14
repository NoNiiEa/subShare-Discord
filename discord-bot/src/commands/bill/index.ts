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
        // Bills are chosen from a select menu rather than an option, since a
        // slash command cannot collect a multi-select up front. The slip is
        // optional: without it the command only quotes the total, so the user
        // can find out what to transfer before transferring it.
        .addSubcommand((subcommand) =>
            subcommand
                .setName("payall")
                .setDescription("Total up several bills, or pay them all with one slip")
                .addAttachmentOption((option) =>
                    option
                        .setName("slip")
                        .setDescription("Your transfer slip (JPG/PNG). Leave empty to just see the total first.")
                        .setRequired(false)
                )
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
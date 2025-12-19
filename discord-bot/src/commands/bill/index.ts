import { 
    SlashCommandBuilder, 
    ChatInputCommandInteraction, 
    AutocompleteInteraction 
} from "discord.js";
import { executePaid, autocompletePaid } from "./paid.js";

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
        ),

    // MAIN EXECUTION HANDLER (User hits Enter)
    async execute(interaction: ChatInputCommandInteraction) {
        const subcommand = interaction.options.getSubcommand();

        switch (subcommand) {
            case "paid":
                await executePaid(interaction);
                break;
            default:
                await interaction.reply({ 
                    content: "Unknown subcommand.", 
                    ephemeral: true 
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
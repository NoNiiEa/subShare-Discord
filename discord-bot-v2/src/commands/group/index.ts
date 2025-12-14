import {
    SlashCommandBuilder,
    ChatInputCommandInteraction,
    EmbedBuilder,
    MessageFlags
} from "discord.js";
import { executeCreate } from "./create.js";
import { executeView } from "./view.js";
import { executeInvite } from "./invite.js";
import { executeAccept } from "./accept.js";

export default {
    data: new SlashCommandBuilder()
        .setName("group")
        .setDescription("group interaction")

        .addSubcommand((subcommand) => 
            subcommand
                .setName("create")
                .setDescription("create group")
                .addStringOption((option) =>
                    option
                        .setName("name")
                        .setDescription("Group name (e.g. Netflix Squad)")
                        .setRequired(true)
                )
                .addIntegerOption((option) =>
                    option
                        .setName("amount")
                        .setDescription("Total amount (e.g. 399)")
                        .setRequired(true)
                        .setMinValue(1)
                )
                .addIntegerOption((option) =>
                    option
                        .setName("due_day")
                        .setDescription("Due day of month (1–31)")
                        .setRequired(true)
                        .setMinValue(1)
                        .setMaxValue(31)
                )
                .addStringOption((option) =>
                    option
                        .setName("payment_method")
                        .setDescription("Payment method")
                        .setRequired(true)
                        .addChoices(
                            { name: "Bank account", value: "BANKAC" },
                            { name: "PromptPay (MSISDN)", value: "MSISDN" },
                        )
                )
                .addStringOption((option) =>
                    option
                        .setName("account")
                        .setDescription("Account / phone number")
                        .setRequired(true)
                )
        )
        .addSubcommand((subcommand) => 
            subcommand
                .setName("view")
                .setDescription("view my group")
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
        }
    }
}
import {
    ChatInputCommandInteraction,
    ActionRowBuilder,
    StringSelectMenuBuilder,
    StringSelectMenuOptionBuilder,
    ButtonBuilder,
    ButtonStyle,
    ComponentType,
    MessageFlags,
} from "discord.js";
import { backend } from "../../utils/backend.js";
import { fetchGroupMap } from "../../utils/groups.js";
import { requireGuild } from "../../utils/interaction.js";
import { toUserMessage, PAYMENT_ERROR_RULES } from "../../utils/errors.js";
import { formatPaymentMethod, maskAccount } from "../../utils/format.js";
import { BillResponse, GroupResponse, PayMultipleResponse } from "../../api/types.js";

/** Discord's hard cap on select-menu options. The backend enforces it too. */
const MAX_SELECT = 25;
const STEP_TIMEOUT = 120_000;

const SLIP_FRESHNESS_NOTE =
    "> ⚠️ Payment slips must be uploaded within **30 minutes** of making the transfer.";

/**
 * A set of bills payable by one slip: same group owner, same receiving account.
 * A slip has exactly one receiver, so bills across clusters cannot be combined.
 */
interface PayeeCluster {
    key: string;
    method: string;
    account: string;
    bills: BillResponse[];
    groups: Map<number, GroupResponse>;
}

function digitsOnly(account: string): string {
    return (account || "").replace(/\D/g, "");
}

function clusterByPayee(
    bills: BillResponse[],
    groupMap: Map<number, GroupResponse>
): PayeeCluster[] {
    const clusters = new Map<string, PayeeCluster>();

    for (const bill of bills) {
        const group = groupMap.get(bill.group_id);
        if (!group) continue; // payee unknown — cannot safely batch it

        const method = group.payment?.method ?? "";
        const account = group.payment?.account ?? "";
        const key = `${group.owner_discord_id}|${method}|${digitsOnly(account)}`;

        let cluster = clusters.get(key);
        if (!cluster) {
            cluster = { key, method, account, bills: [], groups: new Map() };
            clusters.set(key, cluster);
        }
        cluster.bills.push(bill);
        cluster.groups.set(group.id, group);
    }

    return [...clusters.values()];
}

function clusterTotal(cluster: PayeeCluster): number {
    return cluster.bills.reduce((sum, bill) => sum + bill.amount_due, 0);
}

function payeeLabel(cluster: PayeeCluster): string {
    return `${formatPaymentMethod(cluster.method)} ****${maskAccount(cluster.account)}`;
}

function currencyOf(bills: BillResponse[]): string {
    return bills[0]?.currency ?? "";
}

/** Truncates to Discord's 100-char limit for option labels and descriptions. */
function truncate(text: string, max = 100): string {
    return text.length > max ? text.slice(0, max - 3) + "..." : text;
}

function buildBillSelect(cluster: PayeeCluster, selected: Set<number>): ActionRowBuilder<StringSelectMenuBuilder> {
    const bills = cluster.bills.slice(0, MAX_SELECT);

    const menu = new StringSelectMenuBuilder()
        .setCustomId("bill_select")
        .setPlaceholder("Select the bills you are paying for...")
        .setMinValues(1)
        .setMaxValues(bills.length)
        .addOptions(
            bills.map((bill) => {
                const groupName = cluster.groups.get(bill.group_id)?.name ?? `Group #${bill.group_id}`;
                const detail = bill.description
                    ? `${bill.month}/${bill.year} · ${bill.description}`
                    : `${bill.month}/${bill.year}`;

                return new StringSelectMenuOptionBuilder()
                    .setLabel(truncate(`${groupName} — ${bill.amount_due} ${bill.currency}`))
                    .setDescription(truncate(detail))
                    .setValue(String(bill.id))
                    .setDefault(selected.has(bill.id));
            })
        );

    return new ActionRowBuilder<StringSelectMenuBuilder>().addComponents(menu);
}

function buildConfirmRow(disabled: boolean): ActionRowBuilder<ButtonBuilder> {
    return new ActionRowBuilder<ButtonBuilder>().addComponents(
        new ButtonBuilder()
            .setCustomId("confirm")
            .setLabel("Confirm payment")
            .setStyle(ButtonStyle.Success)
            .setDisabled(disabled),
        new ButtonBuilder()
            .setCustomId("cancel")
            .setLabel("Cancel")
            .setStyle(ButtonStyle.Danger)
    );
}

/**
 * "quote" totals the selection up so the user knows what to transfer; "pay"
 * settles it with the slip they already attached.
 */
type Mode = "quote" | "pay";

function selectionPrompt(
    cluster: PayeeCluster,
    selected: Set<number>,
    truncated: boolean,
    mode: Mode
): string {
    let message =
        mode === "pay"
            ? `💸 **Paying ${payeeLabel(cluster)}**\n${SLIP_FRESHNESS_NOTE}\n\n`
            : `🧮 **Bills owed to ${payeeLabel(cluster)}**\n\n`;

    if (truncated) {
        message += `> ℹ️ You have more than ${MAX_SELECT} bills to this account; only the first ${MAX_SELECT} are shown.\n\n`;
    }

    if (selected.size === 0) {
        message +=
            mode === "pay"
                ? "Select the bills you want to pay with this slip."
                : "Select the bills you want to pay to see what to transfer.";
        return message;
    }

    const chosen = cluster.bills.filter((bill) => selected.has(bill.id));
    const total = chosen.reduce((sum, bill) => sum + bill.amount_due, 0);
    const currency = currencyOf(chosen);

    message += `**Selected ${chosen.length} bill${chosen.length === 1 ? "" : "s"}:**\n`;
    for (const bill of chosen) {
        const groupName = cluster.groups.get(bill.group_id)?.name ?? `Group #${bill.group_id}`;
        message += `> • ${groupName} — ${bill.amount_due} ${bill.currency}\n`;
    }

    if (mode === "pay") {
        message += `\n**Total: ${total} ${currency}** — your slip must be for at least this amount.`;
        return message;
    }

    message += `\n**Total: ${total} ${currency}**\n\n`;
    message += `Transfer **${total} ${currency}** to **${formatPaymentMethod(cluster.method)} \`${cluster.account}\`**, `;
    message += `then run \`/bill payall\` again with your slip attached and select these same bills.`;

    return message;
}

/**
 * The public receipt. Posted to the channel rather than ephemerally, so the
 * group owner can see the payment and it stands as a record — hence naming the
 * payer instead of addressing them as "you".
 */
function buildResultMessage(
    res: PayMultipleResponse,
    cluster: PayeeCluster,
    slipUrl: string,
    payerId: string
): string {
    let message = `✅ **Payment Verified — ${res.bills.length} bill${res.bills.length === 1 ? "" : "s"} settled**\n`;
    message += `Paid by <@${payerId}> to ${payeeLabel(cluster)}\n\n`;

    for (const bill of res.bills) {
        const groupName = cluster.groups.get(bill.group_id)?.name ?? `Group #${bill.group_id}`;
        message += `> • **${groupName}** (Bill \`${bill.id}\`) — ${bill.amount_due} ${bill.currency}\n`;
    }

    const currency = currencyOf(res.bills);
    message += `\n> **Total due:** ${res.total_due} ${currency}\n`;
    message += `> **Slip amount:** ${res.amount_paid} ${currency}\n`;

    const surplus = res.amount_paid - res.total_due;
    if (surplus > 0) {
        if (res.surplus_credited > 0) {
            message += `> ℹ️ Overpaid by ${surplus} ${currency} — credited against their remaining debt.\n`;
        } else {
            message += `> ⚠️ Overpaid by ${surplus} ${currency}. These bills span several groups, so it could not be credited automatically.\n`;
        }
    }

    message += `> [View Payment Slip](${slipUrl})`;

    return message;
}

/** Lets the user pick which payee to settle when their bills span several. */
async function pickCluster(
    interaction: ChatInputCommandInteraction,
    clusters: PayeeCluster[]
): Promise<PayeeCluster | null> {
    if (clusters.length === 1) return clusters[0];

    const menu = new StringSelectMenuBuilder()
        .setCustomId("payee_select")
        .setPlaceholder("Select who you are paying...")
        .addOptions(
            clusters.slice(0, MAX_SELECT).map((cluster) =>
                new StringSelectMenuOptionBuilder()
                    .setLabel(truncate(payeeLabel(cluster)))
                    .setDescription(
                        truncate(
                            `${cluster.bills.length} bill(s) · ${clusterTotal(cluster)} ${currencyOf(cluster.bills)}`
                        )
                    )
                    .setValue(cluster.key)
            )
        );

    const response = await interaction.editReply({
        content:
            "Your unpaid bills are owed to more than one account, and a single slip can only pay one of them.\n" +
            "Select which account you are paying:",
        components: [new ActionRowBuilder<StringSelectMenuBuilder>().addComponents(menu)],
    });

    try {
        const choice = await response.awaitMessageComponent({
            filter: (i) => i.user.id === interaction.user.id,
            componentType: ComponentType.StringSelect,
            time: STEP_TIMEOUT,
        });
        await choice.deferUpdate();
        return clusters.find((c) => c.key === choice.values[0]) ?? null;
    } catch {
        await interaction
            .editReply({ content: "⏳ Timed out: you took too long to choose.", components: [] })
            .catch(() => {});
        return null;
    }
}

export async function executePayAll(interaction: ChatInputCommandInteraction) {
    const guildId = await requireGuild(interaction);
    if (!guildId) return;

    // No slip means the user just wants the total, so they know what to transfer.
    const slip = interaction.options.getAttachment("slip");
    const mode: Mode = slip ? "pay" : "quote";

    if (slip && !slip.contentType?.startsWith("image/")) {
        await interaction.reply({
            content: "❌ Please upload a valid image file (JPG/PNG).",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    // The whole selection flow is ephemeral: only the payer needs to see it.
    await interaction.deferReply({ flags: MessageFlags.Ephemeral });

    let bills: BillResponse[];
    try {
        bills = await backend.bill.GetUnpaidByUserAndGuild(interaction.user.id, guildId);
    } catch (err: any) {
        console.error("Pay All (fetch bills) Error:", err);
        await interaction.editReply({ content: `❌ ${toUserMessage(err)}` });
        return;
    }

    if (!bills || bills.length === 0) {
        await interaction.editReply({ content: "🎉 You have no unpaid bills in this server." });
        return;
    }

    const groupMap = await fetchGroupMap(bills.map((bill) => bill.group_id));
    const clusters = clusterByPayee(bills, groupMap);

    if (clusters.length === 0) {
        await interaction.editReply({
            content: "❌ Could not load the groups for your bills. Please try again in a moment.",
        });
        return;
    }

    const cluster = await pickCluster(interaction, clusters);
    if (!cluster) return;

    const truncated = cluster.bills.length > MAX_SELECT;
    const selected = new Set<number>();

    // Quote mode has nothing to confirm — the running total is the whole point.
    const rowsFor = (sel: Set<number>) =>
        mode === "pay"
            ? [buildBillSelect(cluster, sel), buildConfirmRow(sel.size === 0)]
            : [buildBillSelect(cluster, sel)];

    const message = await interaction.editReply({
        content: selectionPrompt(cluster, selected, truncated, mode),
        components: rowsFor(selected),
    });

    // A collector rather than awaitMessageComponent: the user must be able to
    // adjust their selection and watch the total before confirming.
    const collector = message.createMessageComponentCollector({ time: STEP_TIMEOUT });

    collector.on("collect", async (i) => {
        if (i.user.id !== interaction.user.id) {
            await i.reply({ content: "This isn't your payment.", flags: MessageFlags.Ephemeral });
            return;
        }

        if (i.isStringSelectMenu()) {
            selected.clear();
            for (const value of i.values) selected.add(Number(value));

            await i.update({
                content: selectionPrompt(cluster, selected, truncated, mode),
                components: rowsFor(selected),
            });
            return;
        }

        if (i.customId === "cancel") {
            collector.stop("cancel");
            await i.update({ content: "Payment cancelled.", components: [] });
            return;
        }

        if (i.customId === "confirm" && slip) {
            collector.stop("confirm");
            await i.update({ content: "⏳ **Verifying your slip...**", components: [] });

            try {
                const res = await backend.bill.PayMultiple({
                    user_id: interaction.user.id,
                    guild_id: guildId,
                    bill_ids: [...selected],
                    proof_url: slip.url,
                });

                // The selection flow was ephemeral and cannot be made public, so
                // the receipt goes out as a separate public follow-up: the group
                // owner needs to see it, and it serves as the payment log.
                await interaction.editReply({ content: "✅ Payment verified — receipt posted below." });
                await interaction.followUp({
                    content: buildResultMessage(res, cluster, slip.url, interaction.user.id),
                });
            } catch (err: any) {
                console.error("Pay All Error:", err);
                const errorMessage = toUserMessage(
                    err,
                    PAYMENT_ERROR_RULES,
                    "❌ Failed to verify payment. Please try again later."
                );
                await interaction.editReply({
                    content: `**Failed to submit payment.**\n> ${errorMessage}`,
                });
            }
        }
    });

    collector.on("end", async (_collected, reason) => {
        if (reason === "confirm" || reason === "cancel") return;

        // In quote mode the totals are still worth reading, so leave them up and
        // just retire the menu.
        const content =
            mode === "pay"
                ? "⏳ Payment cancelled: you took too long to confirm."
                : selectionPrompt(cluster, selected, truncated, "quote");

        await interaction.editReply({ content, components: [] }).catch(() => {});
    });
}

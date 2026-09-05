import {
    ChatInputCommandInteraction,
    ButtonInteraction,
    ModalSubmitInteraction,
    Attachment,
    ActionRowBuilder,
    StringSelectMenuBuilder,
    StringSelectMenuOptionBuilder,
    ButtonBuilder,
    ButtonStyle,
    ModalBuilder,
    LabelBuilder,
    FileUploadBuilder,
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

/**
 * The selection step spans a trip to the user's banking app, so it has to stay
 * open far longer than a normal prompt. 14 minutes leaves headroom under
 * Discord's 15-minute interaction-token lifetime, which is the hard ceiling.
 */
const SELECT_TIMEOUT = 840_000;
/** Time to fill in and submit the slip modal once it is open. */
const MODAL_TIMEOUT = 600_000;
/** Choosing a payee is a quick decision with no transfer in between. */
const PAYEE_TIMEOUT = 120_000;

const SLIP_FIELD = "slip";
const NOT_AN_IMAGE = "❌ Please upload a valid image file (JPG/PNG).";

const SLIP_FRESHNESS_NOTE =
    "> ⚠️ Payment slips must be uploaded within **30 minutes** of making the transfer.";

/**
 * A set of bills payable by one slip: same group owner, same receiving account.
 * A slip has exactly one receiver, so bills across clusters cannot be combined.
 */
interface PayeeCluster {
    key: string;
    /** The group owner being paid. A cluster is keyed by them, so they are the payee. */
    ownerId: string;
    /** Their guild display name, resolved separately; "" when it could not be looked up. */
    ownerName: string;
    method: string;
    account: string;
    bills: BillResponse[];
    groups: Map<number, GroupResponse>;
}

/**
 * Strips everything but digits. Mirrors the backend's ExtractNumericCharacters,
 * which uses unicode.IsDigit — hence \p{Nd} rather than a bare \D, so an account
 * written with non-ASCII numerals clusters the same on both sides.
 */
function digitsOnly(account: string): string {
    return (account || "").replace(/[^0-9\p{Nd}]/gu, "");
}

function isImage(file: Attachment): boolean {
    return Boolean(file.contentType?.startsWith("image/"));
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
            cluster = {
                key,
                ownerId: group.owner_discord_id,
                ownerName: "",
                method,
                account,
                bills: [],
                groups: new Map(),
            };
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

function accountLabel(cluster: PayeeCluster): string {
    // Bullets rather than asterisks: this label is embedded in **bold** text, and
    // "****1234" collides with Discord's markdown, swallowing the mask.
    return `${formatPaymentMethod(cluster.method)} ••••${maskAccount(cluster.account)}`;
}

/**
 * A masked account on its own does not tell the payer who they are sending money
 * to, so lead with the payee's name whenever it could be resolved.
 */
function payeeLabel(cluster: PayeeCluster): string {
    const account = accountLabel(cluster);
    return cluster.ownerName ? `${cluster.ownerName} — ${account}` : account;
}

/**
 * Fills in each cluster's payee name from the guild. Fetches only the owners
 * involved rather than the whole membership, as the reminder service does.
 * Best-effort: on failure the labels simply fall back to the account alone.
 */
async function attachOwnerNames(
    interaction: ChatInputCommandInteraction,
    clusters: PayeeCluster[]
): Promise<void> {
    const ownerIds = [...new Set(clusters.map((c) => c.ownerId))].filter(Boolean);
    if (ownerIds.length === 0 || !interaction.guild) return;

    const members = await interaction.guild.members.fetch({ user: ownerIds }).catch(() => null);
    if (!members) return;

    for (const cluster of clusters) {
        cluster.ownerName = members.get(cluster.ownerId)?.displayName ?? "";
    }
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

/**
 * The primary action opens the slip modal, so the whole payment happens in one
 * interaction. A slip can only ever be supplied after bills are chosen, which is
 * what keeps the selection from being skipped.
 */
function buildActionRow(disabled: boolean): ActionRowBuilder<ButtonBuilder> {
    return new ActionRowBuilder<ButtonBuilder>().addComponents(
        new ButtonBuilder()
            .setCustomId("pay")
            .setLabel("Upload slip")
            .setStyle(ButtonStyle.Success)
            .setDisabled(disabled),
        new ButtonBuilder()
            .setCustomId("cancel")
            .setLabel("Cancel")
            .setStyle(ButtonStyle.Danger)
    );
}

function buildSlipModal(modalId: string): ModalBuilder {
    return new ModalBuilder()
        .setCustomId(modalId)
        .setTitle("Upload payment slip")
        .addLabelComponents(
            new LabelBuilder()
                .setLabel("Payment slip")
                .setDescription("JPG or PNG, from within the last 30 minutes")
                .setFileUploadComponent(
                    new FileUploadBuilder()
                        .setCustomId(SLIP_FIELD)
                        .setMinValues(1)
                        .setMaxValues(1)
                        .setRequired(true)
                )
        );
}

function selectionPrompt(
    cluster: PayeeCluster,
    selected: Set<number>,
    truncated: boolean
): string {
    let message = `💸 **Paying ${payeeLabel(cluster)}**\n${SLIP_FRESHNESS_NOTE}\n\n`;

    if (truncated) {
        message += `> ℹ️ You have more than ${MAX_SELECT} bills to this account; only the first ${MAX_SELECT} are shown.\n\n`;
    }

    if (selected.size === 0) {
        message += "Select the bills you want to pay.";
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

    message += `\n**Total: ${total} ${currency}**\n\n`;

    const payeeName = cluster.ownerName ? `**${cluster.ownerName}** ` : "";
    message += `Transfer **${total} ${currency}** to ${payeeName}`;
    message += `on **${formatPaymentMethod(cluster.method)} \`${cluster.account}\`**, `;
    message += "then press **Upload slip** to finish.";

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
    message += `Paid by <@${payerId}> to <@${cluster.ownerId}> (${accountLabel(cluster)})\n\n`;

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
                            `${formatPaymentMethod(cluster.method)} ${cluster.account} · ` +
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
            time: PAYEE_TIMEOUT,
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

/**
 * Opens the slip modal on the button interaction and waits for it. Returns null
 * if the user dismissed it, let it expire, or picked a non-image — in every case
 * the selection message is left intact so they can simply press the button again.
 *
 * The returned submit interaction carries a fresh 15-minute token, which is what
 * the rest of the flow answers on: the original command's token is often nearly
 * spent by the time someone gets back from their banking app.
 */
async function collectSlipViaModal(
    button: ButtonInteraction
): Promise<{ submit: ModalSubmitInteraction; slip: Attachment } | null> {
    const modalId = `payall_slip:${button.id}`;
    await button.showModal(buildSlipModal(modalId));

    let submit: ModalSubmitInteraction;
    try {
        submit = await button.awaitModalSubmit({
            time: MODAL_TIMEOUT,
            filter: (m) => m.customId === modalId && m.user.id === button.user.id,
        });
    } catch {
        return null;
    }

    const slip = submit.fields.getField(SLIP_FIELD, ComponentType.FileUpload).attachments.first();
    if (!slip || !isImage(slip)) {
        await submit.reply({ content: NOT_AN_IMAGE, flags: MessageFlags.Ephemeral }).catch(() => {});
        return null;
    }

    await submit.deferUpdate();
    return { submit, slip };
}

/** Where the outcome of a payment is written back to. */
interface Responder {
    edit(content: string): Promise<unknown>;
    followUp(content: string): Promise<unknown>;
}

async function settle(
    cluster: PayeeCluster,
    billIds: number[],
    slip: Attachment,
    guildId: string,
    payerId: string,
    respond: Responder
): Promise<void> {
    let res: PayMultipleResponse;

    // Only the payment call decides success or failure. Reporting is handled
    // separately below: once the money is recorded, a failure to render the
    // receipt must never be shown as a failed payment, or the user will pay
    // twice.
    try {
        res = await backend.bill.PayMultiple({
            user_id: payerId,
            guild_id: guildId,
            bill_ids: billIds,
            proof_url: slip.url,
        });
    } catch (err: any) {
        console.error("Pay All Error:", err);
        const errorMessage = toUserMessage(
            err,
            PAYMENT_ERROR_RULES,
            "❌ Failed to verify payment. Please try again later."
        );
        await respond.edit(`**Failed to submit payment.**\n> ${errorMessage}`).catch(() => {});
        return;
    }

    // The selection flow was ephemeral and cannot be made public, so the receipt
    // goes out as a separate public follow-up: the group owner needs to see it,
    // and it serves as the payment log.
    await respond.edit("✅ Payment verified — receipt posted below.").catch(() => {});
    await respond
        .followUp(buildResultMessage(res, cluster, slip.url, payerId))
        .catch((err) => console.error("Pay All (receipt) Error:", err));
}

export async function executePayAll(interaction: ChatInputCommandInteraction) {
    const guildId = await requireGuild(interaction);
    if (!guildId) return;

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
    await attachOwnerNames(interaction, clusters);

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

    const rowsFor = (sel: Set<number>) => [
        buildBillSelect(cluster, sel),
        buildActionRow(sel.size === 0),
    ];

    const message = await interaction.editReply({
        content: selectionPrompt(cluster, selected, truncated),
        components: rowsFor(selected),
    });

    // A collector rather than awaitMessageComponent: the user must be able to
    // adjust their selection and watch the total before paying.
    const collector = message.createMessageComponentCollector({ time: SELECT_TIMEOUT });

    // Guards against a second click while a modal is open or a payment is in
    // flight, which would otherwise submit the same bills twice.
    let busy = false;

    collector.on("collect", async (i) => {
        if (i.user.id !== interaction.user.id) {
            await i.reply({ content: "This isn't your payment.", flags: MessageFlags.Ephemeral });
            return;
        }

        if (busy) {
            await i.deferUpdate().catch(() => {});
            return;
        }

        if (i.isStringSelectMenu()) {
            selected.clear();
            for (const value of i.values) selected.add(Number(value));

            await i.update({
                content: selectionPrompt(cluster, selected, truncated),
                components: rowsFor(selected),
            });
            return;
        }

        if (i.customId === "cancel") {
            collector.stop("cancel");
            await i.update({ content: "Payment cancelled.", components: [] });
            return;
        }

        if (i.customId !== "pay" || !i.isButton() || selected.size === 0) {
            await i.deferUpdate().catch(() => {});
            return;
        }

        const billIds = [...selected];
        busy = true;

        const collected = await collectSlipViaModal(i);
        if (!collected) {
            // Dismissed, expired or not an image — the selection is still on
            // screen, so let them press the button again.
            busy = false;
            return;
        }

        collector.stop("done");
        const { submit, slip } = collected;
        await submit.editReply({ content: "⏳ **Verifying your slip...**", components: [] });
        await settle(cluster, billIds, slip, guildId, interaction.user.id, {
            edit: (content) => submit.editReply({ content, components: [] }),
            followUp: (content) => submit.followUp({ content }),
        });
    });

    collector.on("end", async (_collected, reason) => {
        if (reason === "done" || reason === "cancel") return;
        // A modal is still open or a payment is in flight; that path owns the
        // message and will write the real outcome to it.
        if (busy) return;

        // Leave the totals up — they are still worth reading — and point at the
        // recovery path in case the money has already been transferred.
        const content =
            selectionPrompt(cluster, selected, truncated) +
            "\n\n⏳ This payment timed out. Run `/bill payall` again to finish it — " +
            "if you have already transferred, re-select these bills and upload the slip.";

        await interaction.editReply({ content, components: [] }).catch(() => {});
    });
}

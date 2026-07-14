import {
    ChatInputCommandInteraction,
    AutocompleteInteraction,
    MessageFlags
} from "discord.js";
import { backend } from "../../utils/backend.js";
import { toUserMessage, PAYMENT_ERROR_RULES } from "../../utils/errors.js";
import { formatPaymentMethod, maskAccount, formatDate } from "../../utils/format.js";
import { BillResponse } from "../../api/types.js";

interface GroupInfo {
    name: string;
    due_day: number;
    payment_method: string;
    payment_account: string;
}

/** Fetches display info for each group referenced by the given bills. */
async function buildGroupInfoMap(bills: BillResponse[]): Promise<Map<number, GroupInfo>> {
    const uniqueGroupIds = [...new Set(bills.map((bill) => bill.group_id))];
    const groupsMap = new Map<number, GroupInfo>();

    await Promise.all(
        uniqueGroupIds.map(async (groupId) => {
            try {
                const group = await backend.group.get(groupId);
                groupsMap.set(groupId, {
                    name: group.name,
                    due_day: group.due_day,
                    payment_method: group.payment?.method ?? "",
                    payment_account: group.payment?.account ?? "",
                });
            } catch {
                groupsMap.set(groupId, {
                    name: `Group #${groupId}`,
                    due_day: 0,
                    payment_method: "",
                    payment_account: "",
                });
            }
        })
    );

    return groupsMap;
}

/** Builds a "Method ****1234" hint from a payment method + account. */
function formatPaymentHint(method: string, account: string): string {
    if (!method && !account) return "";
    return `${formatPaymentMethod(method)} ****${maskAccount(account)}`;
}

/** Turns a bill (enriched with group info) into an autocomplete choice. */
function billToChoice(bill: BillResponse, group: GroupInfo) {
    const dueDayText = group.due_day ? `Due: Day ${group.due_day}` : "";
    const paymentInfo = formatPaymentHint(group.payment_method, group.payment_account);

    let name = `${group.name} - ${bill.amount_due} ${bill.currency}`;
    if (dueDayText) name += ` (${dueDayText})`;
    if (paymentInfo) name += ` [${paymentInfo}]`;

    return {
        name: name.length > 100 ? name.substring(0, 97) + "..." : name,
        value: String(bill.id),
    };
}

export async function autocompletePaid(interaction: AutocompleteInteraction) {
    const guildId = interaction.guildId;
    if (!guildId) {
        await interaction.respond([]);
        return;
    }
    if (interaction.responded) return;

    try {
        const bills = await backend.bill.GetUnpaidByUserAndGuild(interaction.user.id, guildId);

        if (!bills || bills.length === 0) {
            await interaction.respond([]);
            return;
        }

        const groupsMap = await buildGroupInfoMap(bills);
        const searchTerm = interaction.options.getFocused().toLowerCase();

        const fallbackGroup: GroupInfo = { name: "", due_day: 0, payment_method: "", payment_account: "" };

        const choices = bills
            .map((bill) => {
                const group = groupsMap.get(bill.group_id) ?? {
                    ...fallbackGroup,
                    name: `Group #${bill.group_id}`,
                };
                return { bill, group };
            })
            .filter(({ bill, group }) =>
                group.name.toLowerCase().includes(searchTerm) ||
                String(bill.id).includes(searchTerm) ||
                bill.description?.toLowerCase().includes(searchTerm)
            )
            .slice(0, 25)
            .map(({ bill, group }) => billToChoice(bill, group));

        await interaction.respond(choices);
    } catch (err) {
        console.error("Autocomplete Error:", err);
        if (!interaction.responded) {
            await interaction.respond([]).catch(() => {});
        }
    }
}

/** Builds the success message shown after a payment is verified. */
function buildPaymentResultMessage(
    paidBill: BillResponse,
    groupName: string,
    remainingBill: BillResponse | null,
    slipUrl: string
): string {
    let message = `✅ **Payment Verified Successfully!**\n\n`;
    message += `📋 **Bill Information:**\n`;
    message += `> **Group:** ${groupName}\n`;
    message += `> **Bill ID:** \`${paidBill.id}\`\n`;
    message += `> **Amount Due:** ${paidBill.amount_due} ${paidBill.currency}\n`;
    message += `> **Status:** ${(paidBill.status || "").toUpperCase()}\n`;
    if (paidBill.description) {
        message += `> **Description:** ${paidBill.description}\n`;
    }
    message += `> **Period:** ${paidBill.month}/${paidBill.year}\n`;
    message += `> **Submitted At:** ${formatDate(paidBill.submitted_at)}\n`;
    message += `> **Verified At:** ${formatDate(paidBill.verified_at)}\n`;
    message += `> [View Payment Slip](${slipUrl})\n`;

    if (remainingBill) {
        message += `\n⚠️ **Underpayment Detected!**\n`;
        message += `> A new bill has been created for the remaining balance:\n`;
        message += `> **New Bill ID:** \`${remainingBill.id}\`\n`;
        message += `> **Remaining Amount:** ${remainingBill.amount_due} ${remainingBill.currency}\n`;
        message += `> **Status:** ${(remainingBill.status || "").toUpperCase()}\n`;
        message += `> Please pay the remaining balance to complete your payment.`;
    }

    return message;
}

export async function executePaid(interaction: ChatInputCommandInteraction) {
    const guildId = interaction.guildId;
    const billId = interaction.options.getString("target_bill", true);
    const slip = interaction.options.getAttachment("slip", true);

    // Validate inputs before deferring so we can reply ephemerally.
    if (!guildId) {
        await interaction.reply({
            content: "This command can only be used inside a server (not in DMs).",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    if (!slip.contentType?.startsWith("image/")) {
        await interaction.reply({
            content: "❌ Please upload a valid image file (JPG/PNG).",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    const billIdNum = parseInt(billId, 10);
    if (isNaN(billIdNum)) {
        await interaction.reply({
            content: "❌ Invalid bill ID. Please select a bill from the dropdown.",
            flags: MessageFlags.Ephemeral,
        });
        return;
    }

    await interaction.deferReply();
    await interaction.editReply({
        content: "⏳ **Processing payment...**\n> ⚠️ **Important:** Payment slips must be uploaded within **30 minutes** of making the transfer. Please ensure your slip is recent."
    });

    try {
        const paidBill = await backend.bill.Pay({
            user_id: interaction.user.id,
            guild_id: guildId,
            bill_id: billIdNum,
            proof_url: slip.url,
        });

        const groupName = await fetchGroupName(paidBill.group_id);
        const remainingBill = await findRemainingBill(interaction.user.id, guildId, paidBill);

        await interaction.editReply({
            content: buildPaymentResultMessage(paidBill, groupName, remainingBill, slip.url),
        });
    } catch (err: any) {
        console.error("Payment Submission Error:", err);

        const errorMessage = toUserMessage(err, PAYMENT_ERROR_RULES, "❌ Failed to verify payment. Please try again later.");

        await interaction.deleteReply().catch((deleteErr) => {
            console.error("Could not delete reply:", deleteErr);
        });

        await interaction.followUp({
            content: `**Failed to submit payment.**\n> ${errorMessage}`,
            flags: MessageFlags.Ephemeral,
        });
    }
}

/** Best-effort group name lookup; never throws. */
async function fetchGroupName(groupId: number): Promise<string> {
    try {
        const group = await backend.group.get(groupId);
        return group.name;
    } catch (err) {
        console.error("Failed to fetch group info:", err);
        return "Unknown Group";
    }
}

/** Finds a newly-created bill for the same period (underpayment); never throws. */
async function findRemainingBill(
    userId: string,
    guildId: string,
    paidBill: BillResponse
): Promise<BillResponse | null> {
    try {
        const unpaidBills = await backend.bill.GetUnpaidByUserAndGuild(userId, guildId);
        return (
            unpaidBills?.find(
                (bill) =>
                    bill.group_id === paidBill.group_id &&
                    bill.month === paidBill.month &&
                    bill.year === paidBill.year &&
                    bill.id !== paidBill.id
            ) ?? null
        );
    } catch (err) {
        console.error("Failed to check for remaining bills:", err);
        return null;
    }
}

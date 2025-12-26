import { 
    ChatInputCommandInteraction, 
    AutocompleteInteraction, 
    MessageFlags 
} from "discord.js";
import { BackendClient } from "../../api/index.js";
import { config } from "../../config.js";

const backend = new BackendClient({
    baseUrl: config.BACKEND_BASE_URL || "http://localhost:8000",
    apiKey: config.BACKEND_API_KEY
});

export async function autocompletePaid(interaction: AutocompleteInteraction) {
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
        const bills = await backend.bill.GetUnpaidByUserAndGuild(userId, guildId);

        if (!bills || bills.length === 0) {
            await interaction.respond([]);
            return;
        }

        const uniqueGroupIds = [...new Set(bills.map((bill: any) => bill.group_id))];
        const groupsMap = new Map<number, { name: string; due_day: number; payment_method: string; payment_account: string }>();
        
        await Promise.all(
            uniqueGroupIds.map(async (groupId: number) => {
                try {
                    const group = await backend.group.get(groupId);
                    groupsMap.set(groupId, {
                        name: group.name,
                        due_day: group.due_day,
                        payment_method: group.payment?.method ?? "",
                        payment_account: group.payment?.account ?? "",
                    });
                } catch (err) {
                    groupsMap.set(groupId, {
                        name: `Group #${groupId}`,
                        due_day: 0,
                        payment_method: "",
                        payment_account: "",
                    });
                }
            })
        );

        const billsWithGroupInfo = bills.map((bill: any) => {
            const groupInfo =
                groupsMap.get(bill.group_id) || {
                    name: `Group #${bill.group_id}`,
                    due_day: 0,
                    payment_method: "",
                    payment_account: "",
                };
            return {
                ...bill,
                group_name: groupInfo.name,
                group_due_day: groupInfo.due_day,
                payment_method: groupInfo.payment_method,
                payment_account: groupInfo.payment_account,
            };
        });

        const searchTerm = focusedValue.toLowerCase();
        const filtered = billsWithGroupInfo.filter((bill: any) => 
            bill.group_name.toLowerCase().includes(searchTerm) ||
            String(bill.id).includes(searchTerm) ||
            bill.description?.toLowerCase().includes(searchTerm)
        );

        const formatPaymentInfo = (method: string, account: string) => {
            if (!method && !account) return "";

            const cleanAcc = account ? account.replace(/\D/g, "") : "";
            const last4 = cleanAcc.slice(-4) || "????";

            const lower = (method || "").toLowerCase();
            let prettyMethod = method || "Bank";
            if (lower === "promptpay" || lower === "msisdn") prettyMethod = "PromptPay";
            if (lower === "bank_account" || lower === "bank" || lower === "bankac") prettyMethod = "Bank";

            return `${prettyMethod} ****${last4}`;
        };

        const choices = filtered.slice(0, 25).map((bill: any) => {
            const dueDayText = bill.group_due_day ? `Due: Day ${bill.group_due_day}` : "";
            const paymentInfo = formatPaymentInfo(bill.payment_method, bill.payment_account);

            let name = `${bill.group_name} - ${bill.amount_due} ${bill.currency}`;
            if (dueDayText) name += ` (${dueDayText})`;
            if (paymentInfo) name += ` [${paymentInfo}]`;

            return {
                name: name.length > 100 ? name.substring(0, 97) + "..." : name,
                value: String(bill.id),
            };
        });

        await interaction.respond(choices);

    } catch (err: any) {
        console.error("Autocomplete Error:", err);
        if (!interaction.responded) {
            try {
                await interaction.respond([]);
            } catch (respondErr) {
                console.error("Failed to respond to autocomplete:", respondErr);
            }
        }
    }
}

export async function executePaid(interaction: ChatInputCommandInteraction) {
    await interaction.deferReply();

    try {
        const billId = interaction.options.getString("target_bill", true);
        const slip = interaction.options.getAttachment("slip", true);
        const guildId = interaction.guildId

        if (!guildId) {
            await interaction.deleteReply();
            await interaction.followUp({
                content: "This command can only be used inside a server (not in DMs).",
                flags: MessageFlags.Ephemeral
            });
            return;
        }

        if (!slip.contentType?.startsWith("image/")) {
            await interaction.deleteReply();
            await interaction.followUp({
                content: "❌ Please upload a valid image file (JPG/PNG).",
                flags: MessageFlags.Ephemeral
            });
            return;
        }

        await interaction.editReply({
            content: "⏳ **Processing payment...**\n> ⚠️ **Important:** Payment slips must be uploaded within **30 minutes** of making the transfer. Please ensure your slip is recent."
        });

        const billIdNum = parseInt(billId, 10);
        if (isNaN(billIdNum)) {
            await interaction.deleteReply();
            await interaction.followUp({
                content: "❌ Invalid bill ID. Please select a bill from the dropdown.",
                flags: MessageFlags.Ephemeral
            });
            return;
        }

        const paidBill = await backend.bill.Pay({
            user_id: interaction.user.id,
            guild_id: guildId,
            bill_id: billIdNum,
            proof_url: slip.url,
        });

        let groupName = "Unknown Group";
        try {
            const group = await backend.group.get(paidBill.group_id);
            groupName = group.name;
        } catch (err) {
            console.error("Failed to fetch group info:", err);
        }

        let remainingBill = null;
        try {
            const unpaidBills = await backend.bill.GetUnpaidByUserAndGuild(interaction.user.id, guildId);
            if (unpaidBills && unpaidBills.length > 0) {
                remainingBill = unpaidBills.find((bill: any) => 
                    bill.group_id === paidBill.group_id && 
                    bill.month === paidBill.month && 
                    bill.year === paidBill.year &&
                    bill.id !== paidBill.id
                );
            }
        } catch (err) {
            console.error("Failed to check for remaining bills:", err);
        }

        const formatDate = (dateStr: string | null | undefined) => {
            if (!dateStr) return "N/A";
            try {
                const date = new Date(dateStr);
                return date.toLocaleDateString("en-US", { 
                    year: "numeric", 
                    month: "short", 
                    day: "numeric",
                    hour: "2-digit",
                    minute: "2-digit"
                });
            } catch {
                return dateStr;
            }
        };

        let message = `✅ **Payment Verified Successfully!**\n\n`;
        message += `📋 **Bill Information:**\n`;
        message += `> **Group:** ${groupName}\n`;
        message += `> **Bill ID:** \`${paidBill.id}\`\n`;
        message += `> **Amount Due:** ${paidBill.amount_due} ${paidBill.currency}\n`;
        message += `> **Status:** ${paidBill.status.toUpperCase()}\n`;
        if (paidBill.description) {
            message += `> **Description:** ${paidBill.description}\n`;
        }
        message += `> **Period:** ${paidBill.month}/${paidBill.year}\n`;
        message += `> **Submitted At:** ${formatDate(paidBill.submitted_at)}\n`;
        message += `> **Verified At:** ${formatDate(paidBill.verified_at)}\n`;
        message += `> [View Payment Slip](${slip.url})\n`;

        if (remainingBill) {
            message += `\n⚠️ **Underpayment Detected!**\n`;
            message += `> A new bill has been created for the remaining balance:\n`;
            message += `> **New Bill ID:** \`${remainingBill.id}\`\n`;
            message += `> **Remaining Amount:** ${remainingBill.amount_due} ${remainingBill.currency}\n`;
            message += `> **Status:** ${remainingBill.status.toUpperCase()}\n`;
            message += `> Please pay the remaining balance to complete your payment.`;
        }

        await interaction.editReply({
            content: message
        });

    } catch (err: any) {
        console.error("Payment Submission Error:", err);

        let errorMessage = "Unknown error";
        const errorBody = err?.response?.data?.error || err?.response?.data?.message || err?.message || "";

        if (errorBody.includes("bill not found") || err?.response?.status === 404) {
            errorMessage = "❌ Bill not found. Please select a valid bill.";
        } else if (errorBody.includes("bill does not belong to this user") || err?.response?.status === 403) {
            errorMessage = "❌ This bill does not belong to you. You cannot pay someone else's bill.";
        } else if (errorBody.includes("payment receiver does not match")) {
            errorMessage = "❌ Payment receiver does not match group settings. Please check the payment details.";
        } else if (errorBody.includes("proof url is required")) {
            errorMessage = "❌ Proof URL is required. Please upload a valid slip image.";
        } else if (errorBody.includes("member not found in group")) {
            errorMessage = "❌ You are not a member of this group.";
        } else if (errorBody.includes("bill is already paid")) {
            errorMessage = "❌ This bill has already been paid.";
        } else if (errorBody.includes("payment slip is too old") || errorBody.includes("within 30 minutes")) {
            errorMessage = "❌ Payment slip is too old. Please upload the slip within 30 minutes of making the transfer.";
        } else if (errorBody.includes("No valid QR code")) {
            errorMessage = "❌ No valid QR code found in the image. Please ensure the payment slip contains a clear QR code.";
        } else if (errorBody.includes("slip QR code has expired") || errorBody.includes("expired or is invalid")) {
            errorMessage = "❌ The slip QR code has expired or is invalid. Please use a recent slip.";
        } else if (errorBody.includes("slip has already been used") || err?.response?.status === 409) {
            errorMessage = "❌ This slip has already been used. Please use a different payment slip.";
        } else if (errorBody.includes("amount in the slip does not match")) {
            errorMessage = "❌ The amount in the slip does not match the bill amount. Please verify the payment details.";
        } else if (errorBody.includes("Verification timed out") || err?.response?.status === 504) {
            errorMessage = "❌ Verification timed out. Please try again later.";
        } else {
            errorMessage = `❌ Failed to verify payment: ${errorBody || "Unknown error"}`;
        }
        
        try {
            await interaction.deleteReply();
        } catch (deleteErr) {
            console.error("Could not delete reply:", deleteErr);
        }
        
        await interaction.followUp({
            content: `**Failed to submit payment.**\n> ${errorMessage}`,
            flags: MessageFlags.Ephemeral
        });
    }
}
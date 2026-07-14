import cron from "node-cron";
import { Client, EmbedBuilder, ChannelType, TextChannel, Guild, GuildMember } from "discord.js";
import { backend } from "../utils/backend.js";
import { BillResponse } from "../api/types.js";

export async function checkUnpaidBills(client: Client): Promise<number> {
    console.log("⏰ Running unpaid bill check (Channel Mode)...");

    try {
        const today = new Date().getDate();
        const guilds = await client.guilds.fetch();

        let totalSent = 0;

        for (const [guildId] of guilds) {
            try {
                totalSent += await remindGuild(client, guildId, today);
            } catch (e) {
                console.error(`Failed processing guild ${guildId}`, e);
            }
        }

        return totalSent;
    } catch (err) {
        console.error("❌ Critical error in bill checker:", err);
        return 0;
    }
}

/** Sends reminders for all unpaid bills in a single guild. Returns count sent. */
async function remindGuild(client: Client, guildId: string, today: number): Promise<number> {
    const guild = await client.guilds.fetch(guildId);
    const targetChannel = await findPaymentChannel(guild);

    if (!targetChannel) {
        console.warn(`[${guild.name}] Skipped: No suitable channel found.`);
        return 0;
    }

    const unpaidBills = await backend.bill.GetUnpaidByGuild(guildId);
    if (!unpaidBills || unpaidBills.length === 0) return 0;

    console.log(`[${guild.name}] Found ${unpaidBills.length} unpaid bills.`);

    const groupNameCache = new Map<number, string>();
    let sent = 0;

    for (const bill of unpaidBills) {
        try {
            const groupName = await resolveGroupName(bill.group_id, groupNameCache);

            await targetChannel.send({
                content: `<@${bill.member_id}>`, // This triggers the PING
                embeds: [buildBillReminderEmbed(bill, groupName, today)],
            });

            sent++;
            await new Promise((r) => setTimeout(r, 1000));
        } catch (e) {
            console.error(`Could not process bill for user ${bill.member_id}`, e);
        }
    }

    return sent;
}

async function resolveGroupName(groupId: number, cache: Map<number, string>): Promise<string> {
    const cached = cache.get(groupId);
    if (cached) return cached;

    const group = await backend.group.get(groupId);
    cache.set(groupId, group.name);
    return group.name;
}

function buildBillReminderEmbed(bill: BillResponse, groupName: string, today: number): EmbedBuilder {
    return new EmbedBuilder()
        .setTitle("⚠️ Payment Due Reminder")
        .setDescription(`Hello <@${bill.member_id}>! \n\nYour payment for **${groupName}** is currently **Pending**.`)
        .setColor("Red")
        .addFields(
            { name: "Amount Due", value: `${bill.amount_due} THB`, inline: true },
            { name: "Billing Cycle", value: `${bill.month}/${bill.year}`, inline: true },
            { name: "Due Date", value: `Day ${today}`, inline: true }
        )
        .setFooter({ text: "Use /bill paid to submit your slip" });
}

async function findPaymentChannel(guild: Guild): Promise<TextChannel | null> {
    await guild.channels.fetch();

    const botMember = guild.members.me;
    if (!botMember) {
        console.warn(`[${guild.name}] Bot member not available; skipping.`);
        return null;
    }

    const canSend = (channel: { permissionsFor(member: GuildMember): { has(perm: "SendMessages"): boolean } | null }) =>
        channel.permissionsFor(botMember)?.has("SendMessages");

    const paymentChannel = guild.channels.cache.find(
        (c) =>
            c.name.toLowerCase() === "payment" &&
            c.type === ChannelType.GuildText &&
            canSend(c)
    ) as TextChannel;

    if (paymentChannel) {
        console.log(`[${guild.name}] Found payment channel: #${paymentChannel.name}`);
        return paymentChannel;
    }

    if (guild.systemChannel && canSend(guild.systemChannel)) {
        console.log(`[${guild.name}] Using system channel: #${guild.systemChannel.name}`);
        return guild.systemChannel;
    }

    const fallbackChannel = (guild.channels.cache.find(
        (c) => c.type === ChannelType.GuildText && canSend(c)
    ) as TextChannel) || null;

    if (fallbackChannel) {
        console.log(`[${guild.name}] Using fallback channel: #${fallbackChannel.name}`);
    }

    return fallbackChannel;
}

export function initDailyReminders(_client: Client) {
    cron.schedule("0 8 * * *", () => {
        checkUnpaidBills(_client).catch((err) =>
            console.error("Scheduled bill check failed:", err)
        );
    }, {
        timezone: "Asia/Bangkok"
    });
    console.log("✅ Daily Reminder Service Started");
}

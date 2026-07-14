import cron from "node-cron";
import { Client, ChannelType, TextChannel, Guild, GuildMember } from "discord.js";
import { backend } from "../utils/backend.js";

/**
 * Mentions per message. Discord caps a message at 2000 characters and a mention
 * is roughly 22, so this stays well clear of the limit.
 */
const MENTIONS_PER_MESSAGE = 50;

/** Monday 08:00 — a single weekly nudge rather than a daily one. */
const WEEKLY_SCHEDULE = "0 8 * * 1";

/**
 * Reminds everyone with an unpaid bill, one message per guild.
 *
 * Deliberately says nothing about amounts or groups: people can run
 * `/bill payall` to see their own totals. Returns the number of members tagged.
 */
export async function checkUnpaidBills(client: Client): Promise<number> {
    console.log("⏰ Running unpaid bill check...");

    try {
        const guilds = await client.guilds.fetch();

        let totalTagged = 0;

        for (const [guildId] of guilds) {
            try {
                totalTagged += await remindGuild(client, guildId);
            } catch (e) {
                console.error(`Failed processing guild ${guildId}`, e);
            }
        }

        return totalTagged;
    } catch (err) {
        console.error("❌ Critical error in bill checker:", err);
        return 0;
    }
}

/** Reminds one guild's debtors. Returns how many members were tagged. */
async function remindGuild(client: Client, guildId: string): Promise<number> {
    const guild = await client.guilds.fetch(guildId);
    const targetChannel = await findPaymentChannel(guild);

    if (!targetChannel) {
        console.warn(`[${guild.name}] Skipped: No suitable channel found.`);
        return 0;
    }

    const unpaidBills = await backend.bill.GetUnpaidByGuild(guildId);
    if (!unpaidBills || unpaidBills.length === 0) return 0;

    // One tag per person, however many bills they owe.
    const debtors = [...new Set(unpaidBills.map((bill) => bill.member_id))];

    // Someone who has left the server would otherwise render as a dead <@id>.
    const members = await guild.members.fetch();
    const present = debtors.filter((memberId) => members.has(memberId));

    if (present.length === 0) {
        console.log(`[${guild.name}] ${debtors.length} debtor(s), none still in the server.`);
        return 0;
    }

    console.log(
        `[${guild.name}] ${unpaidBills.length} unpaid bill(s) across ${present.length} member(s).`
    );

    for (let i = 0; i < present.length; i += MENTIONS_PER_MESSAGE) {
        const mentions = present
            .slice(i, i + MENTIONS_PER_MESSAGE)
            .map((memberId) => `<@${memberId}>`)
            .join(" ");

        await targetChannel.send({
            content: `⏰ ${mentions}\nYou have unpaid bills. Run \`/bill payall\` to see what you owe and pay.`,
        });
    }

    return present.length;
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

export function initWeeklyReminders(client: Client) {
    cron.schedule(WEEKLY_SCHEDULE, () => {
        checkUnpaidBills(client).catch((err) =>
            console.error("Scheduled bill check failed:", err)
        );
    }, {
        timezone: "Asia/Bangkok"
    });
    console.log("✅ Weekly Reminder Service Started (Mondays, 08:00 Asia/Bangkok)");
}

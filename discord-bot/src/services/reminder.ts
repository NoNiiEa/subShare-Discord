import cron from "node-cron";
import { Client, EmbedBuilder, ChannelType, TextChannel, Guild } from "discord.js";
import { BackendClient } from "../api/index.js";
import { config } from "../config.js";

const backend = new BackendClient({
    baseUrl: config.BACKEND_BASE_URL || "http://localhost:8000",
    apiKey: config.BACKEND_API_KEY
});

export async function checkUnpaidBills(client: Client) {
    console.log("⏰ Running unpaid bill check (Channel Mode)...");

    try {
        const today = new Date().getDate();
        const guilds = await client.guilds.fetch();
        
        let totalSent = 0;

        for (const [guildId, _] of guilds) {
            try {
                const guild = await client.guilds.fetch(guildId);
                
                const targetChannel = await findPaymentChannel(guild);

                if (!targetChannel) {
                    console.warn(`[${guild.name}] Skipped: No suitable channel found.`);
                    continue;
                }

                const unpaidBills = await backend.bill.GetUnpaidByGuild(guildId);
                
                if (!unpaidBills || unpaidBills.length === 0) continue;

                console.log(`[${guild.name}] Found ${unpaidBills.length} unpaid bills.`);

                const groupNameCache = new Map<number, string>();

                for (const bill of unpaidBills) {
                    try {
                        let groupName = groupNameCache.get(bill.group_id);
                        
                        if (!groupName) {
                            const group = await backend.group.get(bill.group_id);
                            groupName = group.name;
                            groupNameCache.set(bill.group_id, groupName);
                        }

                        const embed = new EmbedBuilder()
                            .setTitle("⚠️ Payment Due Reminder")
                            .setDescription(`Hello <@${bill.member_id}>! \n\nYour payment for **${groupName}** is currently **Pending**.`)
                            .setColor("Red")
                            .addFields(
                                { name: "Amount Due", value: `${bill.amount_due} THB`, inline: true },
                                { name: "Billing Cycle", value: `${bill.month}/${bill.year}`, inline: true },
                                { name: "Due Date", value: `Day ${today}`, inline: true } 
                            )
                            .setFooter({ text: "Use /bill paid to submit your slip" });

                        await targetChannel.send({ 
                            content: `<@${bill.member_id}>`, // This triggers the PING
                            embeds: [embed] 
                        });
                        
                        totalSent++;
                        
                        await new Promise(r => setTimeout(r, 1000)); 

                    } catch (e) {
                        console.error(`Could not process bill for user ${bill.member_id}`, e);
                    }
                }
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

async function findPaymentChannel(guild: Guild): Promise<TextChannel | null> {
    await guild.channels.fetch();
    
    const paymentChannel = guild.channels.cache.find(
        (c) => c.name.toLowerCase() === "payment" && 
               c.type === ChannelType.GuildText &&
               c.permissionsFor(guild.members.me!)?.has("SendMessages")
    ) as TextChannel;
    
    if (paymentChannel) {
        console.log(`[${guild.name}] Found payment channel: #${paymentChannel.name}`);
        return paymentChannel;
    }

    if (guild.systemChannel && 
        guild.systemChannel.permissionsFor(guild.members.me!)?.has("SendMessages")) {
        console.log(`[${guild.name}] Using system channel: #${guild.systemChannel.name}`);
        return guild.systemChannel;
    }

    const fallbackChannel = guild.channels.cache.find(
        (c) => c.type === ChannelType.GuildText && 
               c.permissionsFor(guild.members.me!)?.has("SendMessages")
    ) as TextChannel || null;
    
    if (fallbackChannel) {
        console.log(`[${guild.name}] Using fallback channel: #${fallbackChannel.name}`);
    }
    
    return fallbackChannel;
}

export function initDailyReminders(client: Client) {
    cron.schedule("0 8 * * *", () => {
        checkUnpaidBills(client);
    }, {
        timezone: "Asia/Bangkok"
    });
    console.log("✅ Daily Reminder Service Started");
}
import cron from "node-cron";
import { Client, EmbedBuilder, ChannelType, TextChannel } from "discord.js";
import { BackendClient } from "../api/index.js";
import { config } from "../config.js";

const backend = new BackendClient({
    baseUrl: config.BACKEND_BASE_URL || "http://localhost:8000",
    apiKey: config.BACKEND_API_KEY
});

export async function checkUnpaidBills(client: Client) {
    console.log("⏰ Running unpaid bill check (Manual/Auto)...");

    try {
        const today = new Date().getDate();
        // Fetch fresh guild data to ensure cache is up to date
        const guilds = await client.guilds.fetch();
        
        let totalSent = 0;

        for (const [guildId, _] of guilds) {
            try {
                // 1. Get real guild object (needed for channels)
                const guild = await client.guilds.fetch(guildId);

                // 2. Find a channel to send to
                // Priority: System Channel -> A channel named "payments" -> First Sendable Channel
                let targetChannel = guild.systemChannel;
                
                if (!targetChannel) {
                    targetChannel = guild.channels.cache.find(
                        (c) => c.name.includes("payment") && c.type === ChannelType.GuildText
                    ) as TextChannel;
                }

                if (!targetChannel) {
                    // Fallback: Find first text channel we can write to
                    targetChannel = guild.channels.cache.find(
                        (c) => c.type === ChannelType.GuildText && 
                               c.permissionsFor(guild.members.me!)?.has("SendMessages")
                    ) as TextChannel;
                }

                if (!targetChannel) {
                    console.warn(`[${guild.name}] No suitable channel found to send reminders.`);
                    continue;
                }

                // 3. Get Bills
                const unpaidBills = await backend.bill.GetUnpaidByGuild(guildId);
                if (!unpaidBills || unpaidBills.length === 0) continue;

                console.log(`[${guild.name}] Found ${unpaidBills.length} unpaid bills. Sending to #${targetChannel.name}`);

                for (const bill of unpaidBills) {
                    try {
                        const group = await backend.group.get(bill.group_id);
                        
                        const embed = new EmbedBuilder()
                            .setTitle("⚠️ Payment Due Reminder")
                            .setDescription(`Hello <@${bill.member_id}>! \n\nYour payment for **${group.name}** is currently **Pending**.`)
                            .setColor("Red")
                            .addFields(
                                { name: "Amount Due", value: `${bill.amount_due} THB`, inline: true },
                                { name: "Due Date", value: `Day ${today}`, inline: true },
                                { name: "Cycle", value: `${bill.month}/${bill.year}`, inline: true }
                            )
                            .setFooter({ text: "Please submit your payment slip using /pay" });

                        // Send to Channel instead of DM
                        // We add 'content' to ensure the user gets a push notification (Ping)
                        await targetChannel.send({ 
                            content: `<@${bill.member_id}>`, 
                            embeds: [embed] 
                        });
                        
                        totalSent++;
                        
                        // Small delay to prevent rate limits if many bills
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

export function initDailyReminders(client: Client) {
    cron.schedule("0 20 * * *", () => {
        checkUnpaidBills(client);
    }, {
        timezone: "Asia/Bangkok"
    });
    console.log("✅ Daily Reminder Service Started");
}
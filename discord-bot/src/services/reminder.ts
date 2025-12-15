import cron from "node-cron";
import { Client, EmbedBuilder } from "discord.js";
import { BackendClient } from "../api/index.js";
import { config } from "../config.js";

const backend = new BackendClient({
    baseUrl: config.BACKEND_BASE_URL || "http://localhost:8000",
});

export async function checkUnpaidBills(client: Client) {
    console.log("⏰ Running unpaid bill check (Manual/Auto)...");

    try {
        const today = new Date().getDate();
        const guilds = client.guilds.cache;
        
        let totalSent = 0;

        for (const [guildId, guild] of guilds) {
            try {
                const unpaidBills = await backend.bill.GetUnpaidByGuild(guildId)

                if (!unpaidBills || unpaidBills.length === 0) continue;

                console.log(`[${guild.name}] Found ${unpaidBills.length} unpaid bills.`);

                for (const bill of unpaidBills) {
                    try {
                        const user = await client.users.fetch(bill.member_id);
                        const group = await backend.group.get(bill.group_id)
                        
                        const embed = new EmbedBuilder()
                            .setTitle("⚠️ Payment Reminder")
                            .setDescription(`Your payment for **${group.name}** in **${guild.name}** is due!`)
                            .setColor("Red")
                            .addFields(
                                { name: "Amount", value: `${bill.amount_due} THB`, inline: true },
                                { name: "Due Date", value: `Day ${today}`, inline: true }
                            )
                            .setFooter({ text: "Please pay via the group channel." });

                        await user.send({ embeds: [embed] });
                        totalSent++;
                    } catch (e) {
                        console.error(`Could not DM user ${bill.member_id}`);
                    }
                }
            } catch (e) {
                console.error(`Failed guild ${guild.name}`, e);
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
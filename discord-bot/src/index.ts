import { 
    Client, 
    Events, 
    GatewayIntentBits
} from "discord.js"
import { config } from "./config.js"
import { setupCommandHandler } from "./handler/commadHandler.js"
import { initWeeklyReminders } from "./services/reminder.js";
import { registerCommands } from "./register.js";

const client = new Client({ intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildMembers,] });

client.once(Events.ClientReady, (readyClient) => {
	console.log(`Ready! Logged in as ${readyClient.user.tag}`);

    initWeeklyReminders(client);
});

await registerCommands()

await setupCommandHandler(client)

client.login(config.DISCORD_TOKEN).catch((err) => {
    console.error("Failed to log in to Discord:", err);
    process.exit(1);
});


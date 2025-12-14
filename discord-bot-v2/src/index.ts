import { 
    Client, 
    Events, 
    GatewayIntentBits
} from "discord.js"
import { config } from "./config.js"
import { setupCommandHandler } from "./handler/commadHandler.js"
import { initDailyReminders } from "./services/reminder.js";

const client = new Client({ intents: [GatewayIntentBits.Guilds] });

client.once(Events.ClientReady, (readyClient) => {
	console.log(`Ready! Logged in as ${readyClient.user.tag}`);

    initDailyReminders(client);
});

await setupCommandHandler(client)

client.login(config.DISCORD_TOKEN);


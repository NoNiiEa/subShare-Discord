import { 
    Client, 
    Events, 
    GatewayIntentBits
} from "discord.js"
import { config } from "./config"
import { setupCommandHandler } from "./handler/commadHandler"

const client = new Client({ intents: [GatewayIntentBits.Guilds] });

client.once(Events.ClientReady, (readyClient) => {
	console.log(`Ready! Logged in as ${readyClient.user.tag}`);
});

await setupCommandHandler(client)

client.login(config.DISCORD_TOKEN);


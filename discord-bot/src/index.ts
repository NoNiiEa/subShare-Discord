import "dotenv/config";
import {
  Client,
  GatewayIntentBits,
  Message
} from "discord.js";
import { BackendClient } from "./backendClient.js";

const token = process.env.DISCORD_TOKEN;
const backendBaseUrl = process.env.BACKEND_BASE_URL ?? "http://localhost:8080";

if (!token) {
  throw new Error("DISCORD_TOKEN is not set in environment");
}

const backend = new BackendClient(backendBaseUrl);

const client = new Client({
  intents: [
    GatewayIntentBits.Guilds,
    GatewayIntentBits.GuildMessages,
    GatewayIntentBits.MessageContent 
  ]
});

client.once("clientReady", () => {
  if (!client.user) return;
  console.log(`Logged in as ${client.user.tag}`);
});

client.on("messageCreate", async (message: Message) => {
  // Ignore own messages and other bots
  if (message.author.bot) return;

  const content = message.content.trim();

  if (content.startsWith("!ping")) {
    await handlePingCommand(message);
  }

  // Later:
  // if (content.startsWith("!bill")) { ... }
  // if (content.startsWith("!group")) { ... }
});

async function handlePingCommand(message: Message) {
  try {
    const health = await backend.health();
    const status = health.status || "unknown";

    await message.reply(`✅ Backend status: **${status}**`);
  } catch (err) {
    console.error("Error calling backend /api/health:", err);
    await message.reply("❌ Could not reach backend, it might be down.");
  }
}

async function main() {
  await client.login(token);
}

main().catch((err) => {
  console.error("Bot failed to start:", err);
});

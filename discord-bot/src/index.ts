import "dotenv/config";
import {
  Client,
  GatewayIntentBits,
  Interaction,
  Message
} from "discord.js";
import { BackendClient } from "./backendClient.js";
import path from "path";
import fs from "fs";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

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

const commands = new Map<string, any>();

const commandsPath = path.join(__dirname, "commands");
const commandFiles = fs
  .readdirSync(commandsPath)
  .filter((file) => file.endsWith(".ts") || file.endsWith(".js"));

for (const file of commandFiles) {
  const ext = path.extname(file);         
  const base = path.basename(file, ext);
  const importPath = `./commands/${base}${ext}`;

  const commandModule = await import(importPath);
  commands.set(commandModule.data.name, commandModule);
  console.log(`Loaded command: ${commandModule.data.name}`);
}

client.once("clientReady", () => {
  if (!client.user) return;
  console.log(`Logged in as ${client.user.tag}`);
});

client.on("interactionCreate", async (interaction: Interaction) => {
    if (!interaction.isChatInputCommand()) return;

    const command = commands.get(interaction.commandName);
    if (!command) return;

    try {
        await command.execute(interaction);
    } catch (err) {
        console.error(err);
        if (interaction.replied || interaction.deferred) {
            await interaction.followUp("❌ Error executing command");
        } else {
            await interaction.reply("❌ Error executing command");
        }
    }
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

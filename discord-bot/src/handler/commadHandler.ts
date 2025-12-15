// src/handler/commandHandler.ts
import fs from "fs";
import path from "path";
import { fileURLToPath, pathToFileURL } from "url";
import {
  Client,
  Interaction,
  ChatInputCommandInteraction,
  SlashCommandBuilder,
} from "discord.js";

export interface SlashCommand {
  data: SlashCommandBuilder;
  execute: (
    interaction: ChatInputCommandInteraction
  ) => Promise<unknown> | unknown;
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export async function setupCommandHandler(client: Client) {
  const commandsDir = path.join(__dirname, "..", "commands");
  const commands = await loadCommands(commandsDir);

  client.on("interactionCreate", async (interaction: Interaction) => {
    if (!interaction.isChatInputCommand()) return;

    const command = commands.get(interaction.commandName);
    if (!command) return;

    try {
      await command.execute(interaction);
    } catch (error) {
      console.error(`Error executing /${interaction.commandName}`, error);

      if (interaction.deferred || interaction.replied) {
        await interaction.editReply({
          content: "There was an error while executing this command.",
        });
      } else {
        await interaction.reply({
          content: "There was an error while executing this command.",
          ephemeral: true,
        });
      }
    }
  });

  return commands;
}

export async function loadCommands(
  commandsDir: string
): Promise<Map<string, SlashCommand>> {
  const commands = new Map<string, SlashCommand>();

  const entries = fs.readdirSync(commandsDir, { withFileTypes: true });

  for (const entry of entries) {
    if (!entry.isDirectory()) continue;

    const folderPath = path.join(commandsDir, entry.name);

    const possibleFiles = [
      path.join(folderPath, "index.js"),
      path.join(folderPath, "index.mjs"),
      path.join(folderPath, "index.cjs"),
      path.join(folderPath, "index.ts"),
    ];

    const filePath = possibleFiles.find((p) => fs.existsSync(p));
    if (!filePath) {
      console.warn(
        `[commandHandler] No index file for command folder "${entry.name}", skipping`
      );
      continue;
    }

    const moduleUrl = pathToFileURL(filePath).href;
    const imported = await import(moduleUrl);

    const cmd: Partial<SlashCommand> =
      (imported.default as Partial<SlashCommand>) ?? imported;

    if (!cmd.data || !cmd.execute) {
      console.warn(
        `[commandHandler] Command "${entry.name}" missing "data" or "execute", skipping`
      );
      continue;
    }

    commands.set(cmd.data.name, cmd as SlashCommand);
    console.log(`[commandHandler] Loaded command: /${cmd.data.name}`);
  }

  return commands;
}

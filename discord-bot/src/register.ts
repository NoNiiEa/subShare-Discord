import 'dotenv/config';
import fs from 'fs';
import path from 'path';
import { fileURLToPath, pathToFileURL } from 'url';
import { REST, Routes, SlashCommandBuilder } from 'discord.js';
import { config } from './config.js';

interface SlashCommandFile {
  data: SlashCommandBuilder;
  execute: any;
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export const registerCommands = async () => {
    const token = config.DISCORD_TOKEN;
    const clientId = config.DISCORD_CLIENT_ID;
    const guildId = config.DEVELOPMENT_GUILD_ID || "";

    const commands = [];
    const commandsDir = path.join(__dirname, 'commands');

    const entries = fs.readdirSync(commandsDir, { withFileTypes: true });

    console.log('Loading commands...');

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
            console.warn(`Skipped "${entry.name}": No index file found.`);
            continue;
        }

        const moduleUrl = pathToFileURL(filePath).href;
        const imported = await import(moduleUrl);

        const cmd: Partial<SlashCommandFile> = (imported.default) ?? imported;

        if (cmd.data && cmd.execute) {
            commands.push(cmd.data.toJSON());
            console.log(`   Loaded /${cmd.data.name}`);
        } else {
            console.warn(`   Skipped "${entry.name}": Missing "data" or "execute"`);
        }

    }

    const rest = new REST({ version: '10' }).setToken(token);

    try {
        console.log(`\nStarted refreshing ${commands.length} application (/) commands.`);
        if (config.DEVELOPER) {
            await rest.put(
                Routes.applicationGuildCommands(clientId, guildId),
                { body: commands },
            );
            console.log(`Successfully reloaded application (/) commands to ClientID: ${clientId} with development mode`);
        } else {
            await rest.put(
                Routes.applicationCommands(clientId),
                { body: commands },
            );
            console.log(`Successfully reloaded application (/) commands to ClientID: ${clientId}`);
        }
    } catch (error) {
        console.error(error);
    }
}
import dotenv from "dotenv";

dotenv.config();

const { DISCORD_TOKEN, DISCORD_CLIENT_ID, BACKEND_BASE_URL, DEVELOPMENT_GUILD_ID, DEVELOPER, BACKEND_API_KEY } = process.env;

if (!DISCORD_TOKEN || !DISCORD_CLIENT_ID || !BACKEND_BASE_URL || !BACKEND_API_KEY) {
  throw new Error("Missing environment variables");
}

export const config = {
  DISCORD_TOKEN,
  DISCORD_CLIENT_ID,
  BACKEND_BASE_URL,
  DEVELOPMENT_GUILD_ID,
  DEVELOPER,
  BACKEND_API_KEY,
};


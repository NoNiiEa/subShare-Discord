import dotenv from "dotenv";
dotenv.config();
const { DISCORD_TOKEN, DISCORD_CLIENT_ID, BACKEND_BASE_URL } = process.env;
if (!DISCORD_TOKEN || !DISCORD_CLIENT_ID || !BACKEND_BASE_URL) {
    throw new Error("Missing environment variables");
}
export const config = {
    DISCORD_TOKEN,
    DISCORD_CLIENT_ID,
    BACKEND_BASE_URL,
};

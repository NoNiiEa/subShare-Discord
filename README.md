# subShare-Discord

subShare-Discord is a self-hostable solution designed to simplify the management of shared subscriptions (like Netflix, Spotify, etc.) directly within your Discord server. It features a Go backend API and a TypeScript Discord bot, containerized with Docker for easy deployment.

## Features

- **Group Management**: Create, edit, and delete subscription groups.
- **Member Invitations**: Invite and manage members in your groups.
- **Automated Billing**: Automatically generates bills for members based on a monthly cycle.
- **Payment Verification**: Users can pay bills by uploading a payment slip, which is automatically verified using QR code scanning via third-party APIs (OkSlip/EasySlip).
- **Automated Reminders**: Sends daily reminders to users with unpaid bills in a designated channel.
- **Secure**: API access is protected by a secret token.
- **Easy Deployment**: Fully containerized using Docker and Docker Compose.

## Architecture

The project is a monorepo consisting of two main services:

- **`backend` (Go)**: A RESTful API that handles all business logic, including group management, bill creation, payment processing, and database interactions with SQLite. It also integrates with external slip verification services.
- **`discord-bot` (TypeScript/Node.js)**: The user-facing component that interacts with the Discord API. It handles slash commands, autocomplete, and communicates with the Go backend to perform actions.

Both services are designed to run in Docker containers and communicate over a shared Docker network.

## Getting Started

This project is designed to be run with Docker. For detailed instructions on setup, environment variables, and troubleshooting, please see [**DOCKER.md**](DOCKER.md).

### Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+

### Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/NoNiiEa/subShare-Discord.git
    cd subShare-Discord
    ```

2.  **Create an environment file:**
    Create a `.env` file in the project root. You can use the example from [DOCKER.md](DOCKER.md) as a template. This file will contain your Discord bot token, API keys, and other configuration secrets.

3.  **Build and run the services:**
    ```bash
    docker compose up -d --build
    ```

4.  **Register Slash Commands:**
    After the bot is running, you may need to register its slash commands with Discord. You can do this by running the registration script inside the bot's container:
    ```bash
    docker compose exec discord-bot npm run register
    ```

## Available Commands

### Group Management (`/group`)
- `/group create`: Creates a new subscription sharing group.
- `/group view`: Displays a dashboard of your groups or detailed information for a specific group.
- `/group edit`: Edits the details of a group you own (name, amount, due day, payment info).
- `/group invite`: Invites a user to a group that you own.
- `/group accept`: Accept a pending invitation to join a group.
- `/group delete`: Deletes a group that you own, including all associated bills.

### Bill Management (`/bill`)
- `/bill paid`: Submit a payment slip image to pay for an unpaid bill. The bill is selected via an autocomplete menu.

### Utility Commands
- `/health`: Checks the status of the backend API.
- `/user`: A simple command to get information about the user running it.

## Configuration

All configuration is managed through an `.env` file in the root directory.

- `DISCORD_TOKEN`: Your Discord bot's token.
- `DISCORD_CLIENT_ID`: Your Discord bot's client ID.
- `DEVELOPMENT_GUILD_ID`: The ID of your development server for instant command registration.
- `DEVELOPER`: The Discord user ID of the bot owner/developer.
- `BACKEND_API_KEY`: A secret key shared between the bot and the backend for authentication.
- `EASISLIP_API_URL` & `EASISLIP_API_TOKEN`: Credentials for the EasySlip service (optional).
- `SLIPOK_API_URL` & `SLIPOK_API_KEY`: Credentials for the OkSlip service.
- `API_SECRET_TOKEN`: The same secret key as `BACKEND_API_KEY`, used by the backend to validate requests.
- `CLOUDFLARE_TUNNEL_TOKEN`: Your token for the Cloudflare Tunnel service if you choose to expose your backend publicly.

## Deployment

This repository includes a GitHub Actions workflow (`.github/workflows/deploy.yml`) for continuous deployment. On a push to the `main` branch, the workflow will:

1.  SSH into your server.
2.  Pull the latest changes from the `main` branch.
3.  Create the `.env` file on the server using repository secrets.
4.  Rebuild and restart the Docker containers with `docker compose up -d --build`.
5.  Prune old Docker images to save disk space.

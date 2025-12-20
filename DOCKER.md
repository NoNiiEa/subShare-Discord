# Docker Setup Guide

This project uses Docker Compose to run both the backend API and Discord bot services in containers that communicate through a Docker network.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+

## Environment Variables

Create a `.env` file in the project root with the following variables:

```env
# Discord Bot Configuration
DISCORD_TOKEN=your_discord_bot_token_here
DISCORD_CLIENT_ID=your_discord_client_id_here
DEVELOPMENT_GUILD_ID=your_guild_id_here
DEVELOPER=your_discord_user_id_here

# Backend API Configuration
BACKEND_API_KEY=your_backend_api_key_here
BACKEND_BASE_URL=http://localhost:8080

# EasySlip API Configuration
EASISLIP_API_URL=https://api.easyslip.com
EASISLIP_API_TOKEN=your_easyslip_api_token_here

# Backend API Secret (used by backend for authentication)
API_SECRET_TOKEN=your_backend_api_key_here
```

**Note:** `BACKEND_BASE_URL` in `.env` is for local development. In Docker, the bot automatically uses `http://backend:8080` to communicate with the backend service through the Docker network.

## Building and Running

### Start all services:
```bash
docker-compose up -d
```

### View logs:
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f backend
docker-compose logs -f discord-bot
```

### Stop all services:
```bash
docker-compose down
```

### Rebuild after code changes:
```bash
docker-compose up -d --build
```

## Services

### Backend API
- **Container name:** `subshare-backend`
- **Port:** `8080` (mapped to host)
- **Network:** `subshare-network`
- **Database:** SQLite database stored in Docker volume `backend-db`

### Discord Bot
- **Container name:** `subshare-discord-bot`
- **Network:** `subshare-network`
- **Backend URL:** `http://backend:8080` (internal Docker network)

## Network Communication

Both services are connected to the `subshare-network` bridge network. The Discord bot communicates with the backend using the service name `backend` as the hostname (e.g., `http://backend:8080`).

## Volumes

- `./backend/data:/app/data` - Maps local data directory
- `./backend/config:/app/config` - Maps local config directory
- `backend-db:/app` - Persistent volume for SQLite database

## Troubleshooting

### Check if services are running:
```bash
docker-compose ps
```

### Access backend container shell:
```bash
docker-compose exec backend sh
```

### Access Discord bot container shell:
```bash
docker-compose exec discord-bot sh
```

### View network details:
```bash
docker network inspect subshare-discord_subshare-network
```


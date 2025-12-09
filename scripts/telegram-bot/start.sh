#!/bin/bash

# Focalboard Telegram Bot Startup Script

# Change to script directory
cd "$(dirname "$0")"

# Load .env file if it exists
if [ -f .env ]; then
    echo "Loading environment variables from .env file..."
    export $(grep -v '^#' .env | xargs)
fi

# Check if TELEGRAM_BOT_TOKEN is set
if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
    echo "Error: TELEGRAM_BOT_TOKEN environment variable is not set"
    echo ""
    echo "You can either:"
    echo "  1. Create a .env file: cp .env.example .env (then edit it)"
    echo "  2. Export the variable: export TELEGRAM_BOT_TOKEN='your-bot-token'"
    exit 1
fi

# Set default values if not provided
export FOCALBOARD_API_URL="${FOCALBOARD_API_URL:-http://localhost:8000}"
export WEBHOOK_PORT="${WEBHOOK_PORT:-8001}"

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is not installed"
    exit 1
fi

# Check if dependencies are installed
if ! python3 -c "import telegram" &> /dev/null || ! python3 -c "import flask" &> /dev/null; then
    echo "Installing dependencies..."
    pip3 install -r requirements.txt
fi

echo "Starting Focalboard Telegram Bot..."
echo "Bot Token: ${TELEGRAM_BOT_TOKEN:0:20}..."
echo "Focalboard API: $FOCALBOARD_API_URL"
echo "Webhook Port: $WEBHOOK_PORT"
echo ""

# Start the bot
python3 bot.py

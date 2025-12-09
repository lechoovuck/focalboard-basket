import os
import requests
from telegram import Update
from telegram.ext import Application, CommandHandler, ContextTypes
from flask import Flask, request, jsonify
import threading
import asyncio

FOCALBOARD_API_URL = os.getenv("FOCALBOARD_API_URL", "http://localhost:8000")
BOT_TOKEN = os.getenv("TELEGRAM_BOT_TOKEN")
WEBHOOK_PORT = int(os.getenv("WEBHOOK_PORT", "8001"))

if not BOT_TOKEN:
    raise ValueError("TELEGRAM_BOT_TOKEN environment variable must be set")

# Initialize Flask app for webhook endpoint
app = Flask(__name__)
bot_instance = None
bot_loop = None


async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle the /start command with verification code"""
    args = context.args

    if not args:
        await update.message.reply_text(
            "Welcome! To link your Focalboard account, please use the link "
            "provided in your Focalboard settings."
        )
        return

    verification_code = args[0]
    chat_id = update.effective_chat.id

    try:
        response = requests.get(
            f"{FOCALBOARD_API_URL}/api/v2/telegram/verify",
            params={"code": verification_code, "chat_id": str(chat_id)},
            timeout=5,
        )

        if response.status_code == 200:
            await update.message.reply_text(
                "✅ Successfully linked your Telegram account to Focalboard!\n"
                "You will now receive notifications about card updates."
            )
        else:
            await update.message.reply_text(
                "❌ Failed to link account. Please try again from Focalboard settings."
            )
    except Exception as e:
        print(f"Error verifying account: {e}")
        await update.message.reply_text(
            "❌ Error connecting to Focalboard server. Please try again later."
        )


async def unlink(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle the /unlink command"""
    chat_id = update.effective_chat.id

    await update.message.reply_text(
        "To unlink your account, please go to Focalboard settings."
    )


async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle the /help command"""
    help_text = (
        "🤖 *Focalboard Telegram Bot*\n\n"
        "*Available Commands:*\n"
        "/start - Link your Focalboard account\n"
        "/help - Show this help message\n"
        "/unlink - Unlink your account\n\n"
        "*Notifications:*\n"
        "• Card created\n"
        "• Card updated\n"
        "• @mentions\n"
        "• Card assignments\n\n"
        "Configure your notification preferences in Focalboard settings."
    )
    await update.message.reply_text(help_text, parse_mode="Markdown")


# Webhook endpoint for receiving notification requests from Go server
async def send_telegram_message(chat_id: str, message: str):
    try:
        await bot_instance.bot.send_message(
            chat_id=chat_id, text=message, parse_mode="Markdown"
        )
    except Exception as e:
        print(f"Error sending message to {chat_id}: {e}")


@app.route("/send-notification", methods=["POST"])
def send_notification():
    try:
        data = request.json
        chat_id = data.get("chat_id")
        message = data.get("message")
        if not chat_id or not message:
            return jsonify({"error": "Missing data"}), 400

        # Properly schedule on bot's loop
        future = asyncio.run_coroutine_threadsafe(
            bot_instance.bot.send_message(
                chat_id=chat_id, text=message, parse_mode="Markdown"
            ),
            bot_loop,
        )
        future.result(timeout=10)  # optional: wait with timeout

        return jsonify({"success": True}), 200
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/health", methods=["GET"])
def health():
    """Health check endpoint"""
    return jsonify({"status": "ok", "bot": "running"}), 200


def run_flask():
    """Run Flask server in a separate thread"""
    print(f"Starting webhook server on port {WEBHOOK_PORT}...")
    app.run(host="0.0.0.0", port=WEBHOOK_PORT, debug=False, use_reloader=False)


def main():
    global bot_instance, bot_loop

    # Start Flask webhook server in background thread
    flask_thread = threading.Thread(target=run_flask, daemon=True)
    flask_thread.start()

    # Start Telegram bot
    application = Application.builder().token(BOT_TOKEN).build()
    bot_loop = asyncio.get_event_loop()
    bot_instance = application

    application.add_handler(CommandHandler("start", start))
    application.add_handler(CommandHandler("help", help_command))
    application.add_handler(CommandHandler("unlink", unlink))

    print("Telegram bot started...")
    print(f"Webhook endpoint: http://localhost:{WEBHOOK_PORT}/send-notification")
    application.run_polling()


if __name__ == "__main__":
    main()

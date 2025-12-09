# Telegram Bot Setup - Secure Configuration

## Quick Setup

### 1. Copy your real token to config.json

**Important**: Your `config.json` is now in `.gitignore` and won't be committed to git!

```bash
cd /home/arman/WebProjects/focalboard-basket
cp config.json.example config.json
```

Edit `config.json` and add your bot token:
```json
{
  "telegram": {
    "enabled": true,
    "bot_token": "7009174900:AAGhGWXnhaqFQOGJ3GprACyTvCE7LIk2zGw",
    "bot_username": "@basketdev_notifications_bot",
    "bot_webhook_url": "http://localhost:8001/send-notification"
  }
}
```

### 2. Set environment variable for the bot

```bash
export TELEGRAM_BOT_TOKEN="7009174900:AAGhGWXnhaqFQOGJ3GprACyTvCE7LIk2zGw"
```

Or create a `.env` file in `scripts/telegram-bot/`:
```bash
cd scripts/telegram-bot
cp .env.example .env
# Edit .env and add your token
```

### 3. Start the bot

```bash
cd scripts/telegram-bot
./start.sh
```

### 4. Start Focalboard

```bash
cd ../../server
./focalboard-server
```

## What Changed - Security Improvements

✅ **Removed hardcoded tokens** from all files
✅ **Added `config.json` to `.gitignore`** - won't be committed
✅ **Created `config.json.example`** - template without secrets
✅ **Created `.env.example`** - for bot environment variables
✅ **Bot requires `TELEGRAM_BOT_TOKEN` env var** - fails if not set

## Notification Behavior Changes

### Default Preferences (All OFF)
When a user links their Telegram account, all notification types are **disabled by default**:
- ❌ Card Created - OFF
- ❌ Card Updated - OFF
- ❌ Card Assigned - OFF
- ❌ @Mentions - OFF

Users must explicitly enable the notifications they want in Settings.

### Only Board Members Get Notified
- ✅ Only users who are **members of the board** receive notifications
- ✅ The person who created/updated the card **does NOT** receive a notification
- ✅ Example: If User A creates a card on Board X, only User B, C, D (other members) get notified

### How It Works
1. **Card Created**: Notifies all board members except the creator (if they have `notify_on_card_create` enabled)
2. **Card Updated**: Notifies all board members except the updater (if they have `notify_on_card_update` enabled)
3. **@Mentions**: Notifies the mentioned user (if they have `notify_on_mentions` enabled)
4. **Card Assigned**: Not yet implemented

## Files That Are Now Ignored

These files contain secrets and are in `.gitignore`:
- `config.json` - Contains bot token
- `scripts/telegram-bot/.env` - Bot environment variables

**Safe to commit**:
- `config.json.example` - Template without secrets
- `scripts/telegram-bot/.env.example` - Template without secrets

## Troubleshooting

### "TELEGRAM_BOT_TOKEN environment variable is not set"
Set the token before starting:
```bash
export TELEGRAM_BOT_TOKEN="your-token-here"
./start.sh
```

### Config.json not found
Copy the example:
```bash
cp config.json.example config.json
# Then edit config.json and add your real token
```

### Notifications not working
1. **Check user has linked account**: Settings → Telegram should show "✅ Telegram account linked"
2. **Check preferences are enabled**: User must manually enable notification types in Settings
3. **Check user is a board member**: Only board members get notified
4. **Check bot is running**: `curl http://localhost:8001/health`

## Security Best Practices

1. **Never commit `config.json`** - it's in `.gitignore`
2. **Use environment variables** for production
3. **Regenerate bot token** if it was exposed in git history:
   - Talk to @BotFather on Telegram
   - Use `/revoke` command
   - Update your local files

## Example: Testing Notifications

1. **Link two accounts** (User A and User B) to Telegram
2. **Create a board** and add both users as members
3. **User A enables notifications**:
   - Settings → Telegram → Check "Notify on card create" → Save
4. **User B creates a card**
5. **User A receives notification** in Telegram!
6. **User B does NOT receive notification** (they created it)

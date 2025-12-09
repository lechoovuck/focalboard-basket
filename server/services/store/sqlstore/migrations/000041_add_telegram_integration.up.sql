-- Add telegram_chat_id column to users table
ALTER TABLE users ADD COLUMN telegram_chat_id VARCHAR(255);
ALTER TABLE users ADD COLUMN telegram_notifications_enabled INTEGER DEFAULT 0;

-- Create notification preferences table with one-to-one relationship
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id VARCHAR(255) PRIMARY KEY,
    notify_on_card_create INTEGER DEFAULT 1 NOT NULL,
    notify_on_card_update INTEGER DEFAULT 1 NOT NULL,
    notify_on_card_assign INTEGER DEFAULT 1 NOT NULL,
    notify_on_mentions INTEGER DEFAULT 1 NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Insert default notification preferences for all existing users
INSERT INTO notification_preferences (user_id, notify_on_card_create, notify_on_card_update, notify_on_card_assign, notify_on_mentions, created_at, updated_at)
SELECT id, 1, 1, 1, 1, (strftime('%s', 'now') * 1000), (strftime('%s', 'now') * 1000)
FROM users
WHERE delete_at = 0
AND id NOT IN (SELECT user_id FROM notification_preferences);
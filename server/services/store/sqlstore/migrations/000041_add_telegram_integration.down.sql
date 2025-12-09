DROP TABLE IF EXISTS notification_preferences;
ALTER TABLE users DROP COLUMN telegram_chat_id;
ALTER TABLE users DROP COLUMN telegram_notifications_enabled;
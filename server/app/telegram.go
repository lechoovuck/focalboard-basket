package app

import (
	"time"

	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// telegramVerification holds a temporary verification code mapping while a user links
// their Telegram account. This is intentionally in‑memory and short‑lived so that we
// can test the flow without adding schema changes yet.
type telegramVerification struct {
	userID    string
	expiresAt time.Time
}

const telegramVerificationTTL = 10 * time.Minute

// SetTelegramVerificationCode stores a temporary verification code for linking a Telegram account.
func (a *App) SetTelegramVerificationCode(userID, code string) error {
	if userID == "" || code == "" {
		return model.NewErrBadRequest("invalid userID or code")
	}

	a.telegramVerificationMux.Lock()
	defer a.telegramVerificationMux.Unlock()

	// prune expired codes
	now := time.Now()
	for k, v := range a.telegramVerificationCodes {
		if now.After(v.expiresAt) {
			delete(a.telegramVerificationCodes, k)
		}
	}

	a.telegramVerificationCodes[code] = telegramVerification{
		userID:    userID,
		expiresAt: now.Add(telegramVerificationTTL),
	}

	return nil
}

// GetUserIDFromVerificationCode resolves a verification code back to a user ID.
func (a *App) GetUserIDFromVerificationCode(code string) (string, error) {
	if code == "" {
		return "", model.NewErrBadRequest("empty verification code")
	}

	a.telegramVerificationMux.Lock()
	defer a.telegramVerificationMux.Unlock()

	v, ok := a.telegramVerificationCodes[code]
	if !ok {
		return "", model.NewErrNotFound("verification code not found")
	}

	if time.Now().After(v.expiresAt) {
		delete(a.telegramVerificationCodes, code)
		return "", model.NewErrBadRequest("verification code expired")
	}

	// one‑time use
	delete(a.telegramVerificationCodes, code)

	return v.userID, nil
}

// LinkTelegramAccount associates a Telegram chat ID with a user.
func (a *App) LinkTelegramAccount(userID, chatID string) error {
	if userID == "" || chatID == "" {
		return model.NewErrBadRequest("invalid userID or chatID")
	}

	// Get the current user
	user, err := a.store.GetUserByID(userID)
	if err != nil {
		a.logger.Error("Failed to get user for Telegram linking",
			mlog.String("user_id", userID),
			mlog.Err(err),
		)
		return err
	}

	// Update the user with Telegram info
	user.TelegramChatID = chatID
	user.TelegramNotificationsEnabled = 1

	_, err = a.store.UpdateUser(user)
	if err != nil {
		a.logger.Error("Failed to link Telegram account",
			mlog.String("user_id", userID),
			mlog.String("chat_id", chatID),
			mlog.Err(err),
		)
		return err
	}

	// Ensure notification preferences exist for this user (all enabled by default)
	defaultPrefs := map[string]bool{
		"notify_on_card_create": true,
		"notify_on_card_update": true,
		"notify_on_card_assign": true,
		"notify_on_mentions":    true,
	}
	err = a.store.UpsertTelegramNotificationPreferences(userID, defaultPrefs)
	if err != nil {
		a.logger.Error("Failed to create notification preferences",
			mlog.String("user_id", userID),
			mlog.Err(err),
		)
		// Don't fail the linking process if preferences creation fails
	}

	a.logger.Info("Linked Telegram account",
		mlog.String("user_id", userID),
		mlog.String("chat_id", chatID),
	)

	return nil
}


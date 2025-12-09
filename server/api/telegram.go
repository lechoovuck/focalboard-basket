package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

type TelegramLinkRequest struct {
	VerificationCode string `json:"verification_code"`
}

type TelegramLinkResponse struct {
	BotUsername      string `json:"bot_username"`
	VerificationCode string `json:"verification_code"`
	DeepLink         string `json:"deep_link"`
}

func (a *API) handleTelegramLink(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	code := generateVerificationCode()

	err := a.app.SetTelegramVerificationCode(userID, code)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	botUsername := "basketdev_notifications_bot"
	deepLink := fmt.Sprintf("https://t.me/%s?start=%s", botUsername, code)

	response := TelegramLinkResponse{
		BotUsername:      botUsername,
		VerificationCode: code,
		DeepLink:         deepLink,
	}

	json.NewEncoder(w).Encode(response)
}

func (a *API) handleTelegramVerify(w http.ResponseWriter, r *http.Request) {
	verificationCode := r.URL.Query().Get("code")
	chatID := r.URL.Query().Get("chat_id")

	userID, err := a.app.GetUserIDFromVerificationCode(verificationCode)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	err = a.app.LinkTelegramAccount(userID, chatID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (a *API) handleTelegramUnlink(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "telegram unlink not implemented yet"})
}

func (a *API) handleGetTelegramPreferences(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		a.errorResponse(w, r, fmt.Errorf("missing userID"))
		return
	}

	user, err := a.app.GetUserByID(userID)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	prefs, err := a.app.GetTelegramNotificationPreferences(userID)
	if err != nil {
		// If preferences don't exist, create them with default values
		defaultPrefs := map[string]bool{
			"notify_on_card_create": true,
			"notify_on_card_update": true,
			"notify_on_card_assign": true,
			"notify_on_mentions":    true,
		}
		createErr := a.app.UpsertTelegramNotificationPreferences(userID, defaultPrefs)
		if createErr != nil {
			a.errorResponse(w, r, fmt.Errorf("failed to create notification preferences: %w", createErr))
			return
		}
		prefs = defaultPrefs
	}

	resp := map[string]interface{}{
		"linked":                         user.TelegramChatID != "",
		"telegram_chat_id":               user.TelegramChatID,
		"telegram_notifications_enabled": user.TelegramNotificationsEnabled == 1,
		"preferences":                    prefs,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *API) handleUpdateTelegramPreferences(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		a.errorResponse(w, r, fmt.Errorf("missing userID"))
		return
	}

	var req map[string]bool
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.errorResponse(w, r, fmt.Errorf("invalid JSON body: %w", err))
		return
	}

	allowed := map[string]bool{
		"notify_on_card_create": true,
		"notify_on_card_update": true,
		"notify_on_card_assign": true,
		"notify_on_mentions":    true,
	}
	prefs := make(map[string]bool)
	for k, v := range req {
		if allowed[k] {
			prefs[k] = v
		}
	}

	err := a.app.UpsertTelegramNotificationPreferences(userID, prefs)
	if err != nil {
		a.errorResponse(w, r, err)
		return
	}

	updated, _ := a.app.GetTelegramNotificationPreferences(userID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"preferences": updated,
	})
}

func generateVerificationCode() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TelegramService struct {
	botWebhookURL string
	client        *http.Client
}

type TelegramWebhookPayload struct {
	ChatID  string `json:"chat_id"`
	Message string `json:"message"`
}

func NewTelegramService(botWebhookURL string) *TelegramService {
	return &TelegramService{
		botWebhookURL: botWebhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (t *TelegramService) SendMessage(chatID, message string) error {
	payload := TelegramWebhookPayload{
		ChatID:  chatID,
		Message: message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := t.client.Post(t.botWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send notification to telegram bot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram bot webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (t *TelegramService) FormatCardNotification(cardTitle, boardTitle, userName, action string) string {
	return fmt.Sprintf(
		"🔔 *Focalboard Update*\n\n"+
			"*%s* %s a card:\n"+
			"📝 *%s*\n"+
			"📋 Board: %s",
		userName, action, cardTitle, boardTitle,
	)
}

func (t *TelegramService) FormatMentionNotification(cardTitle, boardTitle, userName string) string {
	return fmt.Sprintf(
		"💬 *You were mentioned!*\n\n"+
			"*%s* mentioned you in:\n"+
			"📝 *%s*\n"+
			"📋 Board: %s",
		userName, cardTitle, boardTitle,
	)
}

func (t *TelegramService) FormatAssignmentNotification(cardTitle, boardTitle, userName string) string {
	return fmt.Sprintf(
		"👤 *You were assigned!*\n\n"+
			"*%s* assigned you to:\n"+
			"📝 *%s*\n"+
			"📋 Board: %s",
		userName, cardTitle, boardTitle,
	)
}

func (t *TelegramService) FormatStatusChangeNotification(cardTitle, boardTitle, userName, oldStatus, newStatus string) string {
	return fmt.Sprintf(
		"🔄 *Card Status Changed*\n\n"+
			"*%s* moved:\n"+
			"📝 *%s*\n"+
			"📋 Board: %s\n\n"+
			"From: %s → To: %s",
		userName, cardTitle, boardTitle, oldStatus, newStatus,
	)
}

func (t *TelegramService) FormatCommentNotification(cardTitle, boardTitle, userName, commentText string) string {
	// Truncate comment if too long
	const maxCommentLen = 200
	if len(commentText) > maxCommentLen {
		commentText = commentText[:maxCommentLen] + "..."
	}

	return fmt.Sprintf(
		"💬 *New Comment*\n\n"+
			"*%s* commented on:\n"+
			"📝 *%s*\n"+
			"📋 Board: %s\n\n"+
			"💭 %s",
		userName, cardTitle, boardTitle, commentText,
	)
}

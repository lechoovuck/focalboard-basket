package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
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
	CardURL string `json:"card_url,omitempty"`
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
	return t.SendMessageWithURL(chatID, message, "")
}

func (t *TelegramService) SendMessageWithURL(chatID, message, cardURL string) error {
	payload := TelegramWebhookPayload{
		ChatID:  chatID,
		Message: message,
		CardURL: cardURL,
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
	escapedTitle := html.EscapeString(cardTitle)
	escapedBoard := html.EscapeString(boardTitle)
	escapedUser := html.EscapeString(userName)
	escapedAction := html.EscapeString(action)

	return fmt.Sprintf(
		"🔔 <b>Focalboard Update</b>\n\n"+
			"<b>%s</b> %s a card:\n"+
			"📝 <b>%s</b>\n"+
			"📋 Board: %s",
		escapedUser, escapedAction, escapedTitle, escapedBoard,
	)
}

func (t *TelegramService) FormatMentionNotification(cardTitle, boardTitle, userName string) string {
	escapedTitle := html.EscapeString(cardTitle)
	escapedBoard := html.EscapeString(boardTitle)
	escapedUser := html.EscapeString(userName)

	return fmt.Sprintf(
		"💬 <b>You were mentioned!</b>\n\n"+
			"<b>%s</b> mentioned you in:\n"+
			"📝 <b>%s</b>\n"+
			"📋 Board: %s",
		escapedUser, escapedTitle, escapedBoard,
	)
}

func (t *TelegramService) FormatAssignmentNotification(cardTitle, boardTitle, userName string) string {
	escapedTitle := html.EscapeString(cardTitle)
	escapedBoard := html.EscapeString(boardTitle)
	escapedUser := html.EscapeString(userName)

	return fmt.Sprintf(
		"👤 <b>You were assigned!</b>\n\n"+
			"<b>%s</b> assigned you to:\n"+
			"📝 <b>%s</b>\n"+
			"📋 Board: %s",
		escapedUser, escapedTitle, escapedBoard,
	)
}

func (t *TelegramService) FormatStatusChangeNotification(cardTitle, boardTitle, userName, oldStatus, newStatus string) string {
	escapedTitle := html.EscapeString(cardTitle)
	escapedBoard := html.EscapeString(boardTitle)
	escapedUser := html.EscapeString(userName)
	escapedOldStatus := html.EscapeString(oldStatus)
	escapedNewStatus := html.EscapeString(newStatus)

	return fmt.Sprintf(
		"🔄 <b>Card Status Changed</b>\n\n"+
			"<b>%s</b> moved:\n"+
			"📝 <b>%s</b>\n"+
			"📋 Board: %s\n\n"+
			"From: %s → To: %s",
		escapedUser, escapedTitle, escapedBoard, escapedOldStatus, escapedNewStatus,
	)
}

func (t *TelegramService) FormatCommentNotification(cardTitle, boardTitle, userName, commentText string) string {
	// Truncate comment if too long
	const maxCommentLen = 200
	if len(commentText) > maxCommentLen {
		commentText = commentText[:maxCommentLen] + "..."
	}

	escapedTitle := html.EscapeString(cardTitle)
	escapedBoard := html.EscapeString(boardTitle)
	escapedUser := html.EscapeString(userName)
	escapedComment := html.EscapeString(commentText)

	return fmt.Sprintf(
		"💬 <b>New Comment</b>\n\n"+
			"<b>%s</b> commented on:\n"+
			"📝 <b>%s</b>\n"+
			"📋 Board: %s\n\n"+
			"💭 %s",
		escapedUser, escapedTitle, escapedBoard, escapedComment,
	)
}

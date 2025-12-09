package notify

import (
	"regexp"
	"strings"

	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// TelegramMentionsBackend handles @mention notifications via Telegram
type TelegramMentionsBackend struct {
	telegram *TelegramService
	store    NotificationStore
	logger   mlog.LoggerIFace
}

// NewTelegramMentionsBackend creates a new Telegram mentions notification backend
func NewTelegramMentionsBackend(telegramBotWebhookURL string, store NotificationStore, logger mlog.LoggerIFace) *TelegramMentionsBackend {
	return &TelegramMentionsBackend{
		telegram: NewTelegramService(telegramBotWebhookURL),
		store:    store,
		logger:   logger,
	}
}

// Name returns the name of this backend
func (tmb *TelegramMentionsBackend) Name() string {
	return "telegram-mentions"
}

// Start initializes the backend
func (tmb *TelegramMentionsBackend) Start() error {
	tmb.logger.Info("Starting Telegram mentions notification backend")
	return nil
}

// ShutDown gracefully shuts down the backend
func (tmb *TelegramMentionsBackend) ShutDown() error {
	tmb.logger.Info("Shutting down Telegram mentions notification backend")
	return nil
}

// BlockChanged handles block change events and detects @mentions
func (tmb *TelegramMentionsBackend) BlockChanged(evt BlockChangeEvent) error {
	// Only process text, comment, and image blocks that might contain mentions
	if evt.BlockChanged == nil {
		return nil
	}

	blockType := evt.BlockChanged.Type
	if blockType != model.TypeText && blockType != model.TypeComment && blockType != model.TypeImage {
		return nil
	}

	// Only process Add and Update actions
	if evt.Action != Add && evt.Action != Update {
		return nil
	}

	// Extract mentions from the block title
	mentions := extractMentions(evt.BlockChanged.Title)
	if len(mentions) == 0 {
		return nil
	}

	// If this is an update, check if these are new mentions
	oldMentions := make(map[string]bool)
	if evt.Action == Update && evt.BlockOld != nil {
		for _, username := range extractMentions(evt.BlockOld.Title) {
			oldMentions[username] = true
		}
	}

	// Get the user who made the change
	var mentioningUser *model.User
	if evt.ModifiedBy != nil {
		var err error
		mentioningUser, err = tmb.store.GetUserByID(evt.ModifiedBy.UserID)
		if err != nil {
			tmb.logger.Error("Failed to get mentioning user",
				mlog.String("user_id", evt.ModifiedBy.UserID),
				mlog.Err(err),
			)
		}
	}

	// Send notification to each mentioned user
	for _, username := range mentions {
		// Skip if this mention existed before (not a new mention)
		if oldMentions[username] {
			continue
		}

		// Get the mentioned user by username
		mentionedUser, err := tmb.store.GetUserByUsername(username)
		if err != nil || mentionedUser == nil {
			// User doesn't exist, skip
			continue
		}

		// Check if user has Telegram enabled
		if mentionedUser.TelegramChatID == "" || mentionedUser.TelegramNotificationsEnabled == 0 {
			continue
		}

		// Get user's notification preferences
		prefs, err := tmb.store.GetTelegramNotificationPreferences(mentionedUser.ID)
		if err != nil {
			// If preferences don't exist, use defaults (all false)
			prefs = map[string]bool{
				"notify_on_card_create": false,
				"notify_on_card_update": false,
				"notify_on_card_assign": false,
				"notify_on_mentions":    false,
			}
		}

		// Check if user wants mention notifications
		if !prefs["notify_on_mentions"] {
			continue
		}

		// Get card title
		cardTitle := "Untitled"
		if evt.Card != nil && evt.Card.Title != "" {
			cardTitle = evt.Card.Title
		}

		// Get board title
		boardTitle := "Unknown Board"
		if evt.Board != nil {
			boardTitle = evt.Board.Title
		}

		// Get username for the person who mentioned
		actorUsername := "Someone"
		if mentioningUser != nil && mentioningUser.Username != "" {
			actorUsername = mentioningUser.Username
		}

		// Format and send the message
		message := tmb.telegram.FormatMentionNotification(cardTitle, boardTitle, actorUsername)
		if err := tmb.telegram.SendMessage(mentionedUser.TelegramChatID, message); err != nil {
			tmb.logger.Error("Failed to send Telegram mention notification",
				mlog.String("mentioned_user_id", mentionedUser.ID),
				mlog.String("mentioned_username", username),
				mlog.Err(err),
			)
		} else {
			tmb.logger.Debug("Sent Telegram mention notification",
				mlog.String("mentioned_user", username),
				mlog.String("actor", actorUsername),
			)
		}
	}

	return nil
}

// extractMentions extracts @username mentions from text
func extractMentions(text string) []string {
	// Match @username (alphanumeric, underscore, hyphen, dot)
	re := regexp.MustCompile(`@([a-zA-Z0-9_.-]+)`)
	matches := re.FindAllStringSubmatch(text, -1)

	mentions := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			username := strings.ToLower(match[1])
			if !seen[username] {
				mentions = append(mentions, username)
				seen[username] = true
			}
		}
	}

	return mentions
}

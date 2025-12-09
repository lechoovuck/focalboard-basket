package notify

import (
	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// TelegramBackend is a notification backend that sends notifications via Telegram
type TelegramBackend struct {
	manager *NotificationManager
	logger  mlog.LoggerIFace
	store   NotificationStore
}

// NewTelegramBackend creates a new Telegram notification backend
func NewTelegramBackend(telegramBotWebhookURL string, store NotificationStore, logger mlog.LoggerIFace) *TelegramBackend {
	return &TelegramBackend{
		manager: NewNotificationManager(telegramBotWebhookURL, store, logger),
		logger:  logger,
		store:   store,
	}
}

// Name returns the name of this backend
func (tb *TelegramBackend) Name() string {
	return "telegram"
}

// Start initializes the backend
func (tb *TelegramBackend) Start() error {
	tb.logger.Info("Starting Telegram notification backend")
	return nil
}

// ShutDown gracefully shuts down the backend
func (tb *TelegramBackend) ShutDown() error {
	tb.logger.Info("Shutting down Telegram notification backend")
	return nil
}

// BlockChanged handles block change events and sends notifications
func (tb *TelegramBackend) BlockChanged(evt BlockChangeEvent) error {
	// Only handle card-related events
	if evt.Card == nil {
		return nil
	}

	// Get the user who made the change
	if evt.ModifiedBy == nil {
		return nil
	}

	// Create a user object from the board member
	user := &model.User{
		ID:       evt.ModifiedBy.UserID,
		Username: evt.ModifiedBy.UserID, // We'll use ID as username for now
	}

	// Try to get the actual user to get the real username
	actualUser, err := tb.manager.store.GetUserByID(evt.ModifiedBy.UserID)
	if err == nil && actualUser != nil {
		user = actualUser
	}

	// Handle different types of updates
	if evt.Action == Update && evt.BlockOld != nil {
		// Check for assignment changes
		tb.checkAndNotifyAssignments(evt, user)

		// Check if this is a comment being added/updated
		if evt.BlockChanged != nil && evt.BlockChanged.Type == "comment" {
			commentText := evt.BlockChanged.Title
			return tb.manager.NotifyCardComment(evt.Card, evt.Board, user, commentText)
		}

		// Check for status changes (property changes in the status field)
		oldStatus, newStatus := tb.detectStatusChange(evt)
		if oldStatus != "" && newStatus != "" && oldStatus != newStatus {
			return tb.manager.NotifyCardStatusChanged(evt.Card, evt.Board, user, oldStatus, newStatus)
		}

		// General card update
		return tb.manager.NotifyCardUpdated(evt.Card, evt.Board, user)
	}

	// Send notification based on action
	switch evt.Action {
	case Add:
		// Check if this is a comment
		if evt.BlockChanged != nil && evt.BlockChanged.Type == "comment" {
			commentText := evt.BlockChanged.Title
			return tb.manager.NotifyCardComment(evt.Card, evt.Board, user, commentText)
		}
		return tb.manager.NotifyCardCreated(evt.Card, evt.Board, user)
	default:
		// Don't send notifications for delete or other actions
		return nil
	}
}

// detectStatusChange detects if the card's status property changed
func (tb *TelegramBackend) detectStatusChange(evt BlockChangeEvent) (oldStatus, newStatus string) {
	if evt.Card == nil || evt.BlockOld == nil {
		return "", ""
	}

	oldProps, oldOk := evt.BlockOld.Fields["properties"].(map[string]interface{})
	newProps, newOk := evt.Card.Fields["properties"].(map[string]interface{})

	if !oldOk || !newOk {
		return "", ""
	}

	// Look for the status property - it's usually stored with a specific key
	// We need to find which property changed and check if it looks like a status
	for propKey, newValue := range newProps {
		oldValue, exists := oldProps[propKey]
		if !exists {
			continue
		}

		// Check if values are different
		newStr := getStringValue(newValue)
		oldStr := getStringValue(oldValue)

		if newStr != oldStr && newStr != "" && oldStr != "" {
			// This could be a status change - we'll use the first changed property
			// In a real implementation, you'd check if this property is actually the status field
			return oldStr, newStr
		}
	}

	return "", ""
}

// getStringValue extracts a string value from an interface
func getStringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

// checkAndNotifyAssignments detects and notifies users who were assigned to a card
func (tb *TelegramBackend) checkAndNotifyAssignments(evt BlockChangeEvent, actor *model.User) {
	tb.logger.Debug("=== Checking for assignments ===",
		mlog.String("card_id", evt.Card.ID),
		mlog.Bool("has_card", evt.Card != nil),
		mlog.Bool("has_old", evt.BlockOld != nil))

	if evt.Card == nil || evt.BlockOld == nil {
		tb.logger.Debug("Skipping assignment check - missing card or old block")
		return
	}

	// Get old and new properties
	oldProps, oldOk := evt.BlockOld.Fields["properties"].(map[string]interface{})
	newProps, newOk := evt.Card.Fields["properties"].(map[string]interface{})

	tb.logger.Debug("Properties check",
		mlog.Bool("old_props_ok", oldOk),
		mlog.Bool("new_props_ok", newOk))

	if !newOk {
		tb.logger.Debug("No new properties found")
		return // No new properties
	}

	// Look for person/select properties that might contain assignments
	for propKey, newValue := range newProps {
		oldValue := interface{}(nil)
		if oldOk {
			oldValue = oldProps[propKey]
		}

		// Check if this is a person assignment (array of user IDs)
		newUserIDs := extractUserIDs(newValue)
		oldUserIDs := extractUserIDs(oldValue)

		tb.logger.Debug("Checking property",
			mlog.String("prop_key", propKey),
			mlog.Int("new_user_ids_count", len(newUserIDs)),
			mlog.Int("old_user_ids_count", len(oldUserIDs)),
			mlog.Any("new_user_ids", newUserIDs),
			mlog.Any("old_user_ids", oldUserIDs))

		// Find newly assigned users
		for _, userID := range newUserIDs {
			if !contains(oldUserIDs, userID) {
				// This user was newly assigned
				tb.logger.Info("Found newly assigned user",
					mlog.String("user_id", userID),
					mlog.String("card_id", evt.Card.ID),
					mlog.String("prop_key", propKey))
				tb.notifyUserAssigned(userID, evt.Card, evt.Board, actor)
			}
		}
	}

	tb.logger.Debug("=== Finished checking assignments ===")
}

// extractUserIDs extracts user IDs from a property value
func extractUserIDs(value interface{}) []string {
	var userIDs []string

	switch v := value.(type) {
	case string:
		if v != "" {
			userIDs = append(userIDs, v)
		}
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				userIDs = append(userIDs, str)
			}
		}
	case []string:
		userIDs = v
	}

	return userIDs
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// notifyUserAssigned sends a notification to a user who was assigned to a card
func (tb *TelegramBackend) notifyUserAssigned(userID string, card *model.Block, board *model.Board, actor *model.User) {
	tb.logger.Debug(">>> notifyUserAssigned called",
		mlog.String("user_id", userID),
		mlog.String("card_id", card.ID))

	// Get the assigned user's info
	assignedUser, err := tb.store.GetUserByID(userID)
	if err != nil {
		tb.logger.Error("Failed to get assigned user",
			mlog.String("user_id", userID),
			mlog.Err(err),
		)
		return
	}

	tb.logger.Debug("Got assigned user",
		mlog.String("username", assignedUser.Username),
		mlog.String("telegram_chat_id", assignedUser.TelegramChatID),
		mlog.Int("telegram_enabled", assignedUser.TelegramNotificationsEnabled))

	// Check if user has Telegram enabled
	if assignedUser.TelegramChatID == "" || assignedUser.TelegramNotificationsEnabled == 0 {
		tb.logger.Debug("User does not have Telegram enabled, skipping")
		return
	}

	// Get user's notification preferences
	prefs, err := tb.store.GetTelegramNotificationPreferences(userID)
	if err != nil {
		tb.logger.Debug("Failed to get preferences, using defaults",
			mlog.Err(err))
		// If preferences don't exist, use defaults (all false)
		prefs = map[string]bool{
			"notify_on_card_create": false,
			"notify_on_card_update": false,
			"notify_on_card_assign": false,
			"notify_on_mentions":    false,
		}
	}

	tb.logger.Debug("User preferences",
		mlog.Any("prefs", prefs))

	// Check if user wants assignment notifications
	if !prefs["notify_on_card_assign"] {
		tb.logger.Debug("User has assignment notifications disabled")
		return
	}

	// Don't notify if the user assigned themselves
	if userID == actor.ID {
		return
	}

	// Get card and board titles
	cardTitle := "Untitled"
	if card.Title != "" {
		cardTitle = card.Title
	}

	boardTitle := board.Title
	actorUsername := actor.Username

	// Format the message
	message := tb.manager.telegram.FormatAssignmentNotification(cardTitle, boardTitle, actorUsername)

	if err := tb.manager.telegram.SendMessage(assignedUser.TelegramChatID, message); err != nil {
		tb.logger.Error("Failed to send Telegram assignment notification",
			mlog.String("assigned_user_id", userID),
			mlog.String("card_id", card.ID),
			mlog.Err(err),
		)
	} else {
		tb.logger.Debug("Sent assignment notification",
			mlog.String("assigned_user", assignedUser.Username),
			mlog.String("card", cardTitle),
		)
	}
}

// OnMention implements the MentionListener interface for handling @mentions
func (tb *TelegramBackend) OnMention(mentionedUserID string, evt BlockChangeEvent) {
	// Get the mentioned user's info
	mentionedUser, err := tb.store.GetUserByID(mentionedUserID)
	if err != nil {
		tb.logger.Error("Failed to get mentioned user",
			mlog.String("user_id", mentionedUserID),
			mlog.Err(err),
		)
		return
	}

	// Check if user has Telegram enabled
	if mentionedUser.TelegramChatID == "" || mentionedUser.TelegramNotificationsEnabled == 0 {
		return
	}

	// Get user's notification preferences
	prefs, err := tb.store.GetTelegramNotificationPreferences(mentionedUserID)
	if err != nil {
		// If preferences don't exist, use defaults
		prefs = map[string]bool{
			"notify_on_card_create": true,
			"notify_on_card_update": true,
			"notify_on_card_assign": true,
			"notify_on_mentions":    true,
		}
	}

	// Check if user wants mention notifications
	if !prefs["notify_on_mentions"] {
		return
	}

	// Get the user who made the mention
	var mentioningUser *model.User
	if evt.ModifiedBy != nil {
		mentioningUser, err = tb.store.GetUserByID(evt.ModifiedBy.UserID)
		if err != nil {
			tb.logger.Error("Failed to get mentioning user",
				mlog.String("user_id", evt.ModifiedBy.UserID),
				mlog.Err(err),
			)
			return
		}
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
	username := "Someone"
	if mentioningUser != nil && mentioningUser.Username != "" {
		username = mentioningUser.Username
	}

	// Format and send the message
	message := tb.manager.telegram.FormatMentionNotification(cardTitle, boardTitle, username)
	if err := tb.manager.telegram.SendMessage(mentionedUser.TelegramChatID, message); err != nil {
		tb.logger.Error("Failed to send Telegram mention notification",
			mlog.String("mentioned_user_id", mentionedUserID),
			mlog.String("card_id", evt.Card.ID),
			mlog.Err(err),
		)
	}
}

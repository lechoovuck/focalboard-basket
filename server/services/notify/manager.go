package notify

import (
	"fmt"

	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// NotificationManager handles notification delivery to various channels
type NotificationManager struct {
	telegram   *TelegramService
	store      NotificationStore
	logger     mlog.LoggerIFace
	serverRoot string
}

// NotificationStore defines the interface for fetching notification-related data
type NotificationStore interface {
	GetUserByID(userID string) (*model.User, error)
	GetUserByUsername(username string) (*model.User, error)
	GetTelegramNotificationPreferences(userID string) (map[string]bool, error)
	GetMembersForBoard(boardID string) ([]*model.BoardMember, error)
	GetBlocksWithType(boardID, blockType string) ([]*model.Block, error)
}

func NewNotificationManager(telegramBotToken string, store NotificationStore, logger mlog.LoggerIFace, serverRoot string) *NotificationManager {
	return &NotificationManager{
		telegram:   NewTelegramService(telegramBotToken),
		store:      store,
		logger:     logger,
		serverRoot: serverRoot,
	}
}

func (nm *NotificationManager) buildCardURL(board *model.Board, cardID string) string {
	if nm.serverRoot == "" {
		return ""
	}

	viewID := ""
	views, err := nm.store.GetBlocksWithType(board.ID, "view")
	if err == nil && len(views) > 0 {
		viewID = views[0].ID
	}

	return fmt.Sprintf("%s/%s/%s/%s", nm.serverRoot, board.ID, viewID, cardID)
}

func (nm *NotificationManager) NotifyCardCreated(card *model.Block, board *model.Board, user *model.User) error {
	return nm.NotifyCardCreatedWithURL(card, board, user, "")
}

func (nm *NotificationManager) NotifyCardCreatedWithURL(card *model.Block, board *model.Board, user *model.User, cardURL string) error {
	if nm.telegram == nil || nm.store == nil {
		return nil
	}

	recipients := nm.getTargetedRecipients(card, user.ID)

	nm.logger.Debug("Card creation - targeted recipients",
		mlog.String("card_id", card.ID),
		mlog.Int("recipient_count", len(recipients)),
		mlog.Any("recipients", recipients))

	for _, recipientID := range recipients {
		if err := nm.sendTelegramNotificationWithURL(recipientID, card, board, user, "created", "", "", cardURL); err != nil {
			nm.logger.Error("Failed to send Telegram notification for card creation",
				mlog.String("user_id", recipientID),
				mlog.String("card_id", card.ID),
				mlog.Err(err),
			)
		}
	}

	return nil
}

// NotifyCardUpdated notifies users assigned to the card about updates (including the updater)
func (nm *NotificationManager) NotifyCardUpdated(card *model.Block, board *model.Board, user *model.User) error {
	return nm.NotifyCardUpdatedWithURL(card, board, user, "")
}

func (nm *NotificationManager) NotifyCardUpdatedWithURL(card *model.Block, board *model.Board, user *model.User, cardURL string) error {
	if nm.telegram == nil || nm.store == nil {
		return nil
	}

	// Get assigned users from card properties
	assignedUserIDs := nm.getAssignedUsers(card)

	nm.logger.Debug("Card update - assigned users",
		mlog.String("card_id", card.ID),
		mlog.Int("assigned_count", len(assignedUserIDs)),
		mlog.Any("assigned_users", assignedUserIDs))

	if cardURL == "" {
		cardURL = nm.buildCardURL(board, card.ID)
	}

	for _, assignedUserID := range assignedUserIDs {
		if err := nm.sendTelegramNotificationWithURL(assignedUserID, card, board, user, "updated", "", "", cardURL); err != nil {
			nm.logger.Error("Failed to send Telegram notification for card update",
				mlog.String("user_id", assignedUserID),
				mlog.String("card_id", card.ID),
				mlog.Err(err),
			)
		}
	}

	return nil
}

// NotifyCardStatusChanged notifies users assigned to the card about status changes
func (nm *NotificationManager) NotifyCardStatusChanged(card *model.Block, board *model.Board, user *model.User, oldStatus, newStatus string) error {
	if nm.telegram == nil || nm.store == nil {
		return nil
	}

	// Get assigned users from card properties
	assignedUserIDs := nm.getAssignedUsers(card)

	cardURL := nm.buildCardURL(board, card.ID)

	// Notify each assigned user (including the updater)
	for _, assignedUserID := range assignedUserIDs {
		if err := nm.sendTelegramNotificationWithURL(assignedUserID, card, board, user, "status_changed", oldStatus, newStatus, cardURL); err != nil {
			nm.logger.Error("Failed to send Telegram notification for status change",
				mlog.String("user_id", assignedUserID),
				mlog.String("card_id", card.ID),
				mlog.Err(err),
			)
		}
	}

	return nil
}

// NotifyCardComment notifies users assigned to the card about new comments
func (nm *NotificationManager) NotifyCardComment(card *model.Block, board *model.Board, user *model.User, commentText string) error {
	if nm.telegram == nil || nm.store == nil {
		return nil
	}

	// Get assigned users from card properties
	assignedUserIDs := nm.getAssignedUsers(card)

	cardURL := nm.buildCardURL(board, card.ID)

	// Notify each assigned user (including the commenter)
	for _, assignedUserID := range assignedUserIDs {
		if err := nm.sendTelegramNotificationWithURL(assignedUserID, card, board, user, "comment", commentText, "", cardURL); err != nil {
			nm.logger.Error("Failed to send Telegram notification for comment",
				mlog.String("user_id", assignedUserID),
				mlog.String("card_id", card.ID),
				mlog.Err(err),
			)
		}
	}

	return nil
}

// getAssignedUsers extracts user IDs from "person" type properties in a card
func (nm *NotificationManager) getAssignedUsers(card *model.Block) []string {
	var assignedUsers []string

	props, ok := card.Fields["properties"].(map[string]interface{})
	if !ok {
		return assignedUsers
	}

	// Look through all properties for user assignments
	for _, value := range props {
		userIDs := extractUserIDsFromValue(value)
		for _, userID := range userIDs {
			if user, err := nm.store.GetUserByID(userID); err == nil && user != nil {
				assignedUsers = append(assignedUsers, userID)
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := []string{}
	for _, userID := range assignedUsers {
		if !seen[userID] {
			seen[userID] = true
			unique = append(unique, userID)
		}
	}

	return unique
}

// getTargetedRecipients returns assigned users + mentioned users
func (nm *NotificationManager) getTargetedRecipients(card *model.Block, excludeUserID string) []string {
	recipients := make(map[string]bool)

	// 1. Get assigned users from card properties
	assignedUsers := nm.getAssignedUsers(card)
	for _, userID := range assignedUsers {
		recipients[userID] = true
	}

	// 2. Get mentioned users from card title
	mentions := extractMentions(card.Title)
	for _, username := range mentions {
		user, err := nm.store.GetUserByUsername(username)
		if err == nil && user != nil {
			recipients[user.ID] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(recipients))
	for userID := range recipients {
		result = append(result, userID)
	}
	return result
}

// extractUserIDsFromValue extracts user IDs from a property value
func extractUserIDsFromValue(value interface{}) []string {
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

// sendTelegramNotification sends a Telegram notification to a specific user
func (nm *NotificationManager) sendTelegramNotification(userID string, card *model.Block, board *model.Board, actor *model.User, action string, extra1, extra2 string) error {
	return nm.sendTelegramNotificationWithURL(userID, card, board, actor, action, extra1, extra2, "")
}

// sendTelegramNotificationWithURL sends a Telegram notification to a specific user with an optional card URL
func (nm *NotificationManager) sendTelegramNotificationWithURL(userID string, card *model.Block, board *model.Board, actor *model.User, action string, extra1, extra2, cardURL string) error {
	// Get the user's info
	targetUser, err := nm.store.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Check if user has Telegram enabled
	if targetUser.TelegramChatID == "" || targetUser.TelegramNotificationsEnabled == 0 {
		return nil
	}

	// Get user's notification preferences
	prefs, err := nm.store.GetTelegramNotificationPreferences(userID)
	if err != nil {
		// If preferences don't exist, use defaults (all false)
		prefs = map[string]bool{
			"notify_on_card_create": false,
			"notify_on_card_update": false,
			"notify_on_card_assign": false,
			"notify_on_mentions":    false,
		}
	}

	// Check if user wants this type of notification based on action
	if action == "created" && !prefs["notify_on_card_create"] {
		return nil
	}
	if (action == "updated" || action == "status_changed" || action == "comment") && !prefs["notify_on_card_update"] {
		return nil
	}

	// Get card title
	cardTitle := "Untitled"
	if card.Title != "" {
		cardTitle = card.Title
	}

	var message string
	switch action {
	case "status_changed":
		message = nm.telegram.FormatStatusChangeNotificationWithURL(cardTitle, board.Title, actor.Username, extra1, extra2, cardURL)
	case "comment":
		message = nm.telegram.FormatCommentNotificationWithURL(cardTitle, board.Title, actor.Username, extra1, cardURL)
	default:
		message = nm.telegram.FormatCardNotificationWithURL(cardTitle, board.Title, actor.Username, action, cardURL)
	}

	return nm.telegram.SendMessage(targetUser.TelegramChatID, message)
}

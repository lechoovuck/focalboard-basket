package notify

import (
	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// NotificationManager handles notification delivery to various channels
type NotificationManager struct {
	telegram *TelegramService
	store    NotificationStore
	logger   mlog.LoggerIFace
}

// NotificationStore defines the interface for fetching notification-related data
type NotificationStore interface {
	GetUserByID(userID string) (*model.User, error)
	GetUserByUsername(username string) (*model.User, error)
	GetTelegramNotificationPreferences(userID string) (map[string]bool, error)
	GetMembersForBoard(boardID string) ([]*model.BoardMember, error)
}

func NewNotificationManager(telegramBotToken string, store NotificationStore, logger mlog.LoggerIFace) *NotificationManager {
	return &NotificationManager{
		telegram: NewTelegramService(telegramBotToken),
		store:    store,
		logger:   logger,
	}
}

// NotifyCardCreated notifies board members about a new card (except the creator)
func (nm *NotificationManager) NotifyCardCreated(card *model.Block, board *model.Board, user *model.User) error {
	if nm.telegram == nil || nm.store == nil {
		return nil
	}

	// Get all board members
	members, err := nm.store.GetMembersForBoard(board.ID)
	if err != nil {
		nm.logger.Error("Failed to get board members",
			mlog.String("board_id", board.ID),
			mlog.Err(err),
		)
		return err
	}

	// Notify each member except the creator
	for _, member := range members {
		// Skip the user who created the card
		if member.UserID == user.ID {
			continue
		}

		if err := nm.sendTelegramNotification(member.UserID, card, board, user, "created", "", ""); err != nil {
			nm.logger.Error("Failed to send Telegram notification for card creation",
				mlog.String("user_id", member.UserID),
				mlog.String("card_id", card.ID),
				mlog.Err(err),
			)
		}
	}

	return nil
}

// NotifyCardUpdated notifies users assigned to the card about updates (including the updater)
func (nm *NotificationManager) NotifyCardUpdated(card *model.Block, board *model.Board, user *model.User) error {
	if nm.telegram == nil || nm.store == nil {
		return nil
	}

	// Get assigned users from card properties
	assignedUserIDs := nm.getAssignedUsers(card)

	nm.logger.Debug("Card update - assigned users",
		mlog.String("card_id", card.ID),
		mlog.Int("assigned_count", len(assignedUserIDs)),
		mlog.Any("assigned_users", assignedUserIDs))

	// Notify each assigned user (including the updater)
	for _, assignedUserID := range assignedUserIDs {
		if err := nm.sendTelegramNotification(assignedUserID, card, board, user, "updated", "", ""); err != nil {
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

	// Notify each assigned user (including the updater)
	for _, assignedUserID := range assignedUserIDs {
		if err := nm.sendTelegramNotification(assignedUserID, card, board, user, "status_changed", oldStatus, newStatus); err != nil {
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

	// Notify each assigned user (including the commenter)
	for _, assignedUserID := range assignedUserIDs {
		if err := nm.sendTelegramNotification(assignedUserID, card, board, user, "comment", commentText, ""); err != nil {
			nm.logger.Error("Failed to send Telegram notification for comment",
				mlog.String("user_id", assignedUserID),
				mlog.String("card_id", card.ID),
				mlog.Err(err),
			)
		}
	}

	return nil
}

// getAssignedUsers extracts all assigned user IDs from a card's properties
func (nm *NotificationManager) getAssignedUsers(card *model.Block) []string {
	var assignedUsers []string

	props, ok := card.Fields["properties"].(map[string]interface{})
	if !ok {
		return assignedUsers
	}

	// Look through all properties for user assignments
	for _, value := range props {
		userIDs := extractUserIDsFromValue(value)
		assignedUsers = append(assignedUsers, userIDs...)
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
		message = nm.telegram.FormatStatusChangeNotification(cardTitle, board.Title, actor.Username, extra1, extra2)
	case "comment":
		message = nm.telegram.FormatCommentNotification(cardTitle, board.Title, actor.Username, extra1)
	default:
		message = nm.telegram.FormatCardNotification(cardTitle, board.Title, actor.Username, action)
	}

	return nm.telegram.SendMessage(targetUser.TelegramChatID, message)
}

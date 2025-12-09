// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Link Telegram Account
export interface TelegramLinkRequest {
    verification_code: string
}

export interface TelegramLinkResponse {
    bot_username: string
    verification_code: string
    deep_link: string
}

// Telegram Preferences
export interface TelegramNotificationPreferences {
    notify_on_card_create: boolean
    notify_on_card_update: boolean
    notify_on_card_assign: boolean
    notify_on_mentions: boolean
}

export interface TelegramPreferencesResponse {
    linked: boolean
    telegram_chat_id: string
    telegram_notifications_enabled: boolean
    preferences: TelegramNotificationPreferences
}

export interface UpdateTelegramPreferencesRequest {
    notify_on_card_create?: boolean
    notify_on_card_update?: boolean
    notify_on_card_assign?: boolean
    notify_on_mentions?: boolean
}

export interface UpdateTelegramPreferencesResponse {
    preferences: TelegramNotificationPreferences
}

// Unlink Response
export interface TelegramUnlinkResponse {
    success: boolean
}

// Error Response (generic)
export interface TelegramErrorResponse {
    error: string
}

// Optional: Union type for all possible responses
export type TelegramApiResponse =
    | TelegramLinkResponse
    | TelegramPreferencesResponse
    | UpdateTelegramPreferencesResponse
    | TelegramUnlinkResponse
    | TelegramErrorResponse

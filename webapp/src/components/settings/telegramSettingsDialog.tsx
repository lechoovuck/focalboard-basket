// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react'

import Dialog from '../dialog'
import Button from '../../widgets/buttons/button'
import Switch from '../../widgets/switch'
import octoClient from '../../octoClient'
import {TelegramNotificationPreferences} from '../../telegram'

import './telegramSettingsDialog.scss'

type Props = {
    onClose: () => void
}

const TelegramSettingsDialog: React.FC<Props> = ({onClose}) => {
    const [linked, setLinked] = useState(false)
    const [loading, setLoading] = useState(false)
    const [loadingPreferences, setLoadingPreferences] = useState(true)

    const [preferences, setPreferences] = useState<TelegramNotificationPreferences | null>(null)

    const [saving, setSaving] = useState(false)
    const [linkError, setLinkError] = useState<string>('')
    const [showDeepLink, setShowDeepLink] = useState<string>('')
    const [verificationCode, setVerificationCode] = useState<string>('')

    useEffect(() => {
        const load = async () => {
            setLoadingPreferences(true)
            try {
                const prefs = await octoClient.getTelegramPreferences()

                if (prefs) {
                    setLinked(prefs.linked)

                    // Set preferences from backend - no fallback defaults
                    if (prefs.preferences) {
                        setPreferences({
                            notify_on_card_create: prefs.preferences.notify_on_card_create,
                            notify_on_card_update: prefs.preferences.notify_on_card_update,
                            notify_on_card_assign: prefs.preferences.notify_on_card_assign,
                            notify_on_mentions: prefs.preferences.notify_on_mentions,
                        })
                    }
                } else {
                    setLinked(false)
                }
            } catch (error) {
                // eslint-disable-next-line no-console
                console.error('Failed to load Telegram preferences:', error)
                setLinkError('Failed to load settings')
            } finally {
                setLoadingPreferences(false)
            }
        }

        load()
    }, [])

    const handleUnlink = async () => {
        setLoading(true)
        setLinkError('')
        try {
            await octoClient.unlinkTelegram()
            setLinked(false)
            setShowDeepLink('')
            setVerificationCode('')
        } catch (error) {
            setLinkError('Failed to unlink Telegram account')
            console.error('Unlink error:', error)
        } finally {
            setLoading(false)
        }
    }

    const handleLinkTelegram = async () => {
        setLoading(true)
        setLinkError('')

        try {
            const response = await octoClient.linkTelegram()

            if (response && response.deep_link) {
                setVerificationCode(response.verification_code)
                setShowDeepLink(response.deep_link)

                window.open(response.deep_link, '_blank')

                startPollingForLinkStatus()
            } else {
                setLinkError('Failed to generate Telegram link')
            }
        } catch (error) {
            setLinkError('Failed to link Telegram account')
            console.error('Link error:', error)
        } finally {
            setLoading(false)
        }
    }

    const startPollingForLinkStatus = () => {
        let attempts = 0
        const maxAttempts = 40

        const pollInterval = setInterval(async () => {
            attempts++

            const prefs = await octoClient.getTelegramPreferences()
            if (prefs && prefs.linked) {
                setLinked(true)
                setShowDeepLink('')
                setVerificationCode('')
                clearInterval(pollInterval)
            }

            if (attempts >= maxAttempts) {
                clearInterval(pollInterval)
            }
        }, 3000)
    }

    const handlePreferenceChange = (key: keyof TelegramNotificationPreferences) => {
        if (!preferences) {
            return
        }

        const newValue = !preferences[key]
        const updatedPrefs = {...preferences, [key]: newValue}
        setPreferences(updatedPrefs)

        // Auto-save preference change
        setSaving(true)
        octoClient.updateTelegramPreferences(updatedPrefs).
            then(() => {
                setSaving(false)
            }).
            catch((error) => {
                // eslint-disable-next-line no-console
                console.error('Failed to save preference:', error)
                setSaving(false)

                // Revert on error
                setPreferences({...updatedPrefs, [key]: !newValue})
            })
    }

    return (
        <Dialog
            onClose={onClose}
            size='small'
            className='TelegramSettingsDialog'
        >
            <div className='dialog-content'>
                <h2 className='dialog-title'>
                    {'Telegram Notifications'}
                </h2>

                {linkError && (
                    <div className='error-message'>
                        {linkError}
                    </div>
                )}

                {linked ? (
                    <div className='linked-section'>
                        <div className='status-indicator success'>
                            <span className='icon'>{'✓'}</span>
                            <span>{'Telegram account linked'}</span>
                        </div>

                        {loadingPreferences && (
                            <div className='loading-preferences'>
                                {'Loading preferences...'}
                            </div>
                        )}

                        {!loadingPreferences && preferences && (
                            <div className='notification-preferences'>
                                <h3 className='section-title'>{'Notification Preferences'}</h3>
                                <div className='preferences-list'>
                                    <div className='preference-item'>
                                        <div className='preference-label'>
                                            <span className='preference-title'>{'Notify on card create'}</span>
                                            <span className='preference-description'>{'Receive notifications when new cards are created on boards you\'re a member of'}</span>
                                        </div>
                                        <Switch
                                            isOn={preferences.notify_on_card_create}
                                            onChanged={() => handlePreferenceChange('notify_on_card_create')}
                                            readOnly={saving}
                                        />
                                    </div>
                                    <div className='preference-item'>
                                        <div className='preference-label'>
                                            <span className='preference-title'>{'Notify on card update'}</span>
                                            <span className='preference-description'>{'Receive notifications when cards you\'re assigned to are updated'}</span>
                                        </div>
                                        <Switch
                                            isOn={preferences.notify_on_card_update}
                                            onChanged={() => handlePreferenceChange('notify_on_card_update')}
                                            readOnly={saving}
                                        />
                                    </div>
                                    <div className='preference-item'>
                                        <div className='preference-label'>
                                            <span className='preference-title'>{'Notify on card assign'}</span>
                                            <span className='preference-description'>{'Receive notifications when you\'re assigned to a card'}</span>
                                        </div>
                                        <Switch
                                            isOn={preferences.notify_on_card_assign}
                                            onChanged={() => handlePreferenceChange('notify_on_card_assign')}
                                            readOnly={saving}
                                        />
                                    </div>
                                    <div className='preference-item'>
                                        <div className='preference-label'>
                                            <span className='preference-title'>{'Notify on mentions'}</span>
                                            <span className='preference-description'>{'Receive notifications when someone mentions you in a card'}</span>
                                        </div>
                                        <Switch
                                            isOn={preferences.notify_on_mentions}
                                            onChanged={() => handlePreferenceChange('notify_on_mentions')}
                                            readOnly={saving}
                                        />
                                    </div>
                                </div>
                                {saving && <div className='saving-indicator'>{'Saving...'}</div>}
                            </div>
                        )}

                        {!loadingPreferences && !preferences && (
                            <div className='error-message'>
                                {'Failed to load notification preferences'}
                            </div>
                        )}

                        <div className='dialog-actions'>
                            <Button
                                onClick={handleUnlink}
                                disabled={loading}
                                emphasis='secondary'
                                size='medium'
                                danger={true}
                            >
                                {loading ? 'Unlinking...' : 'Unlink Account'}
                            </Button>
                        </div>
                    </div>
                ) : (
                    <div className='unlinked-section'>
                        <p className='description'>
                            {'Link your Telegram account to receive real-time notifications about card updates, assignments, and mentions.'}
                        </p>

                        <Button
                            onClick={handleLinkTelegram}
                            disabled={loading}
                            emphasis='primary'
                            size='medium'
                        >
                            {loading ? 'Generating Link...' : 'Link Telegram Account'}
                        </Button>

                        {showDeepLink && (
                            <div className='verification-box'>
                                <h4>{'📱 Complete verification in Telegram'}</h4>
                                <p>
                                    {'If Telegram didn\'t open automatically, click the button below:'}
                                </p>
                                <a
                                    href={showDeepLink}
                                    target='_blank'
                                    rel='noopener noreferrer'
                                    className='telegram-link-button'
                                >
                                    {'Open Telegram Bot'}
                                </a>
                                <div className='verification-code'>
                                    <span>{'Verification code: '}</span>
                                    <code>{verificationCode}</code>
                                </div>
                                <p className='waiting-message'>
                                    {'Waiting for verification...'}
                                </p>
                            </div>
                        )}
                    </div>
                )}
            </div>
        </Dialog>
    )
}

export default TelegramSettingsDialog

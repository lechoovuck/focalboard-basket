// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react'

import octoClient from '../../octoClient'
import {TelegramNotificationPreferences} from '../../telegram'

const TelegramSettings: React.FC = () => {
    const [linked, setLinked] = useState(false)
    const [loading, setLoading] = useState(false)

    const [preferences, setPreferences] = useState<TelegramNotificationPreferences>({
        notify_on_card_create: false,
        notify_on_card_update: false,
        notify_on_card_assign: false,
        notify_on_mentions: false,
    })
    const [isLoadingPrefs, setIsLoadingPrefs] = useState(true)

    const [saving, setSaving] = useState(false)
    const [linkError, setLinkError] = useState<string>('')
    const [showDeepLink, setShowDeepLink] = useState<string>('')
    const [verificationCode, setVerificationCode] = useState<string>('')

    useEffect(() => {
        const load = async () => {
            setIsLoadingPrefs(true)
            try {
                const prefs = await octoClient.getTelegramPreferences()

                if (prefs) {
                    setLinked(prefs.linked)

                    // Ensure each field has a proper default value
                    setPreferences({
                        notify_on_card_create: prefs.preferences?.notify_on_card_create ?? false,
                        notify_on_card_update: prefs.preferences?.notify_on_card_update ?? false,
                        notify_on_card_assign: prefs.preferences?.notify_on_card_assign ?? false,
                        notify_on_mentions: prefs.preferences?.notify_on_mentions ?? false,
                    })
                } else {
                    setLinked(false)
                }
            } catch (error) {
                console.error('Failed to load Telegram preferences:', error)

                // Set safe defaults in case of error
                setPreferences({
                    notify_on_card_create: false,
                    notify_on_card_update: false,
                    notify_on_card_assign: false,
                    notify_on_mentions: false,
                })
            } finally {
                setIsLoadingPrefs(false)
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

    const handlePreferenceChange = (key: keyof typeof preferences) => {
        setPreferences((prev) => ({...prev, [key]: !prev[key]}))
    }

    const handleSavePreferences = async () => {
        setSaving(true)
        try {
            await octoClient.updateTelegramPreferences(preferences)
        } catch (error) {
            console.error('Failed to save preferences:', error)
        } finally {
            setSaving(false)
        }
    }

    return (
        <div className='telegram-settings'>
            <h3>Telegram Notifications</h3>

            {linkError && (
                <div
                    className='error-message'
                    style={{color: 'red', marginBottom: '10px'}}
                >
                    {linkError}
                </div>
            )}

            {linked ? (
                <div>
                    <p style={{color: 'green'}}>✅ Telegram account linked</p>
                    <button
                        onClick={handleUnlink}
                        disabled={loading}
                        style={{marginBottom: '20px'}}
                    >
                        {loading ? 'Unlinking...' : 'Unlink Account'}
                    </button>

                    <div className='notification-preferences'>
                        <h4>Notification Preferences</h4>
                        <div style={{display: 'flex', flexDirection: 'column', gap: '10px'}}>
                            {isLoadingPrefs ? (
                                <p>Loading preferences...</p>
                            ) : (<>
                                <label style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                                    <input
                                        type='checkbox'
                                        checked={preferences.notify_on_card_create}
                                        onChange={() => handlePreferenceChange('notify_on_card_create')}
                                    />
                                    <span>Notify on card create</span>
                                </label>
                                <label style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                                    <input
                                        type='checkbox'
                                        checked={preferences.notify_on_card_update}
                                        onChange={() => handlePreferenceChange('notify_on_card_update')}
                                    />
                                    <span>Notify on card update</span>
                                </label>
                                <label style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                                    <input
                                        type='checkbox'
                                        checked={preferences.notify_on_card_assign}
                                        onChange={() => handlePreferenceChange('notify_on_card_assign')}
                                    />
                                    <span>Notify on card assign</span>
                                </label>
                                <label style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                                    <input
                                        type='checkbox'
                                        checked={preferences.notify_on_mentions}
                                        onChange={() => handlePreferenceChange('notify_on_mentions')}
                                    />
                                    <span>Notify on mentions</span>
                                </label>
                            </>)
                            }
                        </div>
                        <button
                            onClick={handleSavePreferences}
                            disabled={saving}
                            style={{marginTop: '15px'}}
                        >
                            {saving ? 'Saving...' : 'Save Preferences'}
                        </button>
                    </div>
                </div>
            ) : (
                <div>
                    <p>Link your Telegram account to receive notifications</p>
                    <button
                        onClick={handleLinkTelegram}
                        disabled={loading}
                        style={{marginBottom: '15px'}}
                    >
                        {loading ? 'Generating Link...' : 'Link Telegram Account'}
                    </button>

                    {showDeepLink && (
                        <div
                            style={{
                                marginTop: '15px',
                                padding: '15px',
                                backgroundColor: '#f0f0f0',
                                borderRadius: '5px',
                                border: '1px solid #ddd',
                            }}
                        >
                            <p><strong>📱 Complete the verification in Telegram:</strong></p>
                            <p style={{fontSize: '14px', marginTop: '10px'}}>
                                If Telegram didn't open automatically, click the button below:
                            </p>
                            <a
                                href={showDeepLink}
                                target='_blank'
                                rel='noopener noreferrer'
                                style={{
                                    display: 'inline-block',
                                    padding: '10px 20px',
                                    backgroundColor: '#0088cc',
                                    color: 'white',
                                    textDecoration: 'none',
                                    borderRadius: '5px',
                                    marginTop: '10px',
                                }}
                            >
                                Open Telegram Bot
                            </a>
                            <p style={{fontSize: '12px', marginTop: '10px', color: '#666'}}>
                                Verification code: <code>{verificationCode}</code>
                            </p>
                            <p style={{fontSize: '12px', marginTop: '5px', color: '#999'}}>
                                Waiting for verification...
                            </p>
                        </div>
                    )}
                </div>
            )}
        </div>
    )
}

export default TelegramSettings

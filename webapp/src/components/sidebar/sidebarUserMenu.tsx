// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react'
import {useIntl} from 'react-intl'
import {useHistory} from 'react-router-dom'

import {Constants} from '../../constants'
import octoClient from '../../octoClient'
import {IUser} from '../../user'
import FocalboardLogoIcon from '../../widgets/icons/focalboard_logo'
import Menu from '../../widgets/menu'
import MenuWrapper from '../../widgets/menuWrapper'
import {getMe, setMe} from '../../store/users'
import {useAppSelector, useAppDispatch} from '../../store/hooks'

import ModalWrapper from '../modalWrapper'
import RootPortal from '../rootPortal'
import TelegramSettingsDialog from '../settings/telegramSettingsDialog'

import {IAppWindow} from '../../types'

import RegistrationLink from './registrationLink'

import './sidebarUserMenu.scss'

declare let window: IAppWindow

const SidebarUserMenu = () => {
    const dispatch = useAppDispatch()
    const history = useHistory()
    const [showRegistrationLinkDialog, setShowRegistrationLinkDialog] = useState(false)
    const [showTelegramSettings, setShowTelegramSettings] = useState(false)
    const [telegramPrefsLoading, setTelegramPrefsLoading] = useState(false)
    const user = useAppSelector<IUser|null>(getMe)
    const intl = useIntl()

    const handleOpenTelegramSettings = async () => {
        setTelegramPrefsLoading(true)
        try {
            // Pre-fetch the data to ensure it's loaded before opening the dialog
            await octoClient.getTelegramPreferences()
            setShowTelegramSettings(true)
        } catch (error) {
            console.error('Failed to load Telegram preferences:', error)
            // Still show the dialog even if there's an error
            setShowTelegramSettings(true)
        } finally {
            setTelegramPrefsLoading(false)
        }
    }

    return (
        <div className='SidebarUserMenu'>
            <ModalWrapper>
                <MenuWrapper>
                    <div className='logo'>
                        <div className='logo-title'>
                            <FocalboardLogoIcon/>
                            <span>{'Focalboard'}</span>
                            <div className='versionFrame'>
                                <div
                                    className='version'
                                    title={`v${Constants.versionString}`}
                                >
                                    {`v${Constants.versionString}`}
                                </div>
                            </div>
                        </div>
                    </div>
                    <Menu>
                        {user && user.username !== 'single-user' && <>
                            <Menu.Label><b>{user.username}</b></Menu.Label>
                            <Menu.Text
                                id='settings'
                                name={intl.formatMessage({id: 'Sidebar.settings', defaultMessage: 'Settings'})}
                                onClick={handleOpenTelegramSettings}
                            />
                            <Menu.Text
                                id='logout'
                                name={intl.formatMessage({id: 'Sidebar.logout', defaultMessage: 'Log out'})}
                                onClick={async () => {
                                    await octoClient.logout()
                                    dispatch(setMe(null))
                                    history.push('/login')
                                }}
                            />
                            <Menu.Text
                                id='changePassword'
                                name={intl.formatMessage({id: 'Sidebar.changePassword', defaultMessage: 'Change password'})}
                                onClick={async () => {
                                    history.push('/change_password')
                                }}
                            />
                            <Menu.Text
                                id='invite'
                                name={intl.formatMessage({id: 'Sidebar.invite-users', defaultMessage: 'Invite users'})}
                                onClick={async () => {
                                    setShowRegistrationLinkDialog(true)
                                }}
                            />

                            <Menu.Separator/>
                        </>}

                        <Menu.Text
                            id='about'
                            name={intl.formatMessage({id: 'Sidebar.about', defaultMessage: 'About Focalboard'})}
                            onClick={async () => {
                                window.open('https://www.focalboard.com?utm_source=webapp', '_blank')

                                // TODO: Review if this is needed in the future, this is to fix the problem with linux webview links
                                if (window.openInNewBrowser) {
                                    window.openInNewBrowser('https://www.focalboard.com?utm_source=webapp')
                                }
                            }}
                        />
                    </Menu>
                </MenuWrapper>

                {showRegistrationLinkDialog &&
                    <RegistrationLink
                        onClose={() => {
                            setShowRegistrationLinkDialog(false)
                        }}
                    />
                }
            </ModalWrapper>

            {showTelegramSettings &&
                <RootPortal>
                    <TelegramSettingsDialog
                        onClose={() => {
                            setShowTelegramSettings(false)
                        }}
                    />
                </RootPortal>
            }
        </div>
    )
}

export default React.memo(SidebarUserMenu)

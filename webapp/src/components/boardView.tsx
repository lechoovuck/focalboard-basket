// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react'

import {FilterGroup} from '../blocks/filterGroup'
import LocalFilterStorage from '../services/localFilterStorage'
import mutator from '../mutator'

type Props = {
    board: {
        id: string
    }
    activeView: {
        id: string
        fields: {
            filter: FilterGroup
        }
    }
}

// Experimental wrapper that manages a "local vs server" filter mode.
// This component is currently not wired into the main UI, but is kept
// compiling so it can be iterated on later.
const BoardView: React.FC<Props> = (props: Props) => {
    const [useLocalFilters, setUseLocalFilters] = useState<boolean>(() => {
        const pref = localStorage.getItem('use_local_filters')
        return pref === 'true'
    })
    const [, setActiveFilters] = useState<FilterGroup | null>(props.activeView.fields.filter)

    useEffect(() => {
        if (useLocalFilters) {
            const localFilters = LocalFilterStorage.getFilter(props.board.id, props.activeView.id)
            if (localFilters) {
                setActiveFilters(localFilters)
            }
        } else {
            setActiveFilters(props.activeView.fields.filter)
        }
    }, [useLocalFilters, props.board.id, props.activeView.id, props.activeView.fields.filter])

    const handleFilterChange = (newFilters: FilterGroup) => {
        if (useLocalFilters) {
            LocalFilterStorage.saveFilter(props.board.id, props.activeView.id, newFilters)
            setActiveFilters(newFilters)
        } else {
            const newView = {...props.activeView}
            newView.fields.filter = newFilters
            mutator.changeViewFilter(props.board.id, props.activeView.id, props.activeView.fields.filter, newFilters)
        }
    }

    const toggleFilterMode = () => {
        const newMode = !useLocalFilters
        setUseLocalFilters(newMode)
        localStorage.setItem('use_local_filters', String(newMode))

        if (newMode) {
            LocalFilterStorage.saveFilter(
                props.board.id,
                props.activeView.id,
                props.activeView.fields.filter,
            )
        }
    }

    // Currently we only expose the toggle UI; wiring handleFilterChange into the
    // rest of the app will come in a later step.
    return (
        <div className='BoardView'>
            <div className='filter-mode-toggle'>
                <label>
                    <input
                        type='checkbox'
                        checked={useLocalFilters}
                        onChange={toggleFilterMode}
                    />
                    {'Use local-only filters'}
                    <span className='hint'>
                        {useLocalFilters ? '(Filters are private and saved locally)' : '(Filters are shared with team)'}
                    </span>
                </label>
            </div>
        </div>
    )
}

export default BoardView

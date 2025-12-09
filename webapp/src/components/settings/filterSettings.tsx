// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import LocalFilterStorage from '../../services/localFilterStorage'

const FilterSettings: React.FC = () => {
    const handleExport = () => {
        const data = LocalFilterStorage.exportFilters()
        const blob = new Blob([data], {type: 'application/json'})
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `focalboard-filters-${Date.now()}.json`
        a.click()
        URL.revokeObjectURL(url)
    }

    const handleImport = (event: React.ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0]
        if (!file) {
            return
        }

        const reader = new FileReader()
        reader.onload = (e) => {
            const content = e.target?.result as string
            const success = LocalFilterStorage.importFilters(content)
            alert(success ? 'Filters imported successfully!' : 'Failed to import filters')
        }
        reader.readAsText(file)
    }

    const handleClearAll = () => {
        if (confirm('Are you sure you want to clear all local filters?')) {
            LocalFilterStorage.clearAllFilters()
            alert('All local filters cleared')
        }
    }

    return (
        <div className='filter-settings'>
            <h3>Local Filter Management</h3>

            <button onClick={handleExport}>
                Export Local Filters
            </button>

            <label className='import-button'>
                Import Local Filters
                <input
                    type='file'
                    accept='.json'
                    onChange={handleImport}
                    style={{display: 'none'}}
                />
            </label>

            <button
                onClick={handleClearAll}
                className='danger'
            >
                Clear All Local Filters
            </button>
        </div>
    )
}

export default FilterSettings

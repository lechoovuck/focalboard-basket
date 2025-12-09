// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FilterGroup} from '../blocks/filterGroup'

interface FilterState {
    boardId: string
    viewId: string
    filters: FilterGroup
    timestamp: number
}

class LocalFilterStorage {
    private static STORAGE_KEY = 'focalboard_local_filters'
    private static MAX_AGE_DAYS = 30

    static saveFilter(boardId: string, viewId: string, filters: FilterGroup): void {
        const key = this.getKey(boardId, viewId)
        const state: FilterState = {
            boardId,
            viewId,
            filters,
            timestamp: Date.now(),
        }

        try {
            const existing = this.getAllFilters()
            existing[key] = state

            // Clean old filters
            this.cleanOldFilters(existing)

            localStorage.setItem(this.STORAGE_KEY, JSON.stringify(existing))
        } catch (error) {
            console.error('Failed to save local filter:', error)
        }
    }

    static getFilter(boardId: string, viewId: string): FilterGroup | null {
        try {
            const key = this.getKey(boardId, viewId)
            const all = this.getAllFilters()
            const state = all[key]

            if (!state) {
                return null
            }

            // Check if filter is too old
            const ageInDays = (Date.now() - state.timestamp) / (1000 * 60 * 60 * 24)
            if (ageInDays > this.MAX_AGE_DAYS) {
                this.deleteFilter(boardId, viewId)
                return null
            }

            return state.filters
        } catch (error) {
            console.error('Failed to get local filter:', error)
            return null
        }
    }

    static deleteFilter(boardId: string, viewId: string): void {
        try {
            const key = this.getKey(boardId, viewId)
            const all = this.getAllFilters()
            delete all[key]
            localStorage.setItem(this.STORAGE_KEY, JSON.stringify(all))
        } catch (error) {
            console.error('Failed to delete local filter:', error)
        }
    }

    static clearAllFilters(): void {
        try {
            localStorage.removeItem(this.STORAGE_KEY)
        } catch (error) {
            console.error('Failed to clear filters:', error)
        }
    }

    private static getKey(boardId: string, viewId: string): string {
        return `${boardId}:${viewId}`
    }

    private static getAllFilters(): Record<string, FilterState> {
        try {
            const data = localStorage.getItem(this.STORAGE_KEY)
            return data ? JSON.parse(data) : {}
        } catch {
            return {}
        }
    }

    private static cleanOldFilters(filters: Record<string, FilterState>): void {
        const now = Date.now()
        const maxAge = this.MAX_AGE_DAYS * 24 * 60 * 60 * 1000

        Object.keys(filters).forEach((key) => {
            if (now - filters[key].timestamp > maxAge) {
                delete filters[key]
            }
        })
    }

    static exportFilters(): string {
        const filters = this.getAllFilters()
        return JSON.stringify(filters, null, 2)
    }

    static importFilters(jsonString: string): boolean {
        try {
            const filters = JSON.parse(jsonString)
            localStorage.setItem(this.STORAGE_KEY, JSON.stringify(filters))
            return true
        } catch {
            return false
        }
    }
}

export default LocalFilterStorage

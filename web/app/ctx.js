import { useEffect, useState } from "haunted"
import { getLocalAuth } from "./auth.js"

/**
 * @template T
 * @param {T} state
 */
function createStore(state) {
    const listeners = new Set()
    const getState = () => state
    /**
     * @param {T|(function(T):T)} withState
     */
    const setState = withState => {
        // @ts-ignore
        state = typeof withState === 'function' ? withState(state) : withState
        for (const fn of listeners) {
            fn(state)
        }
    }
    /**
     * @param {function(T):any} fn
     */
    const subscribe = fn => {
        listeners.add(fn)
        const unsubscribe = () => {
            listeners.delete(fn)
        }
        return unsubscribe
    }
    return { getState, setState, subscribe }
}

/**
 * @template T
 * @param {object} store
 * @param {function():T} store.getState
 * @param {function(T|(function(T):T)):void} store.setState
 * @param {function(function(T):any):function():void} store.subscribe
 * @returns {[T,function(T|function(T):T):void]}
 */
export function useStore(store) {
    const [state, setState] = useState(store.getState)
    useEffect(() => store.subscribe(setState), [store])
    return [state, store.setState]
}

export const authStore = createStore(getLocalAuth())

export const hasUnreadNotificationsStore = createStore(false)

export const notificationsEnabledStore = createStore(getLocalNotifiactionsEnabled())

function getLocalNotifiactionsEnabled() {
    return Notification.permission === "granted"
        && localStorage.getItem("notifications_enabled") === "true"
}

export function setLocalNotificationsEnabled(val) {
    localStorage.setItem("notifications_enabled", String(val))
}

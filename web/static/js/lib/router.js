let clicksHijacked = false
let routersCount = 0

export function createRouter() {
    const routes = /** @type {Set<{ pattern: string|RegExp, fn: function }>} */ (new Set())
    const listeners = /** @type {Set<function>} */ (new Set())
    let installed = false

    /**
     * @param {string|RegExp} pattern
     * @param {function} fn
     */
    const route = (pattern, fn) => {
        routes.add({ pattern, fn })
    }

    /**
     * @param {function} listener
     */
    const subscribe = listener => {
        listeners.add(listener)
        return () => {
            listeners.delete(listener)
        }
    }

    const onNavigation = () => {
        const pathname = location.pathname
        let result
        for (const route of routes) {
            if (typeof route.pattern === 'string') {
                if (route.pattern !== pathname) {
                    continue
                }

                result = route.fn()
                break
            }

            const match = route.pattern.exec(pathname)
            if (match === null) {
                continue
            }

            const params = match.slice(1).map(decodeURIComponent)
            if (typeof match.groups === 'object' && match.groups !== null) {
                for (const [k, v] of Object.entries(match.groups)) {
                    params[k] = decodeURIComponent(v)
                }
            }

            result = route.fn(params)
            break
        }

        for (const listener of listeners) {
            listener(result)
        }
    }

    const install = () => {
        if (!installed) {
            addEventListener('popstate', onNavigation)
            addEventListener('pushstate', onNavigation)
            addEventListener('replacestate', onNavigation)
            addEventListener('hashchange', onNavigation)
            setTimeout(onNavigation, 0)
            installed = true
            routersCount++
        }

        if (!clicksHijacked) {
            document.addEventListener('click', hijackClicks)
            clicksHijacked = true
        }

        return () => {
            if (installed) {
                removeEventListener('popstate', onNavigation)
                removeEventListener('pushstate', onNavigation)
                removeEventListener('replacestate', onNavigation)
                removeEventListener('hashchange', onNavigation)
                installed = false
                routersCount--
                if (routersCount < 0) {
                    routersCount = 0
                }
            }

            if (clicksHijacked && routersCount === 0) {
                document.removeEventListener('click', hijackClicks)
                clicksHijacked = false
            }
        }
    }

    return { route, subscribe, install }
}

export function navigate(to = location.pathname, replace = false) {
    const state = history.state
    const title = document.title
    if (replace) {
        history.replaceState(state, title, to)
        dispatchEvent(new PopStateEvent('replacestate', { state }))
        return
    }

    history.pushState(state, title, to)
    dispatchEvent(new PopStateEvent('pushstate', { state }))
}

/**
 * @param {MouseEvent} ev
 */
function hijackClicks(ev) {
    if (ev.defaultPrevented
        || ev.button !== 0
        || ev.ctrlKey
        || ev.shiftKey
        || ev.altKey
        || ev.metaKey) {
        return
    }

    const a = /** @type {Element} */ (ev.target).closest('a')
    if (a === null
        || (a.target !== '' && a.target !== '_self')
        || a.hostname !== location.hostname) {
        return
    }

    ev.preventDefault()
    if (a.href === location.href) {
        return
    }

    navigate(a.href)
}

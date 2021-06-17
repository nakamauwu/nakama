
export function createRouter() {
    const routes = /** @type {Set<{pattern:string|RegExp,fn:function}>} */ (new Set())

    /**
     * @param {string|RegExp} pattern
     * @param {function} fn
     */
    const route = (pattern, fn) => {
        routes.add({ pattern, fn })
    }

    const exec = (pathname = location.pathname) => {
        for (const route of routes) {
            if (typeof route.pattern === "string") {
                if (route.pattern !== pathname) {
                    continue
                }

                return route.fn()
            }

            const match = route.pattern.exec(pathname)
            if (match === null) {
                continue
            }

            const params = match.slice(1).map(decodeURIComponent)
            if (typeof match.groups === "object" && match.groups !== null) {
                for (const [k, v] of Object.entries(match.groups)) {
                    params[k] = decodeURIComponent(v)
                }
            }

            return route.fn(params)
        }
    }

    return {
        route,
        exec,
    }
}

/**
 * @param {string} to
 * @param {boolean} replace
 */
export function navigate(to, replace = false) {
    const { state } = history
    const { title } = document
    if (replace) {
        history.replaceState(state, title, to)
        dispatchEvent(new PopStateEvent("replacestate", { state }))
        return
    }

    history.pushState(state, title, to)
    dispatchEvent(new PopStateEvent("pushstate", { state }))
}

/**
 * @param {MouseEvent} ev
 */
export function hijackClicks(ev) {
    if (ev.defaultPrevented
        || ev.button !== 0
        || ev.ctrlKey
        || ev.shiftKey
        || ev.altKey
        || ev.metaKey) {
        return
    }

    const el = ev.target
    if (!(el instanceof Element)) {
        return
    }

    const a = el.closest("a")
    if (a === null
        || (a.target !== "" && a.target !== "_self")
        || a.hostname !== location.hostname) {
        return
    }

    ev.preventDefault()
    if (a.href === location.href) {
        return
    }

    navigate(a.href)
}

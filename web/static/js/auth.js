/**
 * @returns {import("./types.js").User}
 */
export function getAuthUser() {
    const userItem = localStorage.getItem("auth_user")
    if (userItem === null) {
        return null
    }
    if (localStorage.getItem("auth_token") === null) {
        return null
    }
    const expiresAtItem = localStorage.getItem("auth_expires_at")
    if (expiresAtItem === null) {
        return null
    }
    const expiresAt = new Date(expiresAtItem)
    if (isNaN(expiresAt.valueOf()) || expiresAt <= new Date()) {
        return null
    }
    try {
        return JSON.parse(userItem)
    } catch (_) { }
    return null
}

export function isAuthenticated() {
    return getAuthUser() !== null
}

/**
 * @param {function} fn1
 * @param {function} fn2
 */
export function guard(fn1, fn2) {
    return (...args) => isAuthenticated() ? fn1(...args) : fn2(...args)
}

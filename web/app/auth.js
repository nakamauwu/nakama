export function getLocalAuth() {
    const authItem = localStorage.getItem("auth")
    if (authItem === null) {
        return null
    }

    let auth
    try {
        auth = JSON.parse(authItem)
    } catch (_) {
        return null
    }

    if (typeof auth !== "object"
        || auth === null
        || typeof auth.token !== "string"
        || typeof auth.expiresAt !== "string"
        || typeof auth.user !== "object"
        || typeof auth.user === null
        || typeof auth.user.id !== "string"
        || typeof auth.user.username !== "string"
        || !(typeof auth.user.avatarURL === "string" || auth.user.avatarURL === null)) {
        return null
    }

    let expiresAt
    try {
        expiresAt = new Date(auth.expiresAt)
    } catch (_) {
        return null
    }

    if (isNaN(expiresAt.valueOf()) || expiresAt < new Date()) {
        return null
    }

    return {
        token: auth.token,
        expiresAt,
        user: {
            id: auth.user.id,
            username: auth.user.username,
            avatarURL: auth.user.avatarURL,
        }
    }
}

export function setLocalAuth(auth) {
    const newAuth = typeof auth === "function" ? auth(getLocalAuth()) : auth
    if (newAuth === null) {
        localStorage.removeItem("auth")
        return
    }

    try {
        localStorage.setItem("auth", JSON.stringify(auth))
    } catch (_) { }
}

import { doGet } from "../http.js"
import { navigate } from "../lib/router.js"

const frag = document.createDocumentFragment()

export default async function renderLoginCallbackPage() {
    const data = new URLSearchParams(location.hash.substr(1))
    for (const [k, v] of data) {
        data.set(decodeURIComponent(k), decodeURIComponent(v))
    }

    const token = data.get("token")
    const exp = data.get("expires_at")
    if (token === null || exp === null) {
        navigate("/", true)
        return frag
    }

    const expiresAt = new Date(exp)
    if (isNaN(expiresAt.valueOf())) {
        throw new Error("zero expires at time")
    }
    if (expiresAt <= new Date()) {
        throw new Error("token expired")
    }

    const user = await authUser(token)
    localStorage.setItem("auth_user", JSON.stringify(user))
    localStorage.setItem("auth_token", token)
    localStorage.setItem("auth_expires_at", String(expiresAt))

    location.replace("/")
    return frag
}

/**
 * @param {string} token
 * @returns {Promise<import("../types.js").User>}
 */
function authUser(token) {
    return doGet("/api/auth_user", { authorization: "Bearer " + token })
}

import { doGet } from "../http.js"
import { navigate } from "../lib/router.js"

const reUsername = /^[a-zA-Z][a-zA-Z0-9_-]{0,17}$/
const template = document.createElement("template")
template.innerHTML = /*html*/`
    <main class="container login-callback-page">
        <h1>Authenticating you...</h1>
        <div class="error-wrapper"></div>
    </main>
`

export default function renderLoginCallbackPage() {
    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const errorWrapper = page.querySelector(".error-wrapper")
    setTimeout(() => {
        loginCallback().then(({ url, hard }) => {
            if (hard) {
                location.assign(url)
            } else {
                navigate(url)
            }
        }).catch(err => {
            if (!(err instanceof Error)) {
                err = new Error(String(err))
            }
            console.error(err)
            errorWrapper.innerHTML = /*html*/`
                <div class="error-message">
                    <span>Something went wrong:</span>
                    <pre class="error">${err.message}</pre>
                </div>
                <div class="error-actions">
                    <a href="/">Go home</a>
                    <span>or</span>
                    <button class="small" onclick="location.reload()">Retry</button>
                </div>
            `
        })
    }, 10)
    return page
}

/**
 *
 * @returns {Promise<{url:string,hard:boolean}>}
 */
async function loginCallback() {
    const data = new URLSearchParams(location.search.substr(1))
    for (const [k, v] of data) {
        data.set(decodeURIComponent(k), decodeURIComponent(v))
    }

    const retryEndpoint = data.get("retry_endpoint")
    const errMsg = data.get("error")
    if (retryEndpoint !== null && (errMsg === "user not found" || errMsg === "username taken")) {
        const endpoint = new URL(retryEndpoint, location.origin)

        switch (errMsg) {
            case "user not found": {
                if (!confirm("User not found. Do you want to create an account?")) {
                    return { url: "/", hard: false }
                }
                break
            }
            case "username taken": {
                alert("Username taken")
                break
            }
        }
        /**
         * @param {string=} username
         * @returns {Promise<{url:string,hard:boolean}>}
         */
        const run = async (username) => {
            username = prompt("Username:", username)
            if (username === null) {
                return { url: "/", hard: false }
            }

            username = username.trim()
            if (!reUsername.test(username)) {
                alert("invalid username")
                return run(username)
            }

            endpoint.searchParams.set("username", username)
            return { url: endpoint.toString(), hard: true }
        }
        return run()
    }

    if (errMsg !== null) {
        throw new Error(errMsg)
    }

    const token = data.get("token")
    const expiresAtStr = data.get("expires_at")
    if (token === null || expiresAtStr === null) {
        throw new Error("nothing to see here")
    }

    const expiresAt = new Date(expiresAtStr)
    if (isNaN(expiresAt.valueOf()) || expiresAt < new Date()) {
        throw new Error("token expired")
    }

    const user = await authUser(token)

    localStorage.setItem("auth_user", JSON.stringify(user))
    localStorage.setItem("auth_token", token)
    localStorage.setItem("auth_expires_at", expiresAt.toJSON())

    return { url: "/", hard: true }
}

/**
 * @param {string} token
 * @returns {Promise<import("../types.js").User>}
 */
function authUser(token) {
    return doGet("/api/auth_user", { authorization: "Bearer " + token })
}

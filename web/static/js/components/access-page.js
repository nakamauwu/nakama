import { doPost } from "../http.js"
import { isLocalhost } from "../utils.js"

const reUsername = /^[a-zA-Z][a-zA-Z0-9_-]{0,17}$/

const template = document.createElement("template")
template.innerHTML = `
    <div class="container">
        <h1>Nakama</h1>
        <p>Welcome to Nakama, the next social network for anime fans 🤗</p>
        <h2>Access</h2>
        <form id="login-form" class="login-form">
            <input type="email" name="email" placeholder="Email" autocomplete="email" value="shinji@example.org" required>
            <button>
                <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="log-in"><rect width="24" height="24" transform="rotate(-90 12 12)" opacity="0"/><path d="M19 4h-2a1 1 0 0 0 0 2h1v12h-1a1 1 0 0 0 0 2h2a1 1 0 0 0 1-1V5a1 1 0 0 0-1-1z"/><path d="M11.8 7.4a1 1 0 0 0-1.6 1.2L12 11H4a1 1 0 0 0 0 2h8.09l-1.72 2.44a1 1 0 0 0 .24 1.4 1 1 0 0 0 .58.18 1 1 0 0 0 .81-.42l2.82-4a1 1 0 0 0 0-1.18z"/></g></g></svg>
                <span>Login</span>
            </button>
        </form>
        <div class="login-info">
            <em>This is a pre-release version of nakama. All data will be deleted on code changes</em>
        </div>
    </div>
`

export default function renderAccessPage() {
    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const loginForm = /** @type {HTMLFormElement} */ (page.getElementById("login-form"))
    loginForm.addEventListener("submit", onLoginFormSubmit)
    return page
}

/**
 * @param {Event} ev
 */
async function onLoginFormSubmit(ev) {
    ev.preventDefault()
    const form = /** @type {HTMLFormElement} */ (ev.currentTarget)
    const input = form.querySelector("input")
    const button = form.querySelector("button")
    const email = input.value

    input.disabled = true
    button.disabled = true

    try {
        await runLoginProgram(email)
    } catch (err) {
        console.error(err)
        if (err.name === "UserNotFoundError") {
            if (confirm("User not found. Do you want to create an account?")) {
                runRegistrationProgram(email)
            }
            return
        }
        alert(err.message)
        setTimeout(() => {
            input.focus()
        })
    } finally {
        input.disabled = false
        button.disabled = false
    }
}

/**
 * @param {import("../types.js").DevLoginOutput} payload
 */
function saveLogin(payload) {
    localStorage.setItem("auth_user", JSON.stringify(payload.user))
    localStorage.setItem("auth_token", payload.token)
    localStorage.setItem("auth_expires_at", String(payload.expiresAt))
}

/**
 * @param {string} email
 */
async function runLoginProgram(email) {
    if (isLocalhost()) {
        saveLogin(await devLogin(email))
        location.reload()
        return
    }

    await sendMagicLink(email, location.origin + "/login-callback")
    alert("Magic link sent. Go check your inbox to login")
}

/**
 * @param {string} email
 * @param {string=} username
 */
async function runRegistrationProgram(email, username) {
    username = prompt("Username:", username)
    if (username === null) {
        return
    }

    username = username.trim()
    if (!reUsername.test(username)) {
        alert("invalid username")
        runRegistrationProgram(email, username)
        return
    }

    try {
        await createUser(email, username)
        await runLoginProgram(email)
    } catch (err) {
        console.error(err)
        alert(err.message)
        if (err.name === "UsernameTakenError") {
            runRegistrationProgram(email)
        }
    }
}

/**
 * @param {string} email
 * @returns {Promise<import("../types.js").DevLoginOutput>}
 */
function devLogin(email) {
    return doPost("/api/dev_login", { email })
}

/**
 * @param {string} email
 * @param {string} redirectURI
 */
async function sendMagicLink(email, redirectURI) {
    await doPost("/api/send_magic_link", { email, redirectURI })
}

/**
 * @param {string} email
 * @param {string} username
 * @returns {Promise<void>}
 */
function createUser(email, username) {
    return doPost("/api/users", { email, username })
}

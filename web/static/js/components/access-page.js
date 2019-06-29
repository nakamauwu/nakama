import { doPost } from "../http.js";
import { stringifyJSON } from "../lib/json.js";

const reUsername = /^[a-zA-Z][a-zA-Z0-9_-]{0,17}$/

const template = document.createElement("template")
template.innerHTML = `
    <div class="container">
        <h1>Nakama</h1>
        <p>Welcome to Nakama, the next social network for anime fans 🤗</p>
        <h2>Access</h2>
        <form id="login-form" class="login-form">
            <input type="email" placeholder="Email" autocomplete="email" value="john@example.org" required>
            <button>Login</button>
        </form>
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
        saveLogin(await login(email))
        location.reload()
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
    localStorage.setItem("token", payload.token)
    localStorage.setItem("expires_at", String(payload.expiresAt))
    localStorage.setItem("auth_user", stringifyJSON(payload.authUser))
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
        saveLogin(await login(email))
        location.reload()
    } catch (err) {
        console.error(err)
        alert(err.message)
        if (err.name === "UsernameTakenError") {
            runRegistrationProgram(email)
        }
    }
}

/**
 * @returns {Promise<import("../types.js").DevLoginOutput>}
 */
function login(email) {
    return doPost("/api/dev_login", { email })
}

/**
 * @param {string} email
 * @param {string} username
 * @returns {Promise<void>}
 */
function createUser(email, username) {
    return doPost("/api/users", { email, username })
}

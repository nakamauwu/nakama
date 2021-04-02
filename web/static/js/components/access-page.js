import { doGet, doPost } from "../http.js"
import { arrayBufferToBase64, base64ToArrayBuffer, isLocalhost } from "../utils.js"

const reUsername = /^[a-zA-Z][a-zA-Z0-9_-]{0,17}$/

const template = document.createElement("template")
template.innerHTML = /*html*/`
    <div class="container">
        <h1>Nakama</h1>
        <p>Welcome to Nakama, the next social network for anime fans 🤗</p>
        <h2>Access</h2>
        <form id="login-form" name="loginform" action="webauthn" class="login-form">
            <input type="email" name="email" placeholder="Email" autocomplete="email" required>
            <div class="login-form__btns">
                <button type="button" formaction="webauthn" onclick="loginform.action = this.formAction" id="webauthn-btn" class="webauthn-login-btn" hidden>Login with device credentials</button>
                <button type="submit" formaction="email" onclick="loginform.action = this.formAction" id="email-btn">Login with email</button>
            </div>
        </form>
        <div class="login-info">
            <em>This is a pre-release version of nakama. All data will be deleted on code changes</em>
        </div>
    </div>
`

export default function renderAccessPage() {
    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const loginForm = /** @type {HTMLFormElement} */ (page.getElementById("login-form"))
    const webAuthnBtn = /** @type {HTMLButtonElement} */ page.getElementById("webauthn-btn")
    const emailBtn = /** @type {HTMLButtonElement} */ (page.getElementById("email-btn"))
    const emailInput = loginForm.querySelector("input")

    loginForm.addEventListener("submit", onLoginFormSubmit)
    if (isLocalhost() && !(new URLSearchParams(location.search.substr(1)).has("disable_dev_login"))) {
        emailInput.value = "shinji@example.org"
    }

    if ("PublicKeyCredential" in window) {
        PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable().then(ok => {
            if (!ok) {
                return
            }

            webAuthnBtn.hidden = false
            webAuthnBtn.setAttribute("type", "submit")

            emailBtn.classList.add("secondary")
        })
    }
    return page
}

/**
 * @param {Event} ev
 */
async function onLoginFormSubmit(ev) {
    ev.preventDefault()
    const form = /** @type {HTMLFormElement} */ (ev.currentTarget)
    const input = form.querySelector("input")
    const webAuthnBtn = /** @type {HTMLButtonElement} */ (form.querySelector("#webauthn-btn"))
    const emailBtn = /** @type {HTMLButtonElement} */ (form.querySelector("#email-btn"))
    const email = input.value

    input.disabled = true
    webAuthnBtn.disabled = true
    emailBtn.disabled = true

    try {
        if (form.action.endsWith("webauthn")) {
            const opts = await createCredentialRequestOptions(email, localStorage.getItem("webauthn_credential_id"))
            const cred = await navigator.credentials.get(opts)
            saveLogin(await webAuthnLogin(cred))
            location.reload()
            return
        }

        await runLoginProgram(email)
        return
    } catch (err) {
        console.error(err)
        if (err.name === "NoCredentialsError") {
            webAuthnBtn.setAttribute("type", "button")
            webAuthnBtn.hidden = true

            emailBtn.classList.remove("secondary")

            alert(err.message)
            return
        }

        alert(err.message)
        setTimeout(() => {
            input.focus()
        })
    } finally {
        input.disabled = false
        webAuthnBtn.disabled = false
        emailBtn.disabled = false
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
    if (isLocalhost() && !(new URLSearchParams(location.search.substr(1)).has("disable_dev_login"))) {
        saveLogin(await devLogin(email))
        location.reload()
        return
    }

    await sendMagicLink(email, location.origin + "/login-callback")
    alert("Magic link sent. Go check your inbox to login")
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
 * @param {string=} credentialID
 * @returns {Promise<CredentialRequestOptions>}
 */
async function createCredentialRequestOptions(email, credentialID) {
    let endpoint = "/api/credential_request_options?email=" + encodeURIComponent(email)
    if (typeof credentialID === "string" && credentialID != "") {
        endpoint += "&credential_id=" + encodeURIComponent(credentialID)
    }
    const opts = await doGet(endpoint)
    if (!Array.isArray(opts.publicKey.allowCredentials) || opts.publicKey.allowCredentials.length === 0) {
        const err = new Error("no credentials")
        err.name = "NoCredentialsError"
        throw err
    }

    opts.publicKey.challenge = base64ToArrayBuffer(opts.publicKey.challenge)
    opts.publicKey.allowCredentials.forEach((cred, i) => {
        opts.publicKey.allowCredentials[i].id = base64ToArrayBuffer(cred.id)
    })

    return opts
}

/**
 * @param {Credential} cred
 */
async function webAuthnLogin(cred) {
    const b = {
        id: cred.id,
        type: cred.type,
    }
    if (cred["rawId"] instanceof ArrayBuffer) {
        b["rawId"] = arrayBufferToBase64(cred["rawId"])
    }

    if (cred["response"] instanceof AuthenticatorAssertionResponse) {
        const resp = cred["response"]
        b["response"] = {}
        if (resp["authenticatorData"] instanceof ArrayBuffer) {
            b["response"]["authenticatorData"] = arrayBufferToBase64(resp.authenticatorData)
        }
        if (resp["clientDataJSON"] instanceof ArrayBuffer) {
            b["response"]["clientDataJSON"] = arrayBufferToBase64(resp.clientDataJSON)
        }
        if (resp["signature"] instanceof ArrayBuffer) {
            b["response"]["signature"] = arrayBufferToBase64(resp.signature)
        }
        if (resp["userHandle"] instanceof ArrayBuffer) {
            b["response"]["userHandle"] = arrayBufferToBase64(resp.userHandle)
        }
    }

    return doPost("/api/webauthn_login", b)
}

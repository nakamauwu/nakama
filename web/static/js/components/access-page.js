import { doGet, doPost } from "../http.js"
import { translate } from "../i18n/i18n.js"
import { arrayBufferToBase64, base64ToArrayBuffer, isLocalhost } from "../utils.js"

const template = document.createElement("template")
template.innerHTML = /*html*/`
    <div class="container">
        <h1>Nakama</h1>
        <p>${translate("homePage.welcomeMsg")}</p>
        <h2>${translate("homePage.accessHeading")}</h2>
        <form id="login-form" name="loginform" action="webauthn" class="login-form">
            <input type="email" name="email" placeholder="${translate("homePage.emailInputPlaceholder")}" autocomplete="email" required>
            <button type="submit">${translate("homePage.accessBtn")}</button>
        </form>
        <div class="login-info">
            <em>${translate("homePage.preReleaseWarn")}</em>
        </div>
    </div>
`

export default function renderAccessPage() {
    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const loginForm = /** @type {HTMLFormElement} */ (page.getElementById("login-form"))
    const emailInput = loginForm.querySelector("input")

    loginForm.addEventListener("submit", onLoginFormSubmit)
    if (isLocalhost() && !(new URLSearchParams(location.search.substr(1)).has("disable_dev_login"))) {
        emailInput.value = "shinji@example.org"
    }

    return page
}

/**
 * @param {Event} ev
 */
async function onLoginFormSubmit(ev) {
    ev.preventDefault()

    if (typeof navigator.vibrate === "function") {
        navigator.vibrate([50])
    }

    const form = /** @type {HTMLFormElement} */ (ev.currentTarget)
    const input = form.querySelector("input")
    const button = form.querySelector("button")

    const email = input.value

    input.disabled = true
    button.disabled = true

    try {
        await runLoginProgram(email)
        return
    } catch (err) {
        console.error(err)
        alert(translate("errors."+err.name, translate("defaultError")))
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
    if (isLocalhost() && !(new URLSearchParams(location.search.substr(1)).has("disable_dev_login"))) {
        saveLogin(await devLogin(email))
        location.reload()
        return
    }

    const credentialID = localStorage.getItem("webauthn_credential_id")
    if (credentialID !== null && "PublicKeyCredential" in window) {
        const ok = await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable().catch(() => false)
        if (ok) {
            try {
                const opts = await createCredentialRequestOptions(email, credentialID)
                const cred = await navigator.credentials.get(opts)
                saveLogin(await webAuthnLogin(cred))
                location.reload()
                return
            } catch (err) {
                console.error(err)
                if (err.name !== "UserNotFoundError" && err.name !== "NoWebAuthnCredentialsError") {
                    alert(translate("accessPage.webAuthnFailMsg"))
                }
            }
        }
    }

    await sendMagicLink(email, location.origin + "/login-callback")
    alert(translate("accessPage.magicLinkSentMsg"))
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
        const err = new Error("no webAuthn credentials")
        err.name = "NoWebAuthnCredentialsError"
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

import { component, useCallback, useState } from "haunted"
import { html, nothing } from "lit-html"
import { setLocalAuth } from "../auth.js"
import { authStore, useStore } from "../ctx.js"
import { request } from "../http.js"
import "./toast-item.js"

const inLocalhost = ["127.0.0.1", "localhost"].some(s => s === location.hostname)

export default function AccessPage() {
    return html`
        <main class="container">
            <h1>Access Page</h1>
            <p>Welcome to Nakama, the next social network for anime fans 🤗</p>
            <login-form></login-form>
            <div class="warning">
                <p>This is a pre-release version of nakama. Things will break.</p>
            </div>
        </main>
    `
}

function LoginForm() {
    const [, setAuth] = useStore(authStore)
    const [email, setEmail] = useState(inLocalhost ? "shinji@example.org" : "")
    const [fetching, setFetching] = useState(false)
    const [toast, setToast] = useState(null)

    const onSubmit = useCallback(ev => {
        ev.preventDefault()

        setFetching(true)

        const promise = inLocalhost ? devLogin(email) : sendMagicLink(email)
        promise.then(auth => {
            setEmail("")

            if (inLocalhost) {
                setLocalAuth(auth)
                setAuth(auth)
                return
            }

            setToast({
                type: "success",
                content: "Click on the link we sent to your email address to access.",
                timeout: 60000 * 120,
            })
        }, err => {
            const msg = (inLocalhost ? "could not login: " : "could not send magic link: ") + err.message
            console.error(msg)
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setFetching(false)
        })
    }, [email])

    const onEmailInput = useCallback(ev => {
        setEmail(ev.currentTarget.value)
    }, [])

    return html`
        <form class="login-form" @submit=${onSubmit}>
            <input id="email-input" type="email" name="email" placeholder="Email" autocomplete="email" aria-label="Email" required .value=${email} .disabled=${fetching} @input=${onEmailInput}>
            <button .disabled=${fetching}>Login</button>
        </form>
        ${toast !== null ? html`<toast-item .toast=${toast}></toast-item>` : nothing}
    `
}

customElements.define("login-form", component(LoginForm, { useShadowDOM: false }))

/**
 * @param {string} email
 */
function devLogin(email) {
    return request("POST", "/api/dev_login", { body: { email } }).then(resp => resp.body)
}

function sendMagicLink(email, redirectURI = location.origin + "/access-callback") {
    return request("POST", "/api/send_magic_link", { body: { email, redirectURI } })
}

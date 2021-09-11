import { component, html, render, useCallback, useEffect, useState } from "haunted"
import { until } from "lit-html/directives/until.js"
import { registerTranslateConfig, use as useLang } from "lit-translate"
import { setLocalAuth } from "./auth.js"
import "./components/app-header.js"
import { authStore, useStore } from "./ctx.js"
import { request } from "./http.js"
import { createRouter, hijackClicks } from "./router.js"

const router = createRouter()
router.route("/", guardView(view("home")))
router.route("/access-callback", view("access-callback"))
router.route("/search", view("search"))
router.route("/notifications", guardView(view("notifications")))
router.route("/privacy-policy", view("privacy-policy"))
router.route(/^\/posts\/(?<postID>[^\/]+)$/, view("post"))
router.route(/^\/@(?<username>[^\/]+)$/, view("user"))
router.route(/^\/@(?<username>[^\/]+)\/followees$/, view("user-followees"))
router.route(/^\/@(?<username>[^\/]+)\/followers$/, view("user-followers"))
router.route(/.*/, view("not-found"))

addEventListener("click", hijackClicks)

function view(name) {
    return params => html`${until(import(`/components/${name}-page.js`).then(m => m.default({ params }), err => {
        console.error("could not import page:", err)
        return html`
            <div class="container">
                <p class="error">Something went wrong while loading the page.</p>
            </div>
        `
    }), PageLoader())}`
}

function GuardedView({ args, component, fallback }) {
    const [auth] = useStore(authStore)

    return auth !== null ? component(...args) : fallback(...args)
}

// @ts-ignore
customElements.define("guarded-view", component(GuardedView, { useShadowDOM: false }))

/**
 * @param {function} component
 */
function guardView(component, fallback = view("access")) {
    return (...args) => {
        return html`<guarded-view .args=${args} .component=${component} .fallback=${fallback}>`
    }
}

function PageLoader() {
    return html`
        <main class="container loader" aria-busy="true" aria-live="polite">
            <p>Loading page... please wait.</p>
        </main>
    `
}

function RouterView({ router }) {
    const [view, setView] = useState(router.exec())

    const onPopState = useCallback(() => {
        setView(router.exec())
    }, [router])

    useEffect(() => {
        addEventListener("popstate", onPopState)
        addEventListener("pushstate", onPopState)
        addEventListener("replacestate", onPopState)
        addEventListener("hashchange", onPopState)
        return () => {
            removeEventListener("popstate", onPopState)
            removeEventListener("pushstate", onPopState)
            removeEventListener("replacestate", onPopState)
            removeEventListener("hashchange", onPopState)
        }
    }, [])

    return view
}

// @ts-ignore
customElements.define("router-view", component(RouterView, { useShadowDOM: false }))

const oneDayInMs = 1000 * 60 * 60 * 24
const sevenDaysInMs = 1000 * 60 * 60 * 24

function NakamaApp() {
    const [auth, setAuth] = useStore(authStore)

    const tryRefreshAuth = useCallback(() => {
        if (auth === null) {
            return
        }

        const inSevenDays = new Date()
        inSevenDays.setTime(inSevenDays.getTime() + sevenDaysInMs)

        if (auth.expiresAt >= inSevenDays) {
            return
        }

        fetchToken().then(payload => {
            setAuth(auth => {
                const newAuth = {
                    ...auth,
                    ...payload,
                }
                setLocalAuth(newAuth)
                return newAuth
            })
        }, err => {
            console.error("could not refresh auth:", err)
        })
    }, [auth])

    useEffect(() => {
        if (auth === null) {
            return
        }

        tryRefreshAuth()

        const id = setInterval(() => {
            tryRefreshAuth()
        }, oneDayInMs)

        return () => {
            clearInterval(id)
        }
    }, [auth])

    return html`
        <app-header></app-header>
        <router-view .router=${router}></router-view>
    `
}

customElements.define("nakama-app", component(NakamaApp, { useShadowDOM: false }))

registerTranslateConfig({
    loader: lang => fetch(`/i18n/${lang}.json`).then(res => res.json()),
})

const lang = detectLang()
document.documentElement.lang = lang
useLang(lang).then(() => {
    render(html`<nakama-app></nakama-app>`, document.body)
})

function fetchToken() {
    return request("GET", "/api/token")
        .then(resp => resp.body)
        .then(auth => {
            auth.expiresAt = new Date(auth.expiresAt)
            return auth
        })
}

function detectLang() {
    let lang = localStorage.getItem("preferred_lang")
    if (lang === "es") {
        return "es"
    }

    if (Array.isArray(window.navigator.languages)) {
        for (const lang of window.navigator.languages) {
            if (lang === "es" || (typeof lang === "string" && lang.startsWith("es-"))) {
                return "es"
            }
        }
    }

    lang = window.navigator["userLanguage"]
    if (lang === "es" || (typeof lang === "string" && lang.startsWith("es-"))) {
        return "es"
    }


    lang = window.navigator.language
    if (lang === "es" || (typeof lang === "string" && lang.startsWith("es-"))) {
        return "es"
    }

    return "en"
}

if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register("/sw.js")
    navigator.serviceWorker.addEventListener("message", ev => {
        if (typeof ev.data !== "object"
            || ev.data === null
            || ev.data.type !== "notificationclick"
            || typeof ev.data.detail !== "object"
            || ev.data.detail === null
            || typeof ev.data.detail.id !== "string") {
            return
        }

        const n = ev.data.detail
        markNotificationAsRead(n.id).then(() => {
            dispatchEvent(new CustomEvent("notification-read", { bubbles: true, detail: n }))
        })
    })
}

addEventListener("error", onError)
addEventListener("unhandledrejection", onUnHandledRejection)

/**
 * @param {ErrorEvent} ev
 */
function onError(ev) {
    if (ev.error instanceof DOMException && ev.error.name === "AbortError") {
        return
    }

    pushLog(String(ev.error)).catch(err => {
        console.error("could not push error log:", err)
    })
}

/**
 * @param {PromiseRejectionEvent} ev
 */
function onUnHandledRejection(ev) {
    if (ev.reason instanceof DOMException && ev.reason.name === "AbortError") {
        return
    }

    pushLog(String(ev.reason)).catch(err => {
        console.error("could not push unhandled rejection log:", err)
    })
}

function pushLog(err) {
    return request("POST", "/api/logs", { body: { error: err } })
}

function markNotificationAsRead(notificationID) {
    return request("POST", `/api/notifications/${encodeURIComponent(notificationID)}/mark_as_read`)
        .then(() => void 0)
}

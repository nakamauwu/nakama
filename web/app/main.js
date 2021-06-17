import { component, useCallback, useEffect, useState } from "haunted"
import { html, render } from "lit-html"
import { until } from "lit-html/directives/until.js"
import "./components/app-header.js"
import { authStore, useStore } from "./ctx.js"
import { createRouter, hijackClicks } from "./router.js"

const router = createRouter()
router.route("/", guardView(view("home")))
router.route("/access-callback", view("access-callback"))
router.route("/search", view("search"))
router.route("/notifications", guardView(view("notifications")))
router.route(/^\/posts\/(?<postID>[^\/]+)$/, view("post"))
router.route(/^\/@(?<username>[^\/]+)$/, view("user"))
router.route(/^\/@(?<username>[^\/]+)\/followees$/, view("user-followees"))
router.route(/^\/@(?<username>[^\/]+)\/followers$/, view("user-followers"))
router.route(/.*/, view("not-found"))

addEventListener("click", hijackClicks)

function view(name) {
    return params => html`${until(import(`/components/${name}-page.js`).then(m => m.default({ params })), PageLoader())}`
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

function NakamaApp() {
    const [auth] = useStore(authStore)

    const refreshAuth = useCallback(() => {
        if (auth === null) {
            return
        }

        const inOneDay = new Date()
        inOneDay.setTime(inOneDay.getTime() + oneDayInMs)

        if (auth.expiresAt >= inOneDay) {
            return
        }

        console.log("auth expirest in less than one day")
    }, [auth])

    useEffect(() => {
        if (auth === null) {
            return
        }

        refreshAuth()

        const id = setInterval(() => {
            refreshAuth()
        }, oneDayInMs)

        return () => {
            clearInterval(id)
        }
    }, [])

    return html`
        <app-header></app-header>
        <router-view .router=${router}></router-view>
    `
}

customElements.define("nakama-app", component(NakamaApp, { useShadowDOM: false }))

render(html`<nakama-app></nakama-app>`, document.body)

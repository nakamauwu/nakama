import { guard } from "./auth.js"
import renderErrorPage from "./components/error-page.js"
import { createRouter } from "./lib/router.js"

const modulesCache = new Map()
const viewsCache = new Map()
const disconnectEvent = new CustomEvent("disconnect")
const viewAccess = view("access")
const r = createRouter()
r.route("/", guard(view("home"), viewAccess))
r.route("/login-callback", view("login-callback"))
r.route("/search", view("search"))
r.route("/notifications", guard(view("notifications"), viewAccess))
r.route(/^\/users\/(?<username>[a-zA-Z][a-zA-Z0-9_-]{0,17})$/, view("user"))
r.route(/^\/users\/(?<username>[a-zA-Z][a-zA-Z0-9_-]{0,17})\/followers$/, view("followers"))
r.route(/^\/users\/(?<username>[a-zA-Z][a-zA-Z0-9_-]{0,17})\/followees$/, view("followees"))
r.route(/^\/posts\/(?<postID>[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/, view("post"))
r.route(/\//, view("not-found"))
r.subscribe(renderInto(document.querySelector("main")))
r.install()

function view(name) {
    return (...args) => {
        if (viewsCache.has(name)) {
            const renderPage = viewsCache.get(name)
            return renderPage(...args)
        }
        return importWithCache(`/js/components/${name}-page.js`).then(m => {
            const renderPage = m.default
            viewsCache.set(name, renderPage)
            return renderPage(...args)
        })
    }
}

async function importWithCache(identifier) {
    if (modulesCache.has(identifier)) {
        return modulesCache.get(identifier)
    }
    const m = await import(identifier)
    modulesCache.set(identifier, m)
    return m
}

/**
 * @param {Element} target
 */
function renderInto(target) {
    let pages = /** @type {Node[]} */ ([])
    let currentPage = /** @type {Node=} */ (null)
    return async result => {
        while (pages.length !== 0) {
            const page = pages.pop()
            page.dispatchEvent(disconnectEvent)
            target.innerHTML = ""
        }

        let page
        try {
            page = await result
        } catch (err) {
            console.error(err)
            page = renderErrorPage(err)
        }
        if (page instanceof Node) {
            pages.push(page)
            target.innerHTML = ""
            target.appendChild(page)
        }

        setTimeout(activateLinks)
    }
}

function activateLinks() {
    const { pathname } = location
    const links = Array.from(document.querySelectorAll("a"))
    for (const link of links) {
        if (link.pathname === pathname) {
            link.setAttribute("aria-current", "page")
        } else {
            link.removeAttribute("aria-current")
        }
    }
}

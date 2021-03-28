// TODO: reimplement service worker.

const version = 0
const staticCacheName = "static-" + version
const expectedCacheKeys = [
    staticCacheName,
]
const staticPaths = [
    "/css/styles.css",

    "/img/youtube-2017.svg",
    "/img/icons/logo-circle-192.png",
    "/img/icons/logo-circle-512.png",
    "/img/icons/logo-circle.svg",
    "/img/icons/logo-square-1024.png",

    "/js/components/access-page.js",
    "/js/components/avatar.js",
    "/js/components/error-page.js",
    "/js/components/followees-page.js",
    "/js/components/followers-page.js",
    "/js/components/home-page.js",
    "/js/components/icons.js",
    "/js/components/list.js",
    "/js/components/login-callback-page.js",
    "/js/components/not-found-page.js",
    "/js/components/notifications-page.js",
    "/js/components/post-page.js",
    "/js/components/post.js",
    "/js/components/search-page.js",
    "/js/components/switch.js",
    "/js/components/user-page.js",
    "/js/components/user-profile.js",

    "/js/lib/focus-visible.js",
    "/js/lib/router.js",

    "/js/auth.js",
    "/js/header.js",
    "/js/http.js",

    // Don't cache dev only file.
    // "/js/jsconfig.json",

    "/js/main.js",

    // Don't cache dev only file.
    // "/js/types.js",

    "/js/utils.js",

    "/", // index.html
    "/manifest.json",
    "/offline.html",
    "/robots.txt",

    "/sw-reg.js",

    // Never cache service workers.
    // "/sw.js",
]

self.addEventListener("activate", onActivate)
self.addEventListener("install", onInstall)
self.addEventListener('fetch', onFetch)
self.addEventListener("message", onMessage)

/**
 * @param {Event} ev
 */
function onActivate(ev) {
    ev.waitUntil(cleanStaleCache())
}

/**
 * @param {Event} ev
 */
function onInstall(ev) {
    self.skipWaiting()
    ev.waitUntil(cacheStaticFiles())
}


/**
 * @param {Event} ev
 */
function onFetch(ev) {
    const req = /** @type {Request} */ (ev["request"])
    const reqURL = new URL(req.url)
    // Never cache API requests.
    if (reqURL.origin === location.origin && reqURL.pathname.startsWith("/api/")) {
        return false
    }

    ev.respondWith(cachedFirst(req).catch(err => {
        if (req.mode === "navigate") {
            return caches.match("/")
        }
        return Promise.reject(err)
    }))
}

/**
 * @param {MessageEvent} ev
 */
function onMessage(ev) {
    if (typeof ev.data === "object" && ev.data !== null && ev.data["action"] === "skipWaiting") {
        self.skipWaiting()
    }
}

async function cacheStaticFiles() {
    const cache = await caches.open(staticCacheName)
    await cache.addAll(staticPaths)
}

async function cleanStaleCache() {
    const keys = await caches.keys()
    const pp = keys
        .filter(key => !expectedCacheKeys.includes(key))
        .map(key => caches.delete(key))
    await Promise.all(pp)
}

/**
 * @param {RequestInfo} req
 */
async function cachedFirst(req) {
    const res = await caches.match(req)
    return res ? res : fetch(req)
}

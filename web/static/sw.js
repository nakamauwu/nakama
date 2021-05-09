// TODO: reimplement service worker.

const version = 0
const staticCacheName = "static-" + version
const expectedCacheKeys = [
    staticCacheName,
]
const staticPaths = [
    "/offline.html",
]

self.addEventListener("activate", onActivate)
self.addEventListener("install", onInstall)
self.addEventListener('fetch', onFetch)

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
    if (reqURL.origin === location.origin && reqURL.pathname.startsWith("/api/")) {
        return false
    }

    ev.respondWith(fetch(req).catch(err => {
        if (req.mode === "navigate") {
            return caches.match("/offline.html")
        }

        return Promise.reject(err)
    }))
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

const OFFLINE_VERSION = 1
const CACHE_NAME = "offline"
const OFFLINE_URL = "/offline.html"

self.addEventListener("install", ev => {
    ev.waitUntil(cacheOfflinePage())
    self.skipWaiting()
})

async function cacheOfflinePage() {
    const cache = await caches.open(CACHE_NAME)
    await cache.add(new Request(OFFLINE_URL, { cache: "reload" }))
}

self.addEventListener("activate", ev => {
    ev.waitUntil(enableNavigationPreload())
    self.clients.claim()
})

async function enableNavigationPreload() {
    if ("navigationPreload" in self.registration) {
        await self.registration.navigationPreload.enable()
    }
}

self.addEventListener("fetch", ev => {
    if (ev.request.mode === "navigate") {
        ev.respondWith(networkWithOfflineNavigationFallback(ev))
    }
})

self.addEventListener("push", ev => {
    if (!ev.data) {
        return
    }

    const n = ev.data.json()
    if (!n) {
        return
    }

    ev.waitUntil(showNotification(n))
})

self.addEventListener("notificationclick", ev => {
    ev.notification.close()
    ev.waitUntil(openNotificationsPage(ev.notification.data))
})

async function showNotification(n) {
    const title = notificationTitle(n)
    const body = notificationBody(n)
    return self.registration.showNotification(title, {
        body,
        tag: n.id,
        timestamp: n.issuedAt,
        data: n,
        icon: location.origin + "/icons/logo-circle-512.png",
    }).then(() => {
        if ("setAppBadge" in navigator) {
            return navigator.setAppBadge()
        }
    })
}

async function openNotificationsPage(n) {
    return clients.matchAll({
        type: "window"
    }).then(clientList => {
        const pathname = notificationPathname(n)
        for (const client of clientList) {
            if (client.url === pathname && "focus" in client) {
                return client.focus()
            }
        }

        for (const client of clientList) {
            if (client.url === "/notifications" && "focus" in client) {
                return client.focus()
            }
        }

        if ("openWindow" in clients) {
            return clients.openWindow(pathname)
        }
    }).then(client => client.postMessage({
        type: "notificationclick",
        detail: n,
    }).then(() => {
        if ("clearAppBadge" in navigator) {
            return navigator.clearAppBadge()
        }
    }))
}

function notificationPathname(n) {
    if (typeof n.postID === "string" && n.postID !== "") {
        return "/posts/" + encodeURIComponent(n.postID)
    }

    if (n.type === "follow") {
        return "/@" + encodeURIComponent(n.actors[0])
    }

    return "/notifications"
}

async function networkWithOfflineNavigationFallback(ev) {
    try {
        const preloadResponse = await ev.preloadResponse
        if (preloadResponse) {
            return preloadResponse
        }

        const networkResponse = await fetch(ev.request)
        return networkResponse
    } catch (error) {
        const cache = await caches.open(CACHE_NAME)
        const cachedResponse = await cache.match(OFFLINE_URL)
        return cachedResponse
    }
}

function notificationTitle(n) {
    switch (n.type) {
        case "follow":
            return "New follow"
        case "comment":
            return "New commented"
        case "post_mention":
            return "New post mention"
        case "comment_mention":
            return "New comment mention"
    }
    return "New notification"
}

function notificationBody(n) {
    const getActors = () => {
        const aa = n.actors
        switch (aa.length) {
            case 0:
                return "Someone"
            case 1:
                return aa[0]
            case 2:
                return `${aa[0]} and ${aa[1]}`
        }

        return `${aa[0]} and ${aa.length - 1} others`
    }

    const getAction = () => {
        switch (n.type) {
            case "follow":
                return "followed you"
            case "comment":
                return "commented in a post"
            case "post_mention":
                return "mentioned you in a post"
            case "comment_mention":
                return "mentioned you in a comment"
        }
        return "did something"
    }

    return getActors() + " " + getAction()
}

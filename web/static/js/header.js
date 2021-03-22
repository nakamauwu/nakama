import { getAuthUser } from "./auth.js"
import renderAvatarHTML from "./components/avatar.js"
import { doGet, doPost, subscribe } from "./http.js"
import { navigate } from "./lib/router.js"

const rePostRoute = /^\/posts\/(?<postID>[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/
const authUser = getAuthUser()
const authenticated = authUser !== null
const header = document.querySelector("header")

void async function updateHeaderView() {
    header.innerHTML = /*html*/`
        <div class="container">
            <nav>
                <a href="/" title="Home">
                    <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="home"><rect width="24" height="24" opacity="0"/><path d="M20.42 10.18L12.71 2.3a1 1 0 0 0-1.42 0l-7.71 7.89A2 2 0 0 0 3 11.62V20a2 2 0 0 0 1.89 2h14.22A2 2 0 0 0 21 20v-8.38a2.07 2.07 0 0 0-.58-1.44zM10 20v-6h4v6zm9 0h-3v-7a1 1 0 0 0-1-1H9a1 1 0 0 0-1 1v7H5v-8.42l7-7.15 7 7.19z"/></g></g></svg>
                </a>
                ${authenticated ? `
                    <a href="/notifications" id="notifications-link" class="notifications-link" title="Notifications">
                        <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="bell"><rect width="24" height="24" opacity="0"/><path d="M20.52 15.21l-1.8-1.81V8.94a6.86 6.86 0 0 0-5.82-6.88 6.74 6.74 0 0 0-7.62 6.67v4.67l-1.8 1.81A1.64 1.64 0 0 0 4.64 18H8v.34A3.84 3.84 0 0 0 12 22a3.84 3.84 0 0 0 4-3.66V18h3.36a1.64 1.64 0 0 0 1.16-2.79zM14 18.34A1.88 1.88 0 0 1 12 20a1.88 1.88 0 0 1-2-1.66V18h4zM5.51 16l1.18-1.18a2 2 0 0 0 .59-1.42V8.73A4.73 4.73 0 0 1 8.9 5.17 4.67 4.67 0 0 1 12.64 4a4.86 4.86 0 0 1 4.08 4.9v4.5a2 2 0 0 0 .58 1.42L18.49 16z"/></g></g></svg>
                    </a>
                ` : ""}
                <a href="/search" title="Search">
                    <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="search"><rect width="24" height="24" opacity="0"/><path d="M20.71 19.29l-3.4-3.39A7.92 7.92 0 0 0 19 11a8 8 0 1 0-8 8 7.92 7.92 0 0 0 4.9-1.69l3.39 3.4a1 1 0 0 0 1.42 0 1 1 0 0 0 0-1.42zM5 11a6 6 0 1 1 6 6 6 6 0 0 1-6-6z"/></g></g></svg>
                </a>
                ${authenticated ? `
                    <a class="profile-link" href="/users/${authUser.username}" title="Profile">
                        ${renderAvatarHTML(authUser)}
                    </a>
                ` : ""}
            </nav>
        </div>
    `

    if (authenticated) {
        const notificationsLink = /** @type {HTMLAnchorElement} */ (header.querySelector("#notifications-link"))
        void async function () {
            const hasUnreadNotifications = await fetchHasUnreadNotifications()
            if (hasUnreadNotifications) {
                notificationsLink.classList.add("has-unread-notifications")
            }
        }()

        /**
         * @param {import("./types.js").Notification} notification
         */
        const onNotificationArrive = notification => {
            notificationsLink.classList.add("has-unread-notifications")
            dispatchEvent(new CustomEvent("notificationarrive", { detail: notification }))

            const match = rePostRoute.exec(location.pathname)
            if (match !== null) {
                const postID = decodeURIComponent(match.groups["postID"])
                if (postID === notification.postID) {
                    return
                }
            }

            if (Notification.permission !== "granted" && localStorage.getItem("notifications_enabled") !== "true") {
                return
            }

            const sysnotif = new Notification("New notification", {
                tag: notification.id,
                body: getNotificationBody(notification),
            })

            /**
             * @param {Event} ev
             */
            const onSysNotificationClick = async ev => {
                ev.preventDefault()
                sysnotif.close()
                navigate(getNotificationHref(notification))
                await markNotificationAsRead(notification.id)
                notification.read = true
            }

            sysnotif.onclick = onSysNotificationClick
        }

        subscribeToNotifications(onNotificationArrive)
    }
}()

/**
 * @param {import("./types.js").Notification} notification
 */
function getNotificationBody(notification) {
    const actorsText = joinActors(notification.actors)
    switch (notification.type) {
        case "follow": return actorsText + " followed you"
        case "comment": return actorsText + " commented on a post"
        case "post_mention": return actorsText + " mentioned you on a post"
        case "comment_mention": return actorsText + " mentioned you on a comment"
        default: return actorsText + " did something"
    }
}

/**
 * @param {string[]} actors
 */
export function joinActors(actors) {
    switch (actors.length) {
        case 0: return "Somebody"
        case 1: return actors[0]
        case 2: return `${actors[0]} and ${actors[1]}`
        default: return `${actors[0]} and ${actors.length - 1} others`
    }
}

/**
 * @param {import("./types.js").Notification} notification
 */
function getNotificationHref(notification) {
    switch (notification.type) {
        case "follow": return `/users/${encodeURIComponent(notification.actors[0])}`
        case "comment":
        case "post_mention":
        case "comment_mention": return `/posts/${encodeURIComponent(notification.postID)}`
        default: return location.href
    }
}

/**
 * @returns {Promise<boolean>}
 */
function fetchHasUnreadNotifications() {
    return doGet("/api/has_unread_notifications")
}

/**
 * @param {function(import("./types.js").Notification):any} cb
 */
function subscribeToNotifications(cb) {
    return subscribe("/api/notifications", cb)
}

/**
 * @param {string} notificationID
 */
export async function markNotificationAsRead(notificationID) {
    await doPost(`/api/notifications/${encodeURIComponent(notificationID)}/mark_as_read`)
}

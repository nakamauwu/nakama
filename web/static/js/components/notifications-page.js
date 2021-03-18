import { joinActors, markNotificationAsRead } from "../header.js"
import { doGet } from "../http.js"
import { ago } from "../utils.js"
import renderList from "./list.js"
import { renderSwitch } from "./switch.js"

const PAGE_SIZE = 10
const template = document.createElement("template")
template.innerHTML = /*html*/`
    <div class="container">
        <div class="notifications-header">
            <h1>Notifications</h1>
            <div class="notifications-enabler">
                <label for="notification-switch">Notify?</label>
            </div>
        </div>
        <div id="notifications-outlet" class="notifications-wrapper"></div>
    </div>
`

export default async function renderNotificationsPage() {
    const notifications = await fetchNotifications()
    const list = renderList({
        items: notifications,
        loadMoreFunc: fetchNotifications,
        pageSize: PAGE_SIZE,
        renderItem: renderNotification,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const notificationsEnabler = page.querySelector(".notifications-enabler")
    const notificationsOutlet = page.getElementById("notifications-outlet")

    /**
     * @param {boolean} checked
     */
    const onNotificationSwitchChange = async checked => {
        if (!checked) {
            localStorage.removeItem("notifications_enabled")
            return { checked, label: "notify" }
        }

        const existingPerm = Notification.permission
        if (existingPerm === "granted") {
            localStorage.setItem("notifications_enabled", "true")
            return { checked: true, label: "notify" }
        }

        const perm = await Notification.requestPermission()
        if (perm === "granted") {
            localStorage.setItem("notifications_enabled", "true")
            return { checked: true, label: "notify" }
        }

        localStorage.removeItem("notifications_enabled")
        if (perm === "denied" && existingPerm === "denied") {
            alert("Your browser configuration has notifications disabled")
        }

        return { checked: false, label: "notify" }
    }

    const notificationsEnabled = localStorage.getItem("notifications_enabled") === "true" && Notification.permission === "granted"
    const onNotificationArrive = list.enqueue
    const unsubscribeFromNotifications = subscribeToNotifications(onNotificationArrive)
    const notificationSwitch = renderSwitch(notificationsEnabled, "notify", onNotificationSwitchChange)
    notificationSwitch.el.id = "notification-switch"

    const onPageDisconnect = () => {
        unsubscribeFromNotifications()
        list.teardown()
    }

    notificationsEnabler.appendChild(notificationSwitch.el)
    notificationsOutlet.appendChild(list.el)
    page.addEventListener("disconnect", onPageDisconnect)

    return page
}

/**
 * @param {string=} before
 * @returns {Promise<import("../types.js").Notification[]>}
 */
function fetchNotifications(before = "") {
    return doGet(`/api/notifications?last=${PAGE_SIZE}&before=${encodeURIComponent(before)}`)
}

/**
 * @param {function(import("../types.js").Notification):any} cb
 */
function subscribeToNotifications(cb) {
    /**
     * @param {CustomEvent} ev
     */
    const onNotificationArrive = ev => {
        cb(ev.detail)
    }
    addEventListener("notificationarrive", onNotificationArrive)
    return () => {
        removeEventListener("notificationarrive", onNotificationArrive)
    }
}

/**
 * @param {import("../types.js").Notification} notification
 */
function renderNotification(notification) {
    const article = document.createElement("article")
    article.className = "notification"
    if (notification.read) {
        article.classList.add("read")
    }
    let content = joinActors(notification.actors.map(s => `<a href="/users/${encodeURIComponent(s)}">${s}</a>`))
    switch (notification.type) {
        case "follow":
            content += " followed you"
            break
        case "comment":
            content += ` commented on a <a href="/posts/${encodeURIComponent(notification.postID)}">post</a>`
            break
        case "post_mention":
            content += ` mentioned you on a <a href="/posts/${encodeURIComponent(notification.postID)}">post</a>`
            break
        case "comment_mention":
            content += ` mentioned you on a <a href="/posts/${encodeURIComponent(notification.postID)}">comment</a>`
            break
        default:
            content += " did something"
            break
    }
    article.innerHTML = /*html*/`
        <p>${content}</p>
        <time datetime="${notification.issuedAt}">${ago(notification.issuedAt)}</time>
    `
    if (!notification.read) {
        const onNotificationClick = async () => {
            await markNotificationAsRead(notification.id)
            notification.read = true
            article.classList.add("read")
            article.removeEventListener("click", onNotificationClick)
        }

        article.addEventListener("click", onNotificationClick)
    }
    return article
}

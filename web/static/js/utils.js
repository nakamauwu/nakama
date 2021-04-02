const mentionsRegExp = /\B@([a-zA-Z][a-zA-Z0-9_-]{0,17})/g
const urlsRegExp = /\b(https?:\/\/[\-A-Za-z0-9+&@#\/%?=~_|!:,\.;]*[\-A-Za-z0-9+&@#\/%=~_|])/gi
const imageExtRegExp = /(\.gif|\.jpg|\.png)$/
const videoExtRegExp = /(\.mp4|\.webm)$/

export function isLocalhost() {
    return ["localhost", "127.0.0.1"].includes(window.location.hostname)
}

export function isObject(x) {
    return typeof x === "object" && x !== null
}

export function isPlainObject(x) {
    return isObject(x) && !Array.isArray(x)
}

export function smartTrim(s) {
    return s
        .split("\n")
        .map(s => s.replace(/(\s)+/g, "$1").trim())
        .join("\n")
        .replace(/(\n){2,}/g, "$1$1")
        .trim()
}

/**
 * @param {string} s
 */
export function escapeHTML(s) {
    return s
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/"/g, "&#039;")
}

/**
 * @param {string} s
 */
export function linkify(s) {
    return s
        .replace(mentionsRegExp, '<a href="/users/$1">@$1</a>')
        .replace(urlsRegExp, '<a href="$1" target="_blank" rel="noopener">$1</a>')
}

/**
 * @param {string|Date} date
 */
export function ago(date) {
    if (!(date instanceof Date)) {
        date = new Date(date)
    }
    const now = new Date()
    let diff = (now.getTime() - date.getTime()) / 1000
    if (diff <= 60) {
        return "Just now"
    } else if ((diff /= 60) < 60) {
        return (diff | 0) + "m"
    } else if ((diff /= 60) < 24) {
        return (diff | 0) + "h"
    } else if ((diff /= 24) < 7) {
        return (diff | 0) + "d"
    }
    let text = String(date).split(" ")[1] + " " + date.getDate()
    if (now.getFullYear() !== date.getFullYear()) {
        text += ", " + date.getFullYear()
    }
    return text
}

/**
 * @param {Node} oldNode
 * @param {Node} newNode
 */
export function replaceNode(oldNode, newNode) {
    oldNode.parentNode.insertBefore(newNode, oldNode)
    oldNode.parentNode.removeChild(oldNode)
    return newNode
}

/**
 * @param {string} html
 */
export function el(html) {
    const template = document.createElement("template")
    template.innerHTML = html
    return template.content.childElementCount === 1
        ? template.content.firstElementChild
        : template.content
}

/**
 * @param {ParentNode} el
 */
export async function collectMedia(el) {
    const media = /** @type {Node[]} */ ([])
    for (const link of Array.from(el.querySelectorAll("a"))) {
        if (link.hostname === window.location.hostname && link.pathname.startsWith("/users/")) {
            continue
        }

        const youtubeVideoID = findYouTubeVideoID(link.href)
        if (youtubeVideoID !== null) {
            const img = document.createElement("img")
            img.src = `https://img.youtube.com/vi/${youtubeVideoID}/maxresdefault.jpg`
            // img.crossOrigin = ""
            img.width = 540
            img.height = 304
            const a = document.createElement("a")
            a.href = link.href
            a.target = "_blank"
            a.rel = "noopener"
            a.className = "media-item youtube-video-wrapper"
            a.appendChild(img)
            media.push(a)
            continue
        }

        const mt = await mediaType(link.href)
        if (mt === "image") {
            const img = document.createElement("img")
            img.src = link.href
            const a = document.createElement("a")
            a.href = link.href
            a.target = "_blank"
            a.rel = "noopener"
            a.className = "media-item"
            a.appendChild(img)
            media.push(a)
            continue
        }

        if (mt === "video") {
            const video = document.createElement("video")
            video.src = link.href
            video.controls = true
            video.loop = true
            video.volume = 0.5
            video.muted = true
            video.className = "media-item"
            media.push(video)
            continue
        }
    }
    return media
}

/**
 * @param {string} url
 */
async function mediaType(url) {
    if (imageExtRegExp.test(url)) {
        return "image"
    }

    if (videoExtRegExp.test(url)) {
        return "video"
    }

    const endpoint = "/api/proxy?target=" + encodeURIComponent(url)
    return fetch(endpoint, { method: "HEAD", redirect: "follow" }).then(res => {
        const ct = res.headers.get("Content-Type")
        if (ct.startsWith("image/")) {
            return "image"
        }
        if (ct.startsWith("video/")) {
            return "video"
        }

        return ""
    }).catch(_ => "")
}

/***
 * @param {string} href
 * @returns {string|null}
 */
function findYouTubeVideoID(href) {
    try {
        const url = new URL(href)
        if ((url.hostname === "www.youtube.com" || url.hostname === "m.youtube.com") && url.pathname === "/watch" && url.searchParams.has("v")) {
            return url.searchParams.get("v")
        }

        if (url.hostname === "youtu.be" && url.pathname !== "" && url.pathname !== "/") {
            return url.pathname
        }

        if (url.hostname === "www.youtube.com" && url.pathname.startsWith("/embed/") && url.pathname !== "/embed/") {
            return url.pathname.substr(7)
        }
    } catch (_) { }
    return null
}

/**
 * @param {ArrayBuffer} buff
 * @returns {string}
 */
export function arrayBufferToBase64(buff) {
    return btoa(
        new Uint8Array(buff)
            .reduce((data, byte) => data + String.fromCharCode(byte), "")
    )
        .replace(/\+/g, "-")
        .replace(/\//g, "_")
        .replace(/=/g, "")
}

/**
 * @param {string} s
 * @returns {ArrayBuffer}
 */
export function base64ToArrayBuffer(s) {
    return Uint8Array.from(atob(s), c => c.charCodeAt(0))
}

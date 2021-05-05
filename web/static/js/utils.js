import { parseResponse } from "./http.js"

const mentionsRegExp = /\B@([a-zA-Z][a-zA-Z0-9_-]{0,17})/g
const urlsRegExp = /\b(https?:\/\/[\-A-Za-z0-9+&@#\/%?=~_|!:,\.;]*[\-A-Za-z0-9+&@#\/%=~_|])/gi
const imageExtRegExp = /(\.gif|\.jpg|\.jpeg|\.png|\.avif|\.apng|\.webp|\.bmp|\.ico|\.tif|\.tiff|\.svg)$/
const videoExtRegExp = /(\.mp4|\.webm|\.3gp|\.mov)$/
const audioExtRegExp = /(\.wav|\.mp3|\.aac|\.ogg|\.flac)$/

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

        const imgurID = findImgurID(link.href)
        if (imgurID !== null) {
            const img = document.createElement("img")
            // TODO: better detect imgur image extension.
            img.src = "https://i.imgur.com/" + encodeURIComponent(imgurID) + ".png"
            const a = document.createElement("a")
            a.href = link.href
            a.target = "_blank"
            a.rel = "noopener"
            a.className = "media-item imgur"
            a.appendChild(img)
            media.push(a)
            continue
        }

        const youtubeVideoID = findYouTubeVideoID(link.href)
        if (youtubeVideoID !== null) {
            const img = document.createElement("img")
            img.src = `https://img.youtube.com/vi/${youtubeVideoID}/0.jpg`
            // img.crossOrigin = ""
            img.width = 540
            img.height = 304
            const a = document.createElement("a")
            a.href = link.href
            a.target = "_blank"
            a.rel = "noopener"
            a.className = "media-item youtube-video-wrapper"
            a.appendChild(img)
            a.onclick = ev => {
                ev.preventDefault()
                ev.stopImmediatePropagation()

                const iframe = document.createElement("iframe")
                iframe.src = "https://www.youtube.com/embed/" + encodeURIComponent(youtubeVideoID) + "?autoplay=1"
                iframe.setAttribute("frameborder", "0")
                iframe.allow = "accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                iframe.allowFullscreen = true
                iframe.className = "media-item youtube"
                a.insertAdjacentElement("afterend", iframe)
                a.remove()
            }
            media.push(a)
            continue
        }

        const coubVideoID = findCoubVideoID(link.href)
        if (coubVideoID !== null) {
            const iframe = document.createElement("iframe")
            iframe.src = "https://coub.com/embed/" + encodeURIComponent(coubVideoID) + "?muted=false&autostart=false&originalSize=true&startWithHD=true"
            iframe.setAttribute("frameborder", "0")
            iframe.allow = "accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
            iframe.allowFullscreen = true
            iframe.className = "media-item coub"
            media.push(iframe)
            continue
        }

        if (isSoundCloudURL(link.href)) {
            const iframe = await getSoundCloudIframe(link.href)
            iframe.className = "media-item soundcloud"
            media.push(iframe)
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

        if (mt === "audio") {
            const audio = document.createElement("audio")
            audio.src = link.href
            audio.controls = true
            audio.loop = true
            audio.volume = 0.5
            audio.autoplay = false
            audio.className = "media-item"
            media.push(audio)
            continue
        }

        if (mt === "video") {
            const video = document.createElement("video")
            video.src = link.href
            video.controls = true
            video.loop = true
            video.volume = 0.5
            video.autoplay = false
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

    if (audioExtRegExp.test(url)) {
        return "audio"
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
        if (ct.startsWith("audio/")) {
            return "audio"
        }

        return ""
    }).catch(_ => "")
}

function findImgurID(href) {
    try {
        const url = new URL(href)
        if (url.hostname !== "imgur.com") {
            return null
        }

        const parts = url.pathname.split("/")
        if (parts.length === 2 && !parts[1].includes(".")) {
            return parts[1]
        }
    } catch (_) { }
    return null
}

/***
 * @param {string} href
 * @returns {string|null}
 */
function findYouTubeVideoID(href) {
    try {
        const url = new URL(href)
        if ((url.hostname === "www.youtube.com" || url.hostname === "m.youtube.com") && url.pathname === "/watch" && url.searchParams.has("v")) {
            return decodeURIComponent(url.searchParams.get("v"))
        }

        if (url.hostname === "youtu.be" && url.pathname.startsWith("/") && url.pathname !== "/") {
            return decodeURIComponent(url.pathname.substr(1))
        }

        if (url.hostname === "www.youtube.com" && url.pathname.startsWith("/embed/") && url.pathname !== "/embed/") {
            return decodeURIComponent(url.pathname.substr(7))
        }
    } catch (_) { }
    return null
}

function findCoubVideoID(href) {
    try {
        const url = new URL(href)
        if (url.hostname !== "coub.com") {
            return null
        }

        const parts = url.pathname.split("/")
        // /view/{id}
        // /embed/{id}
        if (parts.length === 3 && (parts[1] === "view" || parts[1] === "embed")) {
            return decodeURIComponent(parts[2])
        }
    } catch (_) { }
    return null
}

function isSoundCloudURL(href) {
    try {
        const url = new URL(href)
        if (url.hostname === "soundcloud.com") {
            return true
        }

        // /{username}/{slug}
        const parts = url.pathname.split("/")
        return parts.length == 3
    } catch (_) { }
    return false
}

function getSoundCloudIframe(url) {
    return fetch(`https://soundcloud.com/oembed?format=json&url=${encodeURIComponent(url)}`).then(parseResponse).then(body => {
        const tmpl = document.createElement("template")
        tmpl.innerHTML = body.html
        return tmpl.content.querySelector("iframe")
    })
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

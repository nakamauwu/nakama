const mentionsRegExp = /\B@([a-zA-Z][a-zA-Z0-9_-]{0,17})/g
const urlsRegExp = /\b(https?:\/\/[\-A-Za-z0-9+&@#\/%?=~_|!:,\.;]*[\-A-Za-z0-9+&@#\/%=~_|])/gi

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

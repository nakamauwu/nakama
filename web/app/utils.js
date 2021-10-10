const mentionsRegExp = /\B@([a-zA-Z][a-zA-Z0-9_-]{0,17})(\b[^@]|$)/g
const tagsRegExp = /\B#((?:\p{L}|\p{N}|_)+)(\b[^#]|$)/gu
const urlsRegExp = /\b(https?:\/\/[\-A-Za-z0-9+&@#\/%?=~_|!:,\.;]*[\-A-Za-z0-9+&@#\/%=~_|])/gi

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
        .replace(mentionsRegExp, '<a href="/@$1">@$1</a>$2')
        .replace(tagsRegExp, '<a href="/tagged-posts/$1">#$1</a>$2')
        .replace(urlsRegExp, '<a href="$1" target="_blank" rel="noopener">$1</a>')
}

/**
 * @param {string} s
 */
export function collectMediaURLs(s) {
    const out = []
    for (const match of s.matchAll(urlsRegExp)) {
        if (match !== null && match.length >= 2) {
            try {
                const url = new URL(match[1])
                out.push(url)
            } catch (_) { }
        }
    }
    return out
}

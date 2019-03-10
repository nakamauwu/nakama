export function isObject(x) {
    return typeof x === 'object' && x !== null
}

export function isPlainObject(x) {
    return isObject(x) && !Array.isArray(x)
}

/**
 * @param {string} s
 */
export function escapeHTML(s) {
    return s
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;')
}

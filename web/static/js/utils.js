export function isObject(x) {
    return typeof x === 'object' && x !== null
}

export function isPlainObject(x) {
    return isObject(x) && !Array.isArray(x)
}

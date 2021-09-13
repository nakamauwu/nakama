import { getLocalAuth } from "./auth.js"

/**
 * @param {string} method
 * @param {string} url
 */
export function request(method, url, { body = undefined, headers = undefined } = {}) {
    if (!(body instanceof FormData) && !(body instanceof File) && typeof body === "object" && body !== null) {
        body = JSON.stringify(body)
    }
    return fetch(url, {
        method,
        headers: Object.assign(detaultHeaders(), headers),
        credentials: "include",
        body,
    }).then(handleResponse)
}

function detaultHeaders() {
    const auth = getLocalAuth()
    if (auth === null) {
        return {}
    }

    return {
        authorization: "Bearer " + auth.token,
    }
}

/**
 * @param {string} url
 * @param {function} cb
 */
export function subscribe(url, cb) {
    const auth = getLocalAuth()
    if (auth !== null) {
        const u = new URL(url, location.origin)
        u.searchParams.set("auth_token", auth.token)
        url = u.toString()
    }

    const onMessage = ev => {
        try {
            const data = JSON.parse(ev.data)
            cb(data)
        } catch (_) { }
    }

    const noop = () => { }

    const es = new EventSource(url, { withCredentials: true })
    es.addEventListener("message", onMessage)
    es.addEventListener("error", noop)

    return () => {
        es.removeEventListener("message", onMessage)
        es.removeEventListener("error", noop)
        es.close()
    }
}

/**
 * @param {Response} resp
 */
export function handleResponse(resp) {
    return resp.clone().json().catch(() => resp.text()).then(body => {
        if (!resp.ok) {
            const err = new Error()
            if (typeof body === "string" && body.trim() !== "") {
                err.message = body.trim()
            } else if (typeof body === "object" && body !== null && typeof body.error === "string") {
                err.message = body.error
            } else {
                err.message = resp.statusText
            }
            err.name = err.message
                .split(" ")
                .map(word => word.charAt(0).toUpperCase() + word.slice(1))
                .join("")
            if (!err.name.endsWith("Error")) {
                err.name = err.name + "Error"
            }
            err["headers"] = resp.headers
            err["statusCode"] = resp.status
            throw err
        }
        return {
            body,
            headers: resp.headers,
            statusCode: resp.status,
        }
    })
}

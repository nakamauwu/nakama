import { component, useEffect, useState } from "haunted"
import { html } from "lit"
import { get as getTranslation } from "lit-translate"

const ms = 1000 * 60

function RelativeDateTime({ datetime }) {
    const [ago, setAgo] = useState(shortHumanDuration(datetime))

    useEffect(() => {
        const id = setInterval(() => {
            setAgo(shortHumanDuration(datetime))
        }, ms)

        return () => {
            clearInterval(id)
        }
    }, [])

    return html`
        <time datetime="${datetime.toJSON()}" title="${datetime.toLocaleString()}">${ago}</time>
    `
}

// @ts-ignore
customElements.define("relative-datetime", component(RelativeDateTime, { useShadowDOM: false }))

/**
 * @param {Date} t
 * @param {Date} now
 */
function shortHumanDuration(t, now = new Date()) {
    const rtf = new Intl.RelativeTimeFormat(document.documentElement.lang, {
        numeric: "auto",
        style: "short",
    })

    const secs = (now.valueOf() - t.valueOf()) / 1000

    if (secs <= 1) {
        return getTranslation("relativeDateTime.now")
    }

    const mins = is(60, secs)
    const hours = is(60, mins)
    const days = is(24, hours)

    if (is(7, days) > 0) {
        return t.toLocaleDateString()
    }

    if (days > 0) {
        return rtf.format(days * -1, "days")
    }

    if (hours > 0) {
        return rtf.format(hours * -1, "hours")
    }

    if (mins > 0) {
        return rtf.format(mins * -1, "minutes")
    }

    return rtf.format(Math.floor(secs) * -1, "seconds")
}

function is(interval, cycle) {
    return cycle >= interval ? Math.floor(cycle / interval) : 0
}

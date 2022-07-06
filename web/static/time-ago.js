class TimeAgo extends HTMLElement {
    connectedCallback() {
        const el = this.querySelector("time")
        if (el === null || el.dateTime === "") {
            return
        }

        el.textContent = ago(el.dateTime)
    }
}

customElements.define("time-ago", TimeAgo)

function ago(text) {
    const now = new Date()
    const date = new Date(text)
    let diff = (now.getTime() - date.getTime()) / 1000
    if (diff <= 60) {
        return 'Just now'
    }
    if ((diff /= 60) < 60) {
        return (diff | 0) + 'm'
    }
    if ((diff /= 60) < 24) {
        return (diff | 0) + 'h'
    }
    if ((diff /= 24) < 7) {
        return (diff | 0) + 'd'
    }
    text = String(date).split(' ')[1] + ' ' + date.getDate()
    if (diff > 182 && now.getFullYear() !== date.getFullYear()) {
        return text + ', ' + date.getFullYear()
    }
    return text
}

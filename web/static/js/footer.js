import { translate } from "./i18n/i18n.js"

const tmpl = document.createElement("template")
tmpl.innerHTML = `
    <div class="container">
        <p>
            <span data-node="msg-prefix"></span>
            <a href="https://github.com/nicolasparada/nakama"
                target="_blank"
                rel="noopener
                noreferrer">github.com/nicolasparada/nakama</a>.
        </p>
    </div>
`

export function footerView(footer) {
    const frag = /** @type {DocumentFragment} */ (tmpl.content.cloneNode(true))
    const msgPrefixNode = frag.querySelector(`[data-node="msg-prefix"]`)

    let msgPrefix

    /**
     * @param {string} text
     */
    const setMsgPrefixNode = text => {
        msgPrefixNode.textContent = text
    }

    /**
     * @param {string} val
     */
    const setMsgPrefix = val => {
        if (msgPrefix !== val) {
            msgPrefix = val
            setMsgPrefixNode(val)
        }
    }

    setMsgPrefix(translate("footer.msgPrefix"))

    const update = () => frag

    return update
}

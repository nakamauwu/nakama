const template = document.createElement("template")
template.innerHTML = `
    <div class="container">
        <h1>Error 😕</h1>
        <p class="error">Something went wrong: <span id="error-span"></span></p>
        <button id="reload-button">Reload page</button>
        <div class="error-help">
            <em>If the problem persists, try doing a hard reload</em>
            <button id="hard-reload-button">Hard reload</button>
        </div>
    </div>
`

/**
 * @param {Error} err
 */
export default function renderErrorPage(err) {
    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const errorSpan = /** @type {HTMLSpanElement} */ (page.getElementById("error-span"))
    const reloadButton = /** @type {HTMLButtonElement} */ (page.getElementById("reload-button"))
    const hardReloadButton = /** @type {HTMLButtonElement} */ (page.getElementById("hard-reload-button"))

    const onReloadButtonClick = () => {
        reloadButton.disabled = true
        location.reload()
    }

    const onHardReloadButtonClick = () => {
        hardReloadButton.disabled = true
        localStorage.clear()
        location.assign("/")
    }

    errorSpan.textContent = err.message
    reloadButton.addEventListener("click", onReloadButtonClick, { once: true })
    hardReloadButton.addEventListener("click", onHardReloadButtonClick, { once: true })

    return page
}

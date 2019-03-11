const template = document.createElement('template')
template.innerHTML = `
    <div class="container">
        <h1>Error 😕</h1>
        <p>Something went wrong: <span id="error-span"></span></p>
        <button id="reload-button">Reload page</button>
    </div>
`

/**
 * @param {Error} err
 */
export default function renderErrorPage(err) {
    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const errorSpan = /** @type {HTMLSpanElement} */ (page.getElementById('error-span'))
    const reloadButton = /** @type {HTMLButtonElement} */ (page.getElementById('reload-button'))

    const onReloadButtonClick = () => {
        reloadButton.disabled = true
        location.reload()
    }

    errorSpan.textContent = err.message
    reloadButton.addEventListener('click', onReloadButtonClick, { once: true })

    return page
}

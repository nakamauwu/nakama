import { thumbHashToDataURL } from "https://unpkg.com/thumbhash@0.1.1/thumbhash.js"

void function thumbHashImages() {
    /**
     * @type {NodeListOf<HTMLImageElement>}
     */
    const imgs = document.querySelectorAll("img[data-thumbhash]")
    for (const img of imgs) {
        thumbHashImage(img)
    }
}()

/**
 * @param {HTMLImageElement} img
 */
function thumbHashImage(img) {
    if (img.complete) {
        return
    }

    const b64 = img.getAttribute("data-thumbhash")
    if (b64 === null) {
        return
    }

    const decoded = atob(b64)
    const hash = Uint8Array.from(decoded, c => c.charCodeAt(0))

    const originalSrc = img.src

    img.addEventListener("load", () => {
        img.src = originalSrc
    }, { once: true })

    img.src = thumbHashToDataURL(hash)
    img.removeAttribute("data-thumbhash")
}

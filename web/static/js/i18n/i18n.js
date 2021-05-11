import { registerTranslateConfig } from "https://cdn.skypack.dev/pin/lit-translate@v1.2.1-oIF5mWhCoEj61eOjUiCC/mode=imports,min/optimized/lit-translate.js"
export { get as translate, use as useLang } from "https://cdn.skypack.dev/pin/lit-translate@v1.2.1-oIF5mWhCoEj61eOjUiCC/mode=imports,min/optimized/lit-translate.js"

void function initi18n() {
    registerTranslateConfig({
        loader: lang => fetch(`/js/i18n/${lang}.json`).then(res => res.json())
    })
}()

export function detectLang() {
    const preferredLang = localStorage.getItem("preferred_lang")
    if (preferredLang === "es") {
        return "es"
    }

    if (Array.isArray(window.navigator.languages)) {
        for (const lang of window.navigator.languages) {
            if (isSpanish(lang)) {
                return "es"
            }
        }
    }

    if (isSpanish(window.navigator["userLanguage"])) {
        return "es"
    }

    if (isSpanish(window.navigator.language)) {
        return "es"
    }

    return "en"
}

function isSpanish(lang) {
    return lang === "es" || (typeof lang === "string" && lang.startsWith("es-"))
}

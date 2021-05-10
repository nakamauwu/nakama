import { registerTranslateConfig } from "https://cdn.skypack.dev/pin/lit-translate@v1.2.1-oIF5mWhCoEj61eOjUiCC/mode=imports,min/optimized/lit-translate.js"
export { get as translate, use as useLang } from "https://cdn.skypack.dev/pin/lit-translate@v1.2.1-oIF5mWhCoEj61eOjUiCC/mode=imports,min/optimized/lit-translate.js"

registerTranslateConfig({
    loader: lang => fetch(`/js/i18n/${lang}.json`).then(res => res.json())
})

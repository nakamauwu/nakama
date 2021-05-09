if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register("/sw.js", { updateViaCache: "none" })
}

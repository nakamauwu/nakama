/**
 * @returns {Promise<ServiceWorker>}
 */
function registerSW() {
    if (!("serviceWorker" in navigator)) {
        return Promise.reject(new Error("no sw support"))
    }

    return new Promise(resolve => {
        navigator.serviceWorker.register("/sw.js", { updateViaCache: "none" }).then(reg => {
            const onUpdateFound = () => {
                const worker = reg.installing
                if (worker === null) {
                    return
                }

                const onStateChange = () => {
                    if (worker.state !== "installed" || navigator.serviceWorker.controller === null) {
                        return
                    }

                    resolve(worker)
                }
                worker.onstatechange = onStateChange
            }
            reg.onupdatefound = onUpdateFound
        })
    })
}


if ("serviceWorker" in navigator) {
    let reloading = false
    const onCtrlChange = () => {
        if (reloading) {
            return
        }
        reloading = true
        const ok = confirm("New version of nakama available. Refresh to update?")
        if (!ok) {
            return
        }
        location.reload()
    }
    navigator.serviceWorker.oncontrollerchange = onCtrlChange

    registerSW().then(worker => {
        worker.postMessage({ action: "skipWaiting" })
    })
}

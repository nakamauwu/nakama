import { component, html, useCallback, useEffect, useState } from "haunted"
import { nothing } from "lit-html"

function ToastItem({ toast: initialToast }) {
    const [toast, setToast] = useState(Object.assign({ type: "", content: "", timeout: 5000 }, initialToast))
    const [show, setShow] = useState(true)

    const onClick = useCallback(() => {
        setShow(false)
    }, [toast])

    useEffect(() => {
        const id = setTimeout(() => {
            setShow(false)
        }, toast.timeout)

        return () => {
            clearTimeout(id)
        }
    }, [toast])

    useEffect(() => {
        setToast(Object.assign({ type: "", content: "", timeout: 5000 }, initialToast))
        setShow(true)
    }, [initialToast])

    if (!show) {
        return nothing
    }

    return html`
        <div class="toast${toast.type !== "" ? " " + toast.type : ""}" role="status" aria-live="polite" @click=${onClick}>
            <p>${toast.content}</p>
        </div>
    `
}

// @ts-ignore
customElements.define("toast-item", component(ToastItem, { useShadowDOM: false }))

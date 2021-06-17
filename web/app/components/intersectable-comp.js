import { component, html, useEffect, useRef } from "haunted"
import { ref } from "../directives/ref.js"

function IntersectableComp() {
    const nodeRef = useRef(null)

    const dispatchIsIntersecting = () => {
        this.dispatchEvent(new CustomEvent("is-intersecting", { bubbles: true }))
    }

    useEffect(() => {
        if (nodeRef.current === null) {
            return
        }

        const obs = new IntersectionObserver(([entry]) => {
            if (!entry.isIntersecting) {
                return
            }

            dispatchIsIntersecting()
        })

        obs.observe(nodeRef.current)
        return () => {
            obs.disconnect()
        }
    }, [nodeRef])

    return html`<div .ref=${ref(nodeRef)}></div>`
}

customElements.define("intersectable-comp", component(IntersectableComp, { useShadowDOM: false }))

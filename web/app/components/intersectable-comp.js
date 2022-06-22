import { component, useEffect } from "haunted"
import { html } from "lit"
import { createRef, ref } from "lit/directives/ref.js"

function IntersectableComp() {
    const nodeRef = createRef()

    const dispatchIsIntersecting = () => {
        this.dispatchEvent(new CustomEvent("is-intersecting", { bubbles: true }))
    }

    useEffect(() => {
        if (nodeRef.value === undefined) {
            return
        }

        const obs = new IntersectionObserver(([entry]) => {
            if (!entry.isIntersecting) {
                return
            }

            dispatchIsIntersecting()
        })

        obs.observe(nodeRef.value)
        return () => {
            obs.disconnect()
        }
    }, [nodeRef.value])

    return html`<div ${ref(nodeRef)}></div>`
}

customElements.define("intersectable-comp", component(IntersectableComp, { useShadowDOM: false }))

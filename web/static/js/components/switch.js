/**
 * @param {boolean} checked
 * @param {string} label
 * @param {function(boolean):Promise<{checked:boolean,label:string}>} onChange
 */
export function renderSwitch(checked, label, onChange) {
    const btn = document.createElement("button")
    btn.setAttribute("role", "switch")

    const span = document.createElement("span")
    btn.appendChild(span)

    const update = (checked, label) => {
        btn.setAttribute("aria-checked", String(checked))
        btn.setAttribute("aria-label", label)
    }
    update(checked, label)

    btn.onclick = async function onSwitchClick() {
        btn.disabled = true
        const { checked: newValue, label } = await onChange(btn.getAttribute("aria-checked") !== "true")

        checked = newValue
        update(checked, label)
        btn.disabled = false
    }

    return { el: btn, update }
}

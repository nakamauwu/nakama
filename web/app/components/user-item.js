import { component, html, useCallback, useEffect, useState } from "haunted"
import { nothing } from "lit-html"
import { ifDefined } from "lit-html/directives/if-defined"
import { authStore, useStore } from "../ctx.js"
import { Avatar } from "./avatar.js"
import "./user-follow-btn.js"
import "./user-follow-counts.js"

function UserItem({ user: initialUser }) {
    const [auth] = useStore(authStore)
    const [user, setUser] = useState(initialUser)

    const onFollowToggle = useCallback(ev => {
        const payload = ev.detail
        setUser(u => ({
            ...u,
            ...payload,
        }))
    }, [user])

    useEffect(() => {
        setUser(initialUser)
    }, [initialUser])

    return html`
        <article class="user-item" style="${ifDefined(user.coverURL !== null ? `--cover-url: url('${user.coverURL}');` : undefined)}">
            <a href="/@${user.username}" class="user-info">
                ${Avatar(user)}
                <div class="user-text">
                    <span class="username">${user.username}</span>
                    <user-follow-counts .user=${user}></user-follow-counts>
                </div>
            </a>
            ${auth !== null && !user.me ? html`
                <user-follow-btn .user=${user} @follow-toggle=${onFollowToggle}></user-follow-btn>
            ` : nothing}
        </article>
    `
}

// @ts-ignore
customElements.define("user-item", component(UserItem, { useShadowDOM: false }))

import { component, html } from "haunted"

function UserFollowCounts({ user }) {
    return html`
        <div class="user-counts">
            <a href="/@${user.username}/followers">
                <span>${user.followersCount}</span>
                <span class="label">followers</span>
            </a>
            <a href="/@${user.username}/followees">
                <span>${user.followeesCount}</span>
                <span class="label">followees</span>
            </a>
        </div>
    `
}

// @ts-ignore
customElements.define("user-follow-counts", component(UserFollowCounts, { useShadowDOM: false }))

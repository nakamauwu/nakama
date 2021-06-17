import { component, html } from "haunted"
import { repeat } from "lit-html/directives/repeat.js"
import { useFetch } from "../fetch.js"
import "./user-item.js"

export default function ({ params }) {
    return html`<user-followees-page .username=${params.username}></user-followees-page>`
}

function UserFolloweesPage({ username }) {
    const usersState = useFetch(() => fetchFollowees(username), [username])

    return html`
        <main class="container followees-page">
            <h1>${username}'s Followees</h1>
            ${usersState.err !== null ? html`
                <p class="error" role="alert">Could not fetch followees: ${usersState.err.message}</p>
            ` : usersState.isFetching ? html`
                <p class="loader" aria-busy="true" aria-live="polite">Loading followees... please wait.<p>
            ` : html`
                ${usersState.data.length === 0 ? html`
                    <p>0 followees</p>
                ` : html`
                    <div class="users" role="feed">
                        ${repeat(usersState.data, u => u.id, u => html`<user-item .user=${u}></user-item>`)}
                    </div>
                `}
            `}
        </main>
    `
}

// @ts-ignore
customElements.define("user-followees-page", component(UserFolloweesPage, { useShadowDOM: false }))

function fetchFollowees(username) {
    return new Promise(resolve => {
        setTimeout(() => {
            resolve([
                {
                    id: String(Date.now()) + Math.random().toString(36).substring(7),
                    username: "user",
                    avatarURL: null,
                    coverURL: "https://picsum.photos/1920/400?random=" + String(Date.now()) + Math.random().toString(36).substring(7),
                    followersCount: 0,
                    followeesCount: 0,
                    following: false,
                    me: false,
                },
                {
                    id: String(Date.now()) + Math.random().toString(36).substring(7),
                    username: "user2",
                    avatarURL: null,
                    coverURL: "https://picsum.photos/1920/400?random=" + String(Date.now()) + Math.random().toString(36).substring(7),
                    followersCount: 1,
                    followeesCount: 0,
                    following: true,
                    me: false,
                },
                {
                    id: String(Date.now()) + Math.random().toString(36).substring(7),
                    username: "user3",
                    avatarURL: null,
                    coverURL: null,
                    followersCount: 0,
                    followeesCount: 0,
                    following: false,
                    me: false,
                },
                {
                    id: String(Date.now()) + Math.random().toString(36).substring(7),
                    username: "user4",
                    avatarURL: null,
                    coverURL: "https://picsum.photos/1080/320?random=" + String(Date.now()) + Math.random().toString(36).substring(7),
                    followersCount: 0,
                    followeesCount: 0,
                    following: false,
                    me: false,
                },
            ])
        }, 500)
    })
}

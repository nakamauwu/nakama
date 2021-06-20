import { component, html, useCallback, useEffect, useState } from "haunted"
import { nothing } from "lit-html"
import { repeat } from "lit-html/directives/repeat.js"
import { request } from "../http.js"
import "./intersectable-comp.js"
import "./toast-item.js"
import "./user-item.js"

const pageSize = 3

export default function ({ params }) {
    return html`<user-followers-page .username=${params.username}></user-followers-page>`
}

function UserFollowersPage({ username }) {
    const [users, setUsers] = useState([])
    const [usersEndCursor, setUsersEndCursor] = useState(null)
    const [fetching, setFetching] = useState(true)
    const [err, setErr] = useState(null)
    const [loadingMore, setLoadingMore] = useState(false)
    const [noMoreUsers, setNoMoreUsers] = useState(false)
    const [endReached, setEndReached] = useState(false)
    const [toast, setToast] = useState(null)

    const loadMore = useCallback(() => {
        if (loadingMore || noMoreUsers) {
            return
        }

        setLoadingMore(true)
        fetchFollowers(username, usersEndCursor).then(({ items: users, endCursor }) => {
            setUsers(uu => [...uu, ...users])
            setUsersEndCursor(endCursor)

            if (users.length < pageSize) {
                setNoMoreUsers(true)
                setEndReached(true)
            }
        }, err => {
            const msg = "could not fetch more users: " + err.message
            console.error(msg)
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setLoadingMore(false)
        })
    }, [loadingMore, noMoreUsers, username, usersEndCursor])

    useEffect(() => {
        setFetching(true)
        fetchFollowers(username).then(({ items: users, endCursor }) => {
            setUsers(users)
            setUsersEndCursor(endCursor)

            if (users.length < pageSize) {
                setNoMoreUsers(true)
            }
        }, err => {
            console.error("could not fetch users:", err)
            setErr(err)
        }).finally(() => {
            setFetching(false)
        })
    }, [username])

    return html`
        <main class="container followers-page">
            <h1>${username}'s Followers</h1>
            ${err !== null ? html`
                <p class="error" role="alert">Could not fetch followers: ${err.message}</p>
            ` : fetching ? html`
                <p class="loader" aria-busy="true" aria-live="polite">Loading followers... please wait.<p>
            ` : html`
                ${users.length === 0 ? html`
                    <p>0 followers</p>
                ` : html`
                    <div class="users" role="feed">
                        ${repeat(users, u => u.id, u => html`<user-item .user=${u}></user-item>`)}
                    </div>
                    ${!noMoreUsers ? html`
                        <intersectable-comp @is-intersecting=${loadMore}></intersectable-comp>
                        <p class="loader" aria-busy="true" aria-live="polite">Loading users... please wait.<p>
                    ` : endReached ? html`
                        <p>End reached.</p>
                    ` : nothing}
                `}
            `}
        </main>
        ${toast !== null ? html`<toast-item .toast=${toast}></toast-item>` : nothing}
    `
}

// @ts-ignore
customElements.define("user-followers-page", component(UserFollowersPage, { useShadowDOM: false }))

function fetchFollowers(username, after = "", first = pageSize) {
    return request("GET", `/api/users/${encodeURIComponent(username)}/followers?after=${encodeURIComponent(after)}&first=${encodeURIComponent(first)}`)
        .then(resp => resp.body)
}

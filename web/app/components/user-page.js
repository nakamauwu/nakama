import { component, useCallback, useEffect, useRef, useState } from "haunted"
import { html, nothing } from "lit-html"
import { ifDefined } from "lit-html/directives/if-defined"
import { repeat } from "lit-html/directives/repeat.js"
import { setLocalAuth } from "../auth.js"
import { authStore, useStore } from "../ctx.js"
import { ref } from "../directives/ref.js"
import { request } from "../http.js"
import { navigate } from "../router.js"
import { Avatar } from "./avatar.js"
import "./intersectable-comp.js"
import "./post-item.js"
import "./toast-item.js"
import "./user-follow-btn.js"
import "./user-follow-counts.js"

const pageSize = 10

export default function ({ params }) {
    return html`<user-page .username=${params.username}></user-page>`
}

function UserPage({ username }) {
    const [_, setAuth] = useStore(authStore)
    const [user, setUser] = useState(null)
    const [posts, setPosts] = useState([])
    const [postsEndCursor, setPostsEndCursor] = useState(null)
    const [fetching, setFetching] = useState(user === null)
    const [err, setErr] = useState(null)
    const [loadingMore, setLoadingMore] = useState(false)
    const [noMorePosts, setNoMorePosts] = useState(false)
    const [endReached, setEndReached] = useState(false)
    const [toast, setToast] = useState(null)

    const onPostDeleted = useCallback(ev => {
        const payload = ev.detail
        setPosts(pp => pp.filter(p => p.id !== payload.id))
    }, [])

    const onUsernameUpdated = useCallback(ev => {
        updateUser(ev.detail)
    }, [])

    const onAvatarUpdated = useCallback(ev => {
        updateUser(ev.detail)
    }, [])

    const updateUser = useCallback(payload => {
        setUser(u => ({
            ...u,
            ...payload,
        }))
        setPosts(pp => pp.map(p => ({
            ...p,
            user: {
                ...p.user,
                ...payload,
            }
        })))
        setAuth(auth => {
            const newAuth = {
                ...auth,
                user: {
                    ...auth.user,
                    ...payload,
                }
            }
            setLocalAuth(newAuth)
            return newAuth
        })
    }, [])

    const loadMore = useCallback(() => {
        if (loadingMore || noMorePosts) {
            return
        }

        setLoadingMore(true)
        fetchPosts(username, postsEndCursor).then(({ items: posts, endCursor }) => {
            for (let i = 0; i < posts.length; i++) {
                posts[i].user = user
            }

            setPosts(pp => [...pp, ...posts])
            setPostsEndCursor(endCursor)

            if (posts.length < pageSize) {
                setNoMorePosts(true)
                setEndReached(true)
            }
        }, err => {
            const msg = "could not fetch more posts: " + err.message
            console.error(msg)
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setLoadingMore(false)
        })
    }, [loadingMore, noMorePosts, username, postsEndCursor, user])

    useEffect(() => {
        setFetching(true)
        Promise.all([
            fetchUser(username),
            fetchPosts(username),
        ]).then(([user, { items: posts, endCursor }]) => {
            for (let i = 0; i < posts.length; i++) {
                posts[i].user = user
            }

            setUser(user)
            setPosts(posts)
            setPostsEndCursor(endCursor)

            if (posts.length < pageSize) {
                setNoMorePosts(true)
            }
        }, err => {
            console.error("could not fetch user and posts:", err)
            setErr(err)
        }).finally(() => {
            setFetching(false)
        })
    }, [username])

    return html`
        <main class="user-page">
            <div class="user-profile-wrapper"
                style="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx">
                <div class="container">
                    ${err !== null ? html`
                    <p class="error" role="alert">Could not fetch user: ${err.message}</p>
                    ` : fetching ? html`
                    <p class="loader" aria-busy="true" aria-live="polite">Loading user... please wait.<p>
                            ` : html`
                            <user-profile .user=${user} @username-updated=${onUsernameUpdated}
                                @avatar-updated=${onAvatarUpdated}></user-profile>
                            `}
                </div>
            </div>
            <div class="container posts-wrapper">
                <h2>Posts</h2>
                ${err !== null ? html`
                <p class="error" role="alert">Could not fetch posts: ${err.message}</p>
                ` : fetching ? html`
                <p class="loader" aria-busy="true" aria-live="polite">Loading posts... please wait.<p>
                        ` : html`
                        ${posts.length === 0 ? html`
                        <p>0 posts</p>
                        ` : html`
                        <div class="posts" role="feed">
                            ${repeat(posts, p => p.id, p => html`<post-item .post=${p} .type=${"post"}
                                @resource-deleted=${onPostDeleted}></post-item>`)}
                        </div>
                        ${!noMorePosts ? html`
                        <intersectable-comp @is-intersecting=${loadMore}></intersectable-comp>
                        <p class="loader" aria-busy="true" aria-live="polite">Loading posts... please wait.</p>
                        ` : endReached ? html`
                        <p>End reached.</p>
                        ` : nothing}
                        `}
                        `}
            </div>
        </main>
        ${toast !== null ? html`<toast-item .toast=${toast}></toast-item>` : nothing}
    `
}

// @ts-ignore
customElements.define("user-page", component(UserPage, { useShadowDOM: false }))

function UserProfile({ user: initialUser }) {
    const [auth, setAuth] = useStore(authStore)
    const [user, setUser] = useState(initialUser)
    const [username, setUsername] = useState(user.username)
    const settingsDialogRef = useRef(null)
    const avatarInputRef = useRef(null)
    const [updatingUsername, setUpdatingUsername] = useState(false)
    const [updatingAvatar, setUpdatingAvatar] = useState(false)
    const [toast, setToast] = useState(null)

    const dispatchUsernameUpdated = payload => {
        this.dispatchEvent(new CustomEvent("username-updated", { bubbles: true, detail: payload }))
    }

    const dispatchAvatarUpdated = payload => {
        this.dispatchEvent(new CustomEvent("avatar-updated", { bubbles: true, detail: payload }))
    }

    const onFollowToggle = useCallback(ev => {
        const payload = ev.detail
        setUser(u => ({
            ...u,
            ...payload,
        }))
    }, [user])

    const onSettingsBtnClick = useCallback(() => {
        settingsDialogRef.current.showModal()
    }, [settingsDialogRef])

    const onUsernameInput = useCallback(ev => {
        setUsername(ev.currentTarget.value)
    }, [])

    const onUsernameFormSubmit = useCallback(ev => {
        ev.preventDefault()

        setUpdatingUsername(true)
        updateUser({ username }).then(payload => {
            setAuth(auth => ({
                ...auth,
                user: {
                    ...auth.user,
                    ...payload,
                },
            }))
            setUser(u => ({
                ...u,
                ...payload,
            }))
            setToast({ type: "success", content: "username updated" })
            history.replaceState(history.state, document.title, "/@" + encodeURIComponent(payload.username))
            dispatchUsernameUpdated(payload)
        }, err => {
            const msg = "could not update username: " + err.message
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setUpdatingUsername(false)
        })
    }, [username])

    const onAvatarInputChange = useCallback(ev => {
        const files = ev.currentTarget.files
        if (!(files instanceof window.FileList) || files.length !== 1) {
            return
        }

        const avatar = files.item(0)
        submitAvatar(avatar)
    }, [])

    const onAvatarDblClick = useCallback(() => {
        if (updatingAvatar) {
            return
        }

        avatarInputRef.current.click()
    }, [updatingAvatar, avatarInputRef])

    const onAvatarBtnClick = useCallback(() => {
        if (updatingAvatar) {
            return
        }

        avatarInputRef.current.click()
    }, [updatingAvatar, avatarInputRef])

    const onAvatarDragOver = useCallback(ev => {
        ev.preventDefault()
    }, [])

    const onAvatarDrop = useCallback(ev => {
        ev.preventDefault()
        if (updatingAvatar) {
            return
        }

        const files = ev.dataTransfer.files
        if (!(files instanceof FileList) || files.length !== 1) {
            return
        }

        const avatar = files.item(0)
        submitAvatar(avatar)
    }, [updatingAvatar])

    const submitAvatar = useCallback(avatar => {
        setUpdatingAvatar(true)
        updateAvatar(avatar).then(payload => {
            setAuth(auth => ({
                ...auth,
                user: {
                    ...auth.user,
                    ...payload,
                },
            }))
            setUser(u => ({
                ...u,
                ...payload,
            }))
            setToast({ type: "success", content: "avatar updated" })
            dispatchAvatarUpdated(payload)
        }, err => {
            const msg = "could not update avatar: " + err.message
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setUpdatingAvatar(false)
        })
    }, [])

    const onSettingsDialogCloseBtnClick = useCallback(() => {
        settingsDialogRef.current.close()
    }, [settingsDialogRef])

    const onSettingsDialogClose = useCallback(() => {
        setUsername(user.username)
    }, [user])

    useEffect(() => {
        if (settingsDialogRef.current !== null && !(window.HTMLDialogElement || settingsDialogRef.current.showModal)) {
            import("dialog-polyfill").then(m => m.default).then(dialogPolyfill => {
                dialogPolyfill.registerDialog(settingsDialogRef.current)
            })
        }
    }, [settingsDialogRef])

    useEffect(() => {
        setUser(initialUser)
        setUsername(initialUser.username)
    }, [initialUser])

    return html`
        <div class="user-profile">
            <h1>${user.username}</h1>
            <user-follow-counts .user=${user}></user-follow-counts>
            ${Avatar(user)}
            <div class="user-controls">
                ${user.me ? html`
                <button @click=${onSettingsBtnClick}>
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                        <g data-name="Layer 2">
                            <g data-name="settings">
                                <rect width="24" height="24" opacity="0" />
                                <path
                                    d="M8.61 22a2.25 2.25 0 0 1-1.35-.46L5.19 20a2.37 2.37 0 0 1-.49-3.22 2.06 2.06 0 0 0 .23-1.86l-.06-.16a1.83 1.83 0 0 0-1.12-1.22h-.16a2.34 2.34 0 0 1-1.48-2.94L2.93 8a2.18 2.18 0 0 1 1.12-1.41 2.14 2.14 0 0 1 1.68-.12 1.93 1.93 0 0 0 1.78-.29l.13-.1a1.94 1.94 0 0 0 .73-1.51v-.24A2.32 2.32 0 0 1 10.66 2h2.55a2.26 2.26 0 0 1 1.6.67 2.37 2.37 0 0 1 .68 1.68v.28a1.76 1.76 0 0 0 .69 1.43l.11.08a1.74 1.74 0 0 0 1.59.26l.34-.11A2.26 2.26 0 0 1 21.1 7.8l.79 2.52a2.36 2.36 0 0 1-1.46 2.93l-.2.07A1.89 1.89 0 0 0 19 14.6a2 2 0 0 0 .25 1.65l.26.38a2.38 2.38 0 0 1-.5 3.23L17 21.41a2.24 2.24 0 0 1-3.22-.53l-.12-.17a1.75 1.75 0 0 0-1.5-.78 1.8 1.8 0 0 0-1.43.77l-.23.33A2.25 2.25 0 0 1 9 22a2 2 0 0 1-.39 0zM4.4 11.62a3.83 3.83 0 0 1 2.38 2.5v.12a4 4 0 0 1-.46 3.62.38.38 0 0 0 0 .51L8.47 20a.25.25 0 0 0 .37-.07l.23-.33a3.77 3.77 0 0 1 6.2 0l.12.18a.3.3 0 0 0 .18.12.25.25 0 0 0 .19-.05l2.06-1.56a.36.36 0 0 0 .07-.49l-.26-.38A4 4 0 0 1 17.1 14a3.92 3.92 0 0 1 2.49-2.61l.2-.07a.34.34 0 0 0 .19-.44l-.78-2.49a.35.35 0 0 0-.2-.19.21.21 0 0 0-.19 0l-.34.11a3.74 3.74 0 0 1-3.43-.57L15 7.65a3.76 3.76 0 0 1-1.49-3v-.31a.37.37 0 0 0-.1-.26.31.31 0 0 0-.21-.08h-2.54a.31.31 0 0 0-.29.33v.25a3.9 3.9 0 0 1-1.52 3.09l-.13.1a3.91 3.91 0 0 1-3.63.59.22.22 0 0 0-.14 0 .28.28 0 0 0-.12.15L4 11.12a.36.36 0 0 0 .22.45z"
                                    data-name="&lt;Group&gt;" />
                                <path
                                    d="M12 15.5a3.5 3.5 0 1 1 3.5-3.5 3.5 3.5 0 0 1-3.5 3.5zm0-5a1.5 1.5 0 1 0 1.5 1.5 1.5 1.5 0 0 0-1.5-1.5z" />
                            </g>
                        </g>
                    </svg>
                    <span>Settings</span>
                </button>
                <logout-btn></logout-btn>
                ` : auth !== null ? html`
                <user-follow-btn .user=${user} @follow-toggle=${onFollowToggle}></user-follow-btn>
                ` : nothing}
            </div>
        </div>
        <dialog class="user-settings-dialog" .ref=${ref(settingsDialogRef)} @close=${onSettingsDialogClose}>
            <div class="user-settings">
                <div class="user-settings-header">
                    <h2>Settings</h2>
                    <button class="close-btn" title="Close" @click=${onSettingsDialogCloseBtnClick}>
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                            <g data-name="Layer 2">
                                <g data-name="close">
                                    <rect width="24" height="24" transform="rotate(180 12 12)" opacity="0" />
                                    <path
                                        d="M13.41 12l4.3-4.29a1 1 0 1 0-1.42-1.42L12 10.59l-4.29-4.3a1 1 0 0 0-1.42 1.42l4.3 4.29-4.3 4.29a1 1 0 0 0 0 1.42 1 1 0 0 0 1.42 0l4.29-4.3 4.29 4.3a1 1 0 0 0 1.42 0 1 1 0 0 0 0-1.42z" />
                                </g>
                            </g>
                        </svg>
                    </button>
                </div>
                <form @submit=${onUsernameFormSubmit}>
                    <fieldset class="username-fieldset">
                        <legend>Username</legend>
                        <div class="username-grp">
                            <input type="text" name="username" placeholder="Username" required .value=${username}
                                .disabled=${updatingUsername} @input=${onUsernameInput}>
                            <button .disabled=${updatingUsername}>Update</button>
                        </div>
                    </fieldset>
                </form>
                <fieldset class="avatar-fieldset" @drop=${onAvatarDrop} @dragover=${onAvatarDragOver}>
                    <legend>Avatar</legend>
                    <div class="avatar-grp">
                        <div @dblclick=${onAvatarDblClick}>
                            ${Avatar(user)}
                        </div>
                        <input type="file" name="avatar" accept="image/png,image/jpeg" required hidden
                            .disabled=${updatingAvatar} .ref=${ref(avatarInputRef)} @change=${onAvatarInputChange}>
                        <button .disabled=${updatingAvatar} @click=${onAvatarBtnClick}>Update</button>
                    </div>
                </fieldset>
            </div>
            ${toast !== null ? html`<toast-item .toast=${toast}></toast-item>` : nothing}
        </dialog>
    `
}

// @ts-ignore
customElements.define("user-profile", component(UserProfile, { useShadowDOM: false }))

function LogoutBtn() {
    const [, setAuth] = useStore(authStore)

    const onClick = useCallback(() => {
        localStorage.removeItem("auth")
        setAuth(null)
        navigate("/")
    }, [])

    return html`
        <button @click=${onClick}>
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                <g data-name="Layer 2">
                    <g data-name="log-out">
                        <rect width="24" height="24" transform="rotate(90 12 12)" opacity="0" />
                        <path d="M7 6a1 1 0 0 0 0-2H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h2a1 1 0 0 0 0-2H6V6z" />
                        <path
                            d="M20.82 11.42l-2.82-4a1 1 0 0 0-1.39-.24 1 1 0 0 0-.24 1.4L18.09 11H10a1 1 0 0 0 0 2h8l-1.8 2.4a1 1 0 0 0 .2 1.4 1 1 0 0 0 .6.2 1 1 0 0 0 .8-.4l3-4a1 1 0 0 0 .02-1.18z" />
                    </g>
                </g>
            </svg>
            <span>Logout</span>
        </button>
    `
}

customElements.define("logout-btn", component(LogoutBtn, { useShadowDOM: false }))

/**
 * @param {string} username
 */
function fetchUser(username) {
    return request("GET", "/api/users/" + encodeURIComponent(username))
        .then(resp => resp.body)
}

/**
 * @param {string} username
 */
function fetchPosts(username, before = "", last = pageSize) {
    return request("GET", `/api/users/${encodeURIComponent(username)}/posts?last=${encodeURIComponent(last)}&before=${encodeURIComponent(before)}`)
        .then(resp => resp.body)
        .then(page => {
            page.items = page.items.map(p => ({
                ...p,
                createdAt: new Date(p.createdAt),
            }))
            return page
        })
}

function updateUser({ username }) {
    return request("PATCH", "/api/auth_user", { body: { username } })
        .then(resp => resp.body)
}

/**
 * @param {File} avatar
 */
function updateAvatar(avatar) {
    return request("PUT", "/api/auth_user/avatar", { body: avatar })
        .then(resp => resp.body)
        .then(avatarURL => ({ avatarURL }))
}

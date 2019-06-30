import { isAuthenticated } from "../auth.js"
import { doGet, doPost } from "../http.js"
import renderAvatarHTML from "./avatar.js"
import renderList from "./list.js"
import renderPost from "./post.js"
import { personDoneIconSVG, personAddIconSVG } from "./icons.js";
import { replaceNode, el } from "../utils.js";

const PAGE_SIZE = 10

const template = document.createElement("template")
template.innerHTML = `
    <div class="user-wrapper">
        <div class="container">
            <div id="user-outlet"></div>
        </div>
    </div>
    <div class="container">
        <h2>Posts</h2>
        <div id="posts-outlet" class="posts-wrapper"></div>
    </div>
`

export default async function renderUserPage(params) {
    const [user, posts] = await Promise.all([
        fetchUser(params.username),
        fetchPosts(params.username),
    ])
    for (const post of posts) {
        post.user = user
    }

    const loadMore = async before => {
        const posts = await fetchPosts(user.username, before)
        for (const post of posts) {
            post.user = user
        }
        return posts
    }

    const list = renderList({
        items: posts,
        loadMoreFunc: loadMore,
        pageSize: PAGE_SIZE,
        renderItem: renderPost,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const userOutlet = page.getElementById("user-outlet")
    const postsOutlet = page.getElementById("posts-outlet")

    userOutlet.appendChild(renderUserProfile(user))
    postsOutlet.appendChild(list.el)

    page.addEventListener("disconnect", list.teardown)

    return page
}

/**
 * @param {import("../types.js").UserProfile} user
 */
function renderUserProfile(user) {
    const authenticated = isAuthenticated()
    const div = document.createElement("div")
    div.className = "user-profile"
    div.innerHTML = `
        ${renderAvatarHTML(user)}
        <div>
            <div>
                <h1>${user.username}</h1>
                ${user.followeed ? `
                    <span class="badge">Follows you</span>
                ` : ""}
            </div>
            <div class="user-stats">
                <a href="/users/${user.username}/followers">
                    <span class="followers-count">${user.followersCount}</span>
                    <span class="label">followers</span>
                </a>
                <a href="/users/${user.username}/followees">
                    <span class="followees-count">${user.followeesCount}</span>
                    <span class="label">followees</span>
                </a>
            </div>
        </div>
        ${authenticated && !user.me ? `
            <div class="user-controls">
                <button class="follow-button" aria-pressed="${user.following}">
                    ${user.following ? personDoneIconSVG : personAddIconSVG}
                    <span>${user.following ? "Following" : "Follow"}</span>
                </button>
            </div>
        ` : user.me ? `
            <div class="user-controls">
                <button class="avatar-button">
                    <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="upload"><rect width="24" height="24" transform="rotate(180 12 12)" opacity="0"/><rect x="4" y="4" width="16" height="2" rx="1" ry="1" transform="rotate(180 12 5)"/><rect x="17" y="5" width="4" height="2" rx="1" ry="1" transform="rotate(90 19 6)"/><rect x="3" y="5" width="4" height="2" rx="1" ry="1" transform="rotate(90 5 6)"/><path d="M8 14a1 1 0 0 1-.8-.4 1 1 0 0 1 .2-1.4l4-3a1 1 0 0 1 1.18 0l4 2.82a1 1 0 0 1 .24 1.39 1 1 0 0 1-1.4.24L12 11.24 8.6 13.8a1 1 0 0 1-.6.2z"/><path d="M12 21a1 1 0 0 1-1-1v-8a1 1 0 0 1 2 0v8a1 1 0 0 1-1 1z"/></g></g></svg>
                    <span>Change avatar</span>
                </button>
                <button class="logout-button">
                    <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="log-out"><rect width="24" height="24" transform="rotate(90 12 12)" opacity="0"/><path d="M7 6a1 1 0 0 0 0-2H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h2a1 1 0 0 0 0-2H6V6z"/><path d="M20.82 11.42l-2.82-4a1 1 0 0 0-1.39-.24 1 1 0 0 0-.24 1.4L18.09 11H10a1 1 0 0 0 0 2h8l-1.8 2.4a1 1 0 0 0 .2 1.4 1 1 0 0 0 .6.2 1 1 0 0 0 .8-.4l3-4a1 1 0 0 0 .02-1.18z"/></g></g></svg>
                    <span>Logout</span>
                </button>
            </div>
        ` : ""}
    `

    const followersCountSpan = /** @type {HTMLSpanElement} */ (div.querySelector(".followers-count"))
    const followButton = /** @type {HTMLButtonElement=} */ (div.querySelector(".follow-button"))
    const logoutButton = /** @type {HTMLButtonElement=} */ (div.querySelector(".logout-button"))

    if (followButton !== null) {
        const followText = followButton.querySelector("span")
        const onFollowButtonClick = async () => {
            followButton.disabled = true

            try {
                const out = await toggleFollow(user.username)
                followersCountSpan.textContent = String(out.followersCount)
                followButton.setAttribute("aria-pressed", String(out.following))
                replaceNode(
                    followButton.querySelector("svg"),
                    el(out.following ? personDoneIconSVG : personAddIconSVG),
                )
                followText.textContent = out.following ? "Following" : "Follow"
            } catch (err) {
                console.log(err)
                alert(err.message)
            } finally {
                followButton.disabled = false
            }
        }

        followButton.addEventListener("click", onFollowButtonClick)
    }

    if (logoutButton !== null) {
        const onLogoutButtonClick = () => {
            logoutButton.disabled = true
            localStorage.clear()
            location.assign("/")
        }

        logoutButton.addEventListener("click", onLogoutButtonClick)
    }

    return div
}

/**
 * @param {string} username
 * @returns {Promise<import("../types.js").UserProfile>}
 */
function fetchUser(username) {
    return doGet("/api/users/" + username)
}

/**
 * @param {string} username
 * @param {bigint=} before
 * @returns {Promise<import("../types.js").Post[]>}
 */
function fetchPosts(username, before = 0n) {
    return doGet(`/api/users/${username}/posts?before=${before}&last=${PAGE_SIZE}`)
}

/**
 * @param {string} username
 * @returns {Promise<import("../types.js").ToggleFollowOutput>}
 */
function toggleFollow(username) {
    return doPost(`/api/users/${username}/toggle_follow`)
}

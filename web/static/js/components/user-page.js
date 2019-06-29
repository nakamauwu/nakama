import { isAuthenticated } from "../auth.js";
import { doGet, doPost } from "../http.js";
import renderAvatarHTML from "./avatar.js";
import renderList from "./list.js";
import renderPost from "./post.js";

const PAGE_SIZE = 10

const template = document.createElement("template")
template.innerHTML = `
    <div class="user-wrapper">
        <div class="container wide">
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
        <div class="center-vertically">
            <div>
                <h1>${user.username}</h1>
                ${user.followeed ? `
                    <span class="badge">Follows you</span>
                ` : ""}
            </div>
            <div class="user-stats">
                <a href="/users/${user.username}/followers">
                    <span class="followers-count-span">${user.followersCount}</span>
                    followers
                </a>
                <a href="/users/${user.username}/followees">${user.followeesCount} followees</a>
            </div>
        </div>
        ${authenticated && !user.me ? `
            <button class="follow-button" aria-pressed="${user.following}">
                ${user.following ? "Following" : "Follow"}
            </button>
        ` : ""}
    `

    const followersCountSpan = /** @type {HTMLSpanElement} */ (div.querySelector(".followers-count-span"))
    const followButton = /** @type {HTMLButtonElement=} */ (div.querySelector(".follow-button"))

    if (followButton !== null) {
        const onFollowButtonClick = async () => {
            followButton.disabled = true

            try {
                const out = await toggleFollow(user.username)
                followersCountSpan.textContent = String(out.followersCount)
                followButton.setAttribute("aria-pressed", String(out.following))
                followButton.textContent = out.following ? "Following" : "Follow"
            } catch (err) {
                console.log(err)
                alert(err.message)
            } finally {
                followButton.disabled = false
            }
        }

        followButton.addEventListener("click", onFollowButtonClick)
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

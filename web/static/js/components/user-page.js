import { doGet } from "../http.js"
import renderList from "./list.js"
import renderPost from "./post.js"
import renderUserProfile from "./user-profile.js"

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

    userOutlet.appendChild(renderUserProfile(user, true))
    postsOutlet.appendChild(list.el)

    page.addEventListener("disconnect", list.teardown)

    return page
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

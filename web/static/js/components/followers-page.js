import { doGet } from "../http.js"
import renderList from "./list.js"
import renderUserProfile from "./user-profile.js"

const PAGE_SIZE = 10
const template = document.createElement("template")
template.innerHTML = /*html*/`
    <div class="container">
        <h1><span id="username-outlet"></span>'s followers</h1>
        <div id="followers-outlet" class="followers-wrapper users-wrapper"></div>
    </div>
`

/**
 * @param {object} params
 * @param {string} params.username
 */
export default async function renderFollowersPage(params) {
    const paginatedUsers = await fetchFollowers(params.username)
    const list = renderList({
        page: paginatedUsers,
        loadMoreFunc: after => fetchFollowers(params.username, after),
        pageSize: PAGE_SIZE,
        renderItem: renderUserProfile,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const usernameOutlet = page.getElementById("username-outlet")
    const followersOutlet = page.getElementById("followers-outlet")

    usernameOutlet.textContent = params.username
    followersOutlet.appendChild(list.el)

    return page
}

/**
 * @param {string} username
 * @param {string|null} after
 * @returns {Promise<import("../types.js").Page<import("../types.js").UserProfile>>}
 */
function fetchFollowers(username, after = null) {
    return doGet(`/api/users/${encodeURIComponent(username)}/followers?first=${encodeURIComponent(PAGE_SIZE)}` + (after !== null ? "&after=" + encodeURIComponent(after) : ""))
}

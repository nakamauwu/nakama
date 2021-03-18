import { doGet } from "../http.js"
import renderList from "./list.js"
import renderUserProfile from "./user-profile.js"

const PAGE_SIZE = 10
const template = document.createElement("template")
template.innerHTML = /*html*/`
    <div class="container">
        <h1><span id="username-outlet"></span>'s followees</h1>
        <div id="followees-outlet" class="followees-wrapper users-wrapper"></div>
    </div>
`

/**
 * @param {object} params
 * @param {string} params.username
 */
export default async function renderFolloweesPage(params) {
    const users = await fetchFollowees(params.username)
    const list = renderList({
        getID: u => u.username,
        items: users,
        loadMoreFunc: after => fetchFollowees(params.username, after),
        pageSize: PAGE_SIZE,
        renderItem: renderUserProfile,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const usernameOutlet = page.getElementById("username-outlet")
    const followeesOutlet = page.getElementById("followees-outlet")

    usernameOutlet.textContent = params.username
    followeesOutlet.appendChild(list.el)

    return page
}

/**
 * @param {string} username
 * @param {string=} after
 * @returns {Promise<import("../types.js").UserProfile[]>}
 */
function fetchFollowees(username, after = "") {
    return doGet(`/api/users/${username}/followees?after=${after}&first=${PAGE_SIZE}`)
}

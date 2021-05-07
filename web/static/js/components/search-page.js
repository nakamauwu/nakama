import { doGet } from "../http.js"
import { navigate } from "../lib/router.js"
import renderList from "./list.js"
import renderUserProfile from "./user-profile.js"

const PAGE_SIZE = 10
const template = document.createElement("template")
template.innerHTML = /*html*/`
    <div class="container">
        <h1>Search</h1>
        <form id="search-form" class="search-form">
            <input type="search" name="q" placeholder="Search..." autocomplete="off">
        </form>
        <div id="search-results-outlet" class="search-results-wrapper users-wrapper"></div>
    </div>
`

export default async function renderSearchPage() {
    const url = new URL(location.toString())
    const searchQuery = url.searchParams.has("q") ? decodeURIComponent(url.searchParams.get("q")).trim() : ""

    const users = await fetchUsers(searchQuery)
    const list = renderList({
        getID: u => u.username,
        items: users,
        loadMoreFunc: after => fetchUsers(searchQuery, after),
        pageSize: PAGE_SIZE,
        renderItem: renderUserProfile,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const searchForm = /** @type {HTMLFormElement} */ (page.getElementById("search-form"))
    const searchInput = searchForm.querySelector("input")
    const searchResultsOutlet = page.getElementById("search-results-outlet")

    /**
     * @param {Event} ev
     */
    const onSearchFormSubmit = ev => {
        ev.preventDefault()
        const searchQuery = searchInput.value.trim()
        navigate("/search?q=" + encodeURIComponent(searchQuery))
    }

    searchForm.addEventListener("submit", onSearchFormSubmit)
    searchInput.value = searchQuery
    searchResultsOutlet.appendChild(list.el)

    return page
}

/**
 * @param {string} search
 * @param {string=} after
 * @returns {Promise<import("../types.js").UserProfile[]>}
 */
function fetchUsers(search, after = "") {
    return doGet(`/api/users?search=${search}&after=${after}&first=${PAGE_SIZE}`)
}

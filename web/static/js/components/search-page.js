import { doGet } from "../http.js"
import { translate } from "../i18n/i18n.js"
import { navigate } from "../lib/router.js"
import renderList from "./list.js"
import renderUserProfile from "./user-profile.js"

const PAGE_SIZE = 10
const template = document.createElement("template")
template.innerHTML = /*html*/`
    <div class="container">
        <h1>${translate("searchPage.heading")}</h1>
        <form id="search-form" class="search-form">
            <input type="search" name="q" placeholder="${translate("searchPage.searchBoxPlaceholder")}" autocomplete="off">
        </form>
        <div id="search-results-outlet" class="search-results-wrapper users-wrapper"></div>
    </div>
`

export default async function renderSearchPage() {
    const url = new URL(location.toString())
    let searchQuery = /** @type {string|null} */ (null)
    if (url.searchParams.has("q")) {
        const s = decodeURIComponent(url.searchParams.get("q")).trim()
        if (s !== "") {
            searchQuery = s
        }
    }

    const paginatedUsers = await fetchUsers(searchQuery)
    const list = renderList({
        page: paginatedUsers,
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
 * @param {string|null} search
 * @param {string|null} after
 * @returns {Promise<import("../types.js").Page<import("../types.js").UserProfile>>}
 */
function fetchUsers(search = null, after = null) {
    return doGet(`/api/users?first=${encodeURIComponent(PAGE_SIZE)}` + (search !== null ? "&search=" + encodeURIComponent(search) : "") + (after !== null ? "&after=" + encodeURIComponent(after) : ""))
}

import { isAuthenticated } from '../auth.js';
import { doGet, doPost } from '../http.js';
import renderAvatarHTML from './avatar.js';
import renderList from './list.js';
import renderPost from './post.js';

const PAGE_SIZE = 10

const template = document.createElement('template')
template.innerHTML = `
    <div class="user-wrapper">
        <div class="container wide">
            <div id="user-div"></div>
        </div>
    </div>
    <div class="container">
        <h2>Posts</h2>
        <div id="posts-div" class="posts-wrapper"></div>
    </div>
`

export default async function renderUserPage(params) {
    const [user, posts] = await Promise.all([
        http.fetchUser(params.username),
        http.fetchPosts(params.username),
    ])
    for (const post of posts) {
        post.user = user
    }

    const loadMore = async before => {
        const posts = await http.fetchPosts(user.username, before)
        for (const post of posts) {
            post.user = user
        }
        return posts
    }

    const list = renderList({
        items: posts,
        fetchMoreItems: loadMore,
        pageSize: PAGE_SIZE,
        renderItem: renderPost,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const userDiv = page.getElementById('user-div')
    const postsDiv = page.getElementById('posts-div')

    userDiv.appendChild(renderUserProfile(user))
    postsDiv.appendChild(list.el)

    page.addEventListener('disconnect', list.teardown)

    return page
}

/**
 * @param {import('../types.js').UserProfile} user
 */
function renderUserProfile(user) {
    const authenticated = isAuthenticated()
    const div = document.createElement('div')
    div.className = 'user-profile'
    div.innerHTML = `
        ${renderAvatarHTML(user)}
        <div class="center-vertically">
            <div>
                <h1>${user.username}</h1>
                ${user.followeed ? `
                    <span class="badge">Follows you</span>
                ` : ''}
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
                ${user.following ? 'Following' : 'Follow'}
            </button>
        ` : ''}
    `

    const followersCountSpan = /** @type {HTMLSpanElement} */ (div.querySelector('.followers-count-span'))
    const followButton = /** @type {HTMLButtonElement=} */ (div.querySelector('.follow-button'))

    if (followButton !== null) {
        const onFollowButtonClick = async () => {
            followButton.disabled = true

            try {
                const out = await http.toggleFollow(user.username)
                followersCountSpan.textContent = String(out.followersCount)
                followButton.setAttribute('aria-pressed', String(out.following))
                followButton.textContent = out.following ? 'Following' : 'Follow'
            } catch (err) {
                console.log(err)
                alert(err.message)
            } finally {
                followButton.disabled = false
            }
        }

        followButton.addEventListener('click', onFollowButtonClick)
    }

    return div
}

const http = {
    /**
     * @param {string} username
     * @returns {Promise<import('../types.js').UserProfile>}
     */
    fetchUser: username => doGet('/api/users/' + username),

    /**
     * @param {string} username
     * @param {bigint=} before
     * @returns {Promise<import('../types.js').Post[]>}
     */
    fetchPosts: (username, before = 0n) =>
        doGet(`/api/users/${username}/posts?before=${before}&last=${PAGE_SIZE}`),

    /**
     * @param {string} username
     * @returns {Promise<import('../types.js').ToggleFollowOutput>}
     */
    toggleFollow: username =>
        doPost(`/api/users/${username}/toggle_follow`),
}

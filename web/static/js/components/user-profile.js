import { isAuthenticated } from "../auth.js"
import { doPost } from "../http.js"
import { el, replaceNode } from "../utils.js"
import renderAvatarHTML from "./avatar.js"
import { personAddIconSVG, personDoneIconSVG } from "./icons.js"

/**
 * @param {import("../types.js").UserProfile} user
 */
export default function renderUserProfile(user, full = false) {
    const authenticated = isAuthenticated()
    const article = document.createElement("article")
    article.className = "user-profile"
    article.innerHTML = `
        ${full ? `
            ${renderAvatarHTML(user, "Double click to update avatar")}
            <h1 class="user-username" title="Double click to update username">${user.username}</h1>
        ` : `
            <a href="/users/${user.username}">${renderAvatarHTML(user)}</a>
            <a href="/users/${user.username}" class="user-username">${user.username}</a>
        `}
        ${user.followeed ? `<span class="badge">Follows you</span>` : ""}
        ${authenticated && !user.me ? `
            <div class="user-controls">
                <button class="follow-button" aria-pressed="${user.following}">
                    ${user.following ? personDoneIconSVG : personAddIconSVG}
                    <span>${user.following ? "Following" : "Follow"}</span>
                </button>
            </div>
        ` : full && user.me ? `
            <div class="user-controls">
                <button class="logout-button">
                    <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="log-out"><rect width="24" height="24" transform="rotate(90 12 12)" opacity="0"/><path d="M7 6a1 1 0 0 0 0-2H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h2a1 1 0 0 0 0-2H6V6z"/><path d="M20.82 11.42l-2.82-4a1 1 0 0 0-1.39-.24 1 1 0 0 0-.24 1.4L18.09 11H10a1 1 0 0 0 0 2h8l-1.8 2.4a1 1 0 0 0 .2 1.4 1 1 0 0 0 .6.2 1 1 0 0 0 .8-.4l3-4a1 1 0 0 0 .02-1.18z"/></g></g></svg>
                    <span>Logout</span>
                </button>
                <input class="js-avatar-input" type="file" name="avatar" accept="image/png,image/jpeg" required hidden>
            </div>
        ` : ""}
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
    `

    const usernameText = article.querySelector(".user-username")
    const avatarPic = article.querySelector(".avatar")
    const followersCountSpan = /** @type {HTMLSpanElement} */ (article.querySelector(".followers-count"))
    const avatarInput = /** @type {HTMLInputElement=} */ (article.querySelector(".js-avatar-input"))
    const logoutButton = /** @type {HTMLButtonElement=} */ (article.querySelector(".logout-button"))
    const followButton = /** @type {HTMLButtonElement=} */ (article.querySelector(".follow-button"))

    if (full && user.me) {
        const onUsernameDoubleClick = () => {
            prompt("New username:", user.username)
        }

        const onAvatarDoubleClick = () => {
            avatarInput.click()
        }

        const onLogoutButtonClick = () => {
            logoutButton.disabled = true
            localStorage.clear()
            location.assign("/")
        }

        usernameText.addEventListener("dblclick", onUsernameDoubleClick)
        avatarPic.addEventListener("dblclick", onAvatarDoubleClick)
        logoutButton.addEventListener("click", onLogoutButtonClick)
    }

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

    return article
}

/**
 * @param {string} username
 * @returns {Promise<import("../types.js").ToggleFollowOutput>}
 */
function toggleFollow(username) {
    return doPost(`/api/users/${username}/toggle_follow`)
}

import { getAuthUser, isAuthenticated } from "../auth.js"
import { doGet, doPatch, doPost, doPut } from "../http.js"
import { arrayBufferToBase64, base64ToArrayBuffer, el, replaceNode, reUsername } from "../utils.js"
import renderAvatarHTML from "./avatar.js"
import { personAddIconSVG, personDoneIconSVG } from "./icons.js"

/**
 * @param {import("../types.js").UserProfile} user
 */
export default function renderUserProfile(user, full = false) {
    const authenticated = isAuthenticated()
    const article = document.createElement("article")
    article.className = "user-profile"
    article.innerHTML = /*html*/`
        ${full ? /*html*/`
            ${renderAvatarHTML(user, "Double click to update avatar")}
            <h1 class="user-username" title="Double click to update username">${user.username}</h1>
        ` : /*html*/`
            <a href="/users/${user.username}">${renderAvatarHTML(user)}</a>
            <a href="/users/${user.username}" class="user-username">${user.username}</a>
        `}
        ${user.followeed ? `<span class="badge">Follows you</span>` : ""}
        ${authenticated && !user.me ? /*html*/`
            <div class="user-controls">
                <button class="follow-button" aria-pressed="${user.following}">
                    ${user.following ? personDoneIconSVG : personAddIconSVG}
                    <span>${user.following ? "Following" : "Follow"}</span>
                </button>
            </div>
        ` : full && user.me ? /*html*/`
            <div class="user-controls">
                <button class="webauthn-btn" hidden>Register device credentials</button>
                <button class="logout-button">
                    <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="log-out"><rect width="24" height="24" transform="rotate(90 12 12)" opacity="0"/><path d="M7 6a1 1 0 0 0 0-2H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h2a1 1 0 0 0 0-2H6V6z"/><path d="M20.82 11.42l-2.82-4a1 1 0 0 0-1.39-.24 1 1 0 0 0-.24 1.4L18.09 11H10a1 1 0 0 0 0 2h8l-1.8 2.4a1 1 0 0 0 .2 1.4 1 1 0 0 0 .6.2 1 1 0 0 0 .8-.4l3-4a1 1 0 0 0 .02-1.18z"/></g></g></svg>
                    <span>Logout</span>
                </button>
                <form hidden>
                    <input class="js-avatar-input" type="file" name="avatar" accept="image/png,image/jpeg" required hidden>
                </form>
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
    const avatarPic = /** @type {HTMLImageElement|HTMLSpanElement} */ (article.querySelector(".avatar"))
    const followersCountSpan = /** @type {HTMLSpanElement} */ (article.querySelector(".followers-count"))
    const avatarInput = /** @type {HTMLInputElement=} */ (article.querySelector(".js-avatar-input"))
    const logoutButton = /** @type {HTMLButtonElement=} */ (article.querySelector(".logout-button"))
    const followButton = /** @type {HTMLButtonElement=} */ (article.querySelector(".follow-button"))
    const webAuthnBtn = /** @type {HTMLButtonElement} */ (article.querySelector(".webauthn-btn"))

    if (full && user.me) {
        const onUsernameDoubleClick = () => {
            const username = prompUsername(user.username)
            if (username === null) {
                return null
            }

            updateUser({ username }).then(updated => {
                user.username = updated.username
                const authUser = getAuthUser()
                authUser.username = updated.username
                localStorage.setItem("auth_user", JSON.stringify(authUser))
                usernameText.textContent = updated.username

                const ok = confirm("Username changed, refresh the page to see the changes")
                if (ok) {
                    window.location.replace("/users/" + encodeURIComponent(updated.username))
                }
            }).catch(err => {
                console.error(err)
                alert("Something went wrong while updating your username: " + err.message)
            })
        }

        const onAvatarDoubleClick = () => {
            avatarInput.click()
        }

        const onAvatarInputChange = () => {
            const files = avatarInput.files
            if (!(files instanceof window.FileList) || files.length !== 1) {
                return
            }

            const file = files.item(0)
            updateAvatar(file).then(avatarURL => {
                const authUser = getAuthUser()
                authUser.avatarURL = avatarURL
                localStorage.setItem("auth_user", JSON.stringify(authUser))
                avatarPic["src"] = avatarURL

                const ok = confirm("Avatar updated, refresh the page to see the changes")
                if (ok) {
                    window.location.reload()
                }
            }).catch(err => {
                console.error(err)
                alert("Something went wrong while updating your avatar: " + err.message)
            })
        }

        const onLogoutButtonClick = () => {
            if (typeof navigator.vibrate === "function") {
                navigator.vibrate([50])
            }

            logoutButton.disabled = true
            localStorage.removeItem("auth_user")
            localStorage.removeItem("auth_token")
            localStorage.removeItem("auth_expires_at")
            location.assign("/")
        }

        const onWebAuthnClick = async () => {
            if (typeof navigator.vibrate === "function") {
                navigator.vibrate([50])
            }

            webAuthnBtn.disabled = true
            try {
                const opts = await createCredentialCreationOptions()
                const cred = await navigator.credentials.create(opts)

                await createCredential(cred)

                localStorage.setItem("webauthn_credential_id", cred.id)
                alert("Device registered successfully. Now you can login with device credentials")
            } catch (err) {
                if (err instanceof Error && err.name === "InvalidStateError") {
                    alert("Device already registered")
                    return
                }
                console.error(err)
                alert(err.message)
            } finally {
                webAuthnBtn.disabled = false
            }
        }

        usernameText.style.userSelect = "none"
        usernameText.style.touchAction = "manipulation"

        avatarPic.style.userSelect = "none"
        avatarPic.style.touchAction = "manipulation"

        usernameText.addEventListener("dblclick", onUsernameDoubleClick)
        avatarPic.addEventListener("dblclick", onAvatarDoubleClick)
        avatarPic.addEventListener("webkitmouseforcewillbegin", ev => { ev.preventDefault() })
        avatarPic.addEventListener("webkitmouseforcedown", onAvatarDoubleClick)
        avatarInput.addEventListener("change", onAvatarInputChange)
        logoutButton.addEventListener("click", onLogoutButtonClick)

        if ("PublicKeyCredential" in window) {
            PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable().then(ok => {
                if (!ok) {
                    return
                }

                webAuthnBtn.addEventListener("click", onWebAuthnClick)
                webAuthnBtn.hidden = false
            })
        }
    }

    if (followButton !== null) {
        const followText = followButton.querySelector("span")
        const onFollowButtonClick = async () => {
            if (typeof navigator.vibrate === "function") {
                navigator.vibrate([50])
            }

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
                console.error(err)
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

/**
 * @param {File} avatar
 * @returns {Promise<string>}
 */
function updateAvatar(avatar) {
    return doPut("/api/auth_user/avatar", avatar)
}

/**
 * @returns {Promise<CredentialCreationOptions>}
 */
async function createCredentialCreationOptions() {
    const opts = await doGet("/api/credential_creation_options")

    opts.publicKey.user.id = base64ToArrayBuffer(opts.publicKey.user.id)
    opts.publicKey.challenge = base64ToArrayBuffer(opts.publicKey.challenge)

    if (Array.isArray(opts.publicKey.excludeCredentials)) {
        opts.publicKey.excludeCredentials.forEach((cred, i) => {
            opts.publicKey.excludeCredentials[i].id = base64ToArrayBuffer(cred.id)
        })
    }

    return opts
}

/**
 * @param {Credential} cred
 * @returns {Promise<any>}
 */
async function createCredential(cred) {
    const b = {
        id: cred.id,
        type: cred.type,
    }
    if ("rawId" in cred) {
        b["rawId"] = arrayBufferToBase64(cred["rawId"])
    }

    if ("response" in cred) {
        const resp = /** @type {AuthenticatorAttestationResponse} */ (cred["response"])
        b["response"] = {}
        if ("attestationObject" in resp) {
            b["response"]["attestationObject"] = arrayBufferToBase64(resp.attestationObject)
        }
        if ("clientDataJSON" in resp) {
            b["response"]["clientDataJSON"] = arrayBufferToBase64(resp.clientDataJSON)
        }
    }

    await doPost("/api/credentials", b)
}

function prompUsername(initialUsername) {
    let username = prompt("New username:", initialUsername)
    if (username === null) {
        return null
    }

    username = username.trim()

    if (!reUsername.test(username)) {
        alert("invalid username")
        return prompUsername(initialUsername)
    }

    return username
}

/**
 * @param {object} params
 * @param {string|null} params.username
 * @returns {Promise<{username:string}>}
 */
function updateUser(params) {
    return doPatch("/api/auth_user", params)
}

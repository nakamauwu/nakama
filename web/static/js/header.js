import { getAuthUser } from './auth.js';

const authUser = getAuthUser()
const authenticated = authUser !== null
const header = document.querySelector('header')
header.innerHTML = `
    <div class="container wide">
        <nav>
            <a href="/">Home</a>
            ${authenticated ? `
                <a href="/users/${authUser.username}">Profile</a>
                <button id="logout-button">Logout</button>
            ` : ''}
        </nav>
    </div>
`

if (authenticated) {
    const logoutButton = /** @type {HTMLButtonElement} */ (header.querySelector('#logout-button'))
    logoutButton.addEventListener('click', onLogoutButtonClick)
}

/**
 * @param {MouseEvent} ev
 */
function onLogoutButtonClick(ev) {
    const button = /** @type {HTMLButtonElement} */ (ev.currentTarget)
    button.disabled = true
    localStorage.clear()
    location.reload()
}

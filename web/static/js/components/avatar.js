/**
 * @param {import("../types.js").User} user
 */
export default function renderAvatarHTML(user, title = "") {
    return user.avatarURL !== null
        ? `<img class="avatar" src="${user.avatarURL}" alt="${user.username}'s avatar" title="${title}">`
        : `<span class="avatar" data-initial="${user.username[0]}" title="${title}"></span>`
}

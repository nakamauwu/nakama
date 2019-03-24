/**
 * @param {import('../types.js').User} user
 */
export default function renderAvatarHTML(user) {
    return user.avatarURL !== null
        ? `<img class="avatar" src="${user.avatarURL}" alt="${user.username}'s avatar">`
        : `<span class="avatar" data-initial="${user.username[0]}"></span>`
}

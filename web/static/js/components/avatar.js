
/**
 * @param {import('../types.js').User} user
 * @param {'small'|'big'=} type
 */
export default function renderAvatarHTML(user, type) {
    let className = 'avatar'
    if (typeof type === 'string') {
        className += ' ' + type
    }
    return user.avatarURL !== null
        ? `<img class="${className}" src="${user.avatarURL}" alt="${user.username}'s avatar">`
        : `<span class="${className}" data-initial="${user.username[0]}"></span>`
}

import { html } from "lit-html"

export function Avatar(user) {
    return user.avatarURL !== null ? html`
        <img class="avatar" src="${user.avatarURL}" alt="">
    ` : html`
        <span class="avatar" data-initial="${user.username[0]}"></span>
    `
}

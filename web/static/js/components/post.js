import { isAuthenticated } from '../auth.js';
import { escapeHTML, linkify } from '../utils.js';
import renderAvatarHTML from './avatar.js';
import { heartFilledSVG, heartOutlinedSVG } from './heart-icons.js';

// #region icons
const squareMessageSVG = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="message-square"><rect width="24" height="24" opacity="0"/><circle cx="12" cy="11" r="1"/><circle cx="16" cy="11" r="1"/><circle cx="8" cy="11" r="1"/><path d="M19 3H5a3 3 0 0 0-3 3v15a1 1 0 0 0 .51.87A1 1 0 0 0 3 22a1 1 0 0 0 .51-.14L8 19.14a1 1 0 0 1 .55-.14H19a3 3 0 0 0 3-3V6a3 3 0 0 0-3-3zm1 13a1 1 0 0 1-1 1H8.55a3 3 0 0 0-1.55.43l-3 1.8V6a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1z"/></g></g></svg>`
// #endregion

/**
 * @param {import('../types.js').Post} post
 * @param {bigint=} timelineItemID
 */
export default function renderPost(post, timelineItemID) {
    const authenticated = isAuthenticated()
    const { user } = post
    const timestamp = new Date(post.createdAt).toLocaleString()
    const content = linkify(escapeHTML(post.content))

    const article = document.createElement('article')
    article.className = 'micro-post'
    article.setAttribute('aria-label', `${user.username}'s post`)
    article.innerHTML = `
        <div class="micro-post-header">
            <a class="micro-post-user" href="/users/${user.username}">
                ${renderAvatarHTML(user)}
                <span>${user.username}</span>
            </a>
            <a href="/posts/${post.id}">
                <time datetime="${post.createdAt}">${timestamp}</time>
            </a>
        </div>
        <div class="micro-post-content">${content}</div>
        <div class="micro-post-controls">
            ${authenticated ? `
                <button class="like-button"
                    title="${post.liked ? 'Unlike' : 'Like'}"
                    aria-pressed="${post.liked}"
                    aria-label="${post.likesCount} likes">
                    <span class="likes-count">${post.likesCount}</span>
                    ${post.liked ? heartFilledSVG : heartOutlinedSVG}
                </button>
            ` : `
                <span class="brick" aria-label="${post.likesCount} likes">
                    <span>${post.likesCount}</span>
                    ${post.liked ? heartFilledSVG : heartOutlinedSVG}
                </span>
            `}
            <a class="brick comments-link"
                href="/posts/${post.id}"
                title="Comments"
                aria-label="${post.commentsCount} comments">
                <span class="comments-count">${post.commentsCount}</span>
                ${squareMessageSVG}
            </a>
        </div>
    `

    return article
}

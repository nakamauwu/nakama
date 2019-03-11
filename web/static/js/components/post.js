import { isAuthenticated } from '../auth.js';
import { escapeHTML } from '../utils.js';
import renderAvatarHTML from './avatar.js';

const heartOutlined = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="heart"><rect width="24" height="24" opacity="0"/><path d="M12 21a1 1 0 0 1-.71-.29l-7.77-7.78a5.26 5.26 0 0 1 0-7.4 5.24 5.24 0 0 1 7.4 0L12 6.61l1.08-1.08a5.24 5.24 0 0 1 7.4 0 5.26 5.26 0 0 1 0 7.4l-7.77 7.78A1 1 0 0 1 12 21zM7.22 6a3.2 3.2 0 0 0-2.28.94 3.24 3.24 0 0 0 0 4.57L12 18.58l7.06-7.07a3.24 3.24 0 0 0 0-4.57 3.32 3.32 0 0 0-4.56 0l-1.79 1.8a1 1 0 0 1-1.42 0L9.5 6.94A3.2 3.2 0 0 0 7.22 6z"/></g></g></svg>`
const heartFilled = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="heart"><rect width="24" height="24" opacity="0"/><path d="M12 21a1 1 0 0 1-.71-.29l-7.77-7.78a5.26 5.26 0 0 1 0-7.4 5.24 5.24 0 0 1 7.4 0L12 6.61l1.08-1.08a5.24 5.24 0 0 1 7.4 0 5.26 5.26 0 0 1 0 7.4l-7.77 7.78A1 1 0 0 1 12 21z"/></g></g></svg>`

/**
 * @param {import('../types.js').Post} post
 */
export default function renderPost(post) {
    const authenticated = isAuthenticated()
    const { user } = post
    const timestamp = new Date(post.createdAt).toLocaleString()
    const li = document.createElement('li')
    li.className = 'post-item'
    li.innerHTML = `
        <article class="post">
            <div class="post-header">
                <a class="post-user" href="/users/${user.username}">
                    ${renderAvatarHTML(user)}
                    <span>${user.username}</span>
                </a>
                <a href="/posts/${post.id}">
                    <time datetime="${post.createdAt}">${timestamp}</time>
                </a>
            </div>
            <div class="post-content">${escapeHTML(post.content)}</div>
            <div class="post-controls">
                ${authenticated ? `
                    <button class="like-button${post.liked ? ' liked' : ''}">
                        <span class="likes-count">${post.likesCount}</span>
                        ${post.liked ? heartFilled : heartOutlined}
                    </button>
                ` : `
                    <span class="brick">
                        <span>${post.likesCount}</span>
                        ${post.liked ? heartFilled : heartOutlined}
                    </span>
                `}
                <a class="brick comments-link" href="/posts/${post.id}">
                    <span class="comments-count">${post.commentsCount}</span>
                    <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="message-square"><rect width="24" height="24" opacity="0"/><circle cx="12" cy="11" r="1"/><circle cx="16" cy="11" r="1"/><circle cx="8" cy="11" r="1"/><path d="M19 3H5a3 3 0 0 0-3 3v15a1 1 0 0 0 .51.87A1 1 0 0 0 3 22a1 1 0 0 0 .51-.14L8 19.14a1 1 0 0 1 .55-.14H19a3 3 0 0 0 3-3V6a3 3 0 0 0-3-3zm1 13a1 1 0 0 1-1 1H8.55a3 3 0 0 0-1.55.43l-3 1.8V6a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1z"/></g></g></svg>
                </a>
                ${authenticated ? `
                    <button title="More">
                        <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="more-horizotnal"><rect width="24" height="24" opacity="0"/><circle cx="12" cy="12" r="2"/><circle cx="19" cy="12" r="2"/><circle cx="5" cy="12" r="2"/></g></g></svg>
                    </button>
                ` : ''}
            </div>
        </article>
    `
    return li
}

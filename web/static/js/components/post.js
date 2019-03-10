import { escapeHTML } from '../utils.js';
import renderAvatarHTML from './avatar.js';

/**
 * @param {import('../types.js').Post} post
 */
export default function renderPost(post) {
    const { user } = post
    const timestamp = new Date(post.createdAt).toLocaleString()
    const li = document.createElement('li')
    li.className = 'post-item'
    li.innerHTML = `
        <article class="post">
            <div class="post-header">
                <a href="/users/${user.username}">
                    ${renderAvatarHTML(user)}
                    <span>${user.username}</span>
                </a>
                <a href="/posts/${post.id}">
                    <time datetime="${post.createdAt}">${timestamp}</time>
                </a>
            </div>
            <div class="post-content">${escapeHTML(post.content)}</div>
            <div class="post-controls">
                <button class="like-button">${post.likesCount}</button>
                <a class="comments-link" href="/posts/${post.id}">${post.commentsCount}</a>
            </div>
        </article>
    `
    return li
}

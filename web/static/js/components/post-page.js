import { getAuthUser, isAuthenticated } from '../auth.js';
import { doGet, doPost, subscribe } from '../http.js';
import { ago, escapeHTML, linkify } from '../utils.js';
import renderAvatarHTML from './avatar.js';
import heartIconSVG from './heart-icon.js';
import renderList from './list.js';
import renderPost from './post.js';

const PAGE_SIZE = 3

const template = document.createElement('template')
template.innerHTML = `
    <div class="post-wrapper">
        <div class="container">
            <div id="post-outlet"></div>
        </div>
    </div>
    <div class="container">
        <div id="comments-outlet" class="comments-wrapper"></div>
        <form id="comment-form" class="comment-form" hidden>
            <textarea placeholder="Say something..." maxlength="480" required></textarea>
            <button>Comment</button>
        </form>
    </div>
`

export default async function renderPostPage(params) {
    const postID = BigInt(params.postID)
    const [post, comments] = await Promise.all([
        fetchPost(postID),
        fetchComments(postID),
    ])

    const list = renderList({
        items: comments,
        renderItem: renderComment,
        loadMoreFunc: before => fetchComments(post.id, before),
        pageSize: PAGE_SIZE,
        reverse: true,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const postOutlet = page.getElementById('post-outlet')
    let commentsLink = /** @type {HTMLAnchorElement} */ (null)
    let commentsCountEl = /** @type {HTMLElement=} */ (null)
    const commentsOutlet = page.getElementById('comments-outlet')
    const commentForm = /** @type {HTMLFormElement} */ (page.getElementById('comment-form'))
    const commentFormTextArea = commentForm.querySelector('textarea')
    const commentFormButton = commentForm.querySelector('button')

    const incrementCommentsCount = () => {
        if (commentsLink === null) {
            commentsLink = postOutlet.querySelector('.comments-link')
        }
        if (commentsCountEl === null) {
            commentsCountEl = postOutlet.querySelector('.comments-count')
        }
        post.commentsCount++
        commentsLink.setAttribute('aria-title', post.commentsCount + ' comments')
        commentsCountEl.textContent = String(post.commentsCount)
    }

    /**
     * @param {Event} ev
     */
    const onCommentFormSubmit = async ev => {
        ev.preventDefault()
        const content = commentFormTextArea.value
        commentFormTextArea.disabled = true
        commentFormButton.disabled = true
        try {
            const comment = await createComment(post.id, content)

            list.enqueue(comment)
            list.flush()
            incrementCommentsCount()

            commentForm.reset()
        } catch (err) {
            console.error(err)
            alert(err.message)
            setTimeout(() => {
                commentFormTextArea.focus()
            })
        } finally {
            commentFormTextArea.disabled = false
            commentFormButton.disabled = false
        }
    }

    /**
     * @param {import('../types.js').Comment} comment
     */
    const onCommentArrive = comment => {
        list.enqueue(comment)
        incrementCommentsCount()
    }

    const unsubscribeFromComments = subscribeToComments(post.id, onCommentArrive)

    const onPageDisconnect = () => {
        unsubscribeFromComments()
        list.teardown()
    }

    postOutlet.appendChild(renderPost(post))
    commentsOutlet.appendChild(list.el)
    if (isAuthenticated()) {
        commentForm.hidden = false
        commentForm.addEventListener('submit', onCommentFormSubmit)
    } else {
        commentForm.remove()
    }
    page.addEventListener('disconnect', onPageDisconnect)

    return page
}

/**
 * @param {import('../types.js').Comment} comment
 */
function renderComment(comment) {
    const authenticated = isAuthenticated()
    const { user } = comment
    const content = linkify(escapeHTML(comment.content))

    const article = document.createElement('article')
    article.className = 'micro-post'
    article.setAttribute('aria-label', `${user.username}'s comment`)
    article.innerHTML = `
        <div class="micro-post-header">
            <a class="micro-post-user" href="/users/${user.username}">
                ${renderAvatarHTML(user)}
                <span>${user.username}</span>
            </a>
            <time datetime="${comment.createdAt}">${ago(comment.createdAt)}</time>
        </div>
        <div class="micro-post-content">${content}</div>
        <div class="micro-post-controls">
            ${authenticated ? `
                <button class="like-button"
                    title="${comment.liked ? 'Unlike' : 'Like'}"
                    aria-pressed="${comment.liked}"
                    aria-label="${comment.likesCount} likes">
                    <span class="likes-count">${comment.likesCount}</span>
                    ${heartIconSVG}
                </button>
            ` : `
                <span class="brick" aria-label="${comment.likesCount} likes">
                    <span>${comment.likesCount}</span>
                    ${heartIconSVG}
                </span>
            `}
        </div>
    `

    const likeButton = /** @type {HTMLButtonElement=} */ (article.querySelector('.like-button'))
    if (likeButton !== null) {
        const likesCountEl = likeButton.querySelector('.likes-count')

        const onLikeButtonClick = async () => {
            likeButton.disabled = true
            try {
                const out = await toggleCommentLike(comment.id)

                comment.likesCount = out.likesCount
                comment.liked = out.liked

                likeButton.title = out.liked ? 'Unlike' : 'Like'
                likeButton.setAttribute('aria-pressed', String(out.liked))
                likeButton.setAttribute('aria-label', out.likesCount + ' likes')
                likesCountEl.textContent = String(out.likesCount)
            } catch (err) {
                console.error(err)
                alert(err.message)
            } finally {
                likeButton.disabled = false
            }
        }

        likeButton.addEventListener('click', onLikeButtonClick)
    }

    return article
}

/**
 * @param {bigint} postID
 * @returns {Promise<import('../types.js').Post>}
 */
function fetchPost(postID) {
    return doGet('/api/posts/' + postID)
}

/**
 * @param {bigint} postID
 * @param {bigint=} before
 * @returns {Promise<import('../types.js').Comment[]>}
 */
function fetchComments(postID, before = 0n) {
    return doGet(`/api/posts/${postID}/comments?before=${before}&last=${PAGE_SIZE}`)
}

/**
 * @param {bigint} postID
 * @param {string} content
 * @returns {Promise<import('../types.js').Comment>}
 */
async function createComment(postID, content) {
    const comment = await doPost(`/api/posts/${postID}/comments`, { content })
    comment.user = getAuthUser()
    return comment
}

/**
 *
 * @param {bigint} postID
 * @param {function(import('../types.js').Comment):any} cb
 */
function subscribeToComments(postID, cb) {
    return subscribe(`/api/posts/${postID}/comments`, cb)
}

/**
 * @param {bigint} commentID
 * @returns {Promise<import('../types.js').ToggleLikeOutput>}
 */
function toggleCommentLike(commentID) {
    return doPost(`/api/comments/${commentID}/toggle_like`)
}

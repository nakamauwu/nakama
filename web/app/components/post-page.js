import { Textcomplete } from "@textcomplete/core"
import { TextareaEditor } from "@textcomplete/textarea"
import { component, useEffect, useState } from "haunted"
import { html } from "lit"
import { createRef, ref } from "lit/directives/ref.js"
import { repeat } from "lit/directives/repeat.js"
import { authStore, useStore } from "../ctx.js"
import { request, subscribe } from "../http.js"
import { navigate } from "../router.js"
import "./post-item.js"
import "./toast-item.js"

const pageSize = 10

export default function ({ params }) {
    return html`<post-page .postID=${params.postID}></post-page>`
}

function PostPage({ postID }) {
    const [auth] = useStore(authStore)
    const [post, setPost] = useState(null)
    const [comments, setComments] = useState([])
    const [commentsEndCursor, setCommentsEndCursor] = useState(null)
    const [fetching, setFetching] = useState(post === null)
    const [postErr, setPostErr] = useState(null)
    const [commentsErr, setCommentsErr] = useState(null)
    const [loadingMore, setLoadingMore] = useState(false)
    const [noMoreComments, setNoMoreComments] = useState(false)
    const [queue, setQueue] = useState([])
    const [toast, setToast] = useState(null)

    const onCommentCreated = ev => {
        const payload = ev.detail
        setPost(p => ({
            ...p,
            commentsCount: p.commentsCount + 1,
        }))
        setComments(cc => [payload, ...queue, ...cc])
        setQueue([])
    }

    const onNewCommentArrive = c => {
        setQueue(cc => [c, ...cc])
        setPost(p => ({
            ...p,
            commentsCount: p.commentsCount + 1,
        }))
    }

    const onPostDeleted = () => {
        navigate("/", true)
        setToast({ type: "success", content: "post deleted" })
    }

    const onCommentDeleted = ev => {
        const payload = ev.detail
        setComments(cc => cc.filter(c => c.id !== payload.id))
        setPost(p => ({
            ...p,
            commentsCount: p.commentsCount - 1,
        }))
    }

    const onQueueBtnClick = () => {
        setComments(cc => [...queue, ...cc])
        setQueue([])
    }

    const onLoadMoreBtnClick = () => {
        if (loadingMore || noMoreComments) {
            return
        }

        setLoadingMore(true)
        fetchComments(postID, commentsEndCursor).then(({ items: comments, endCursor }) => {
            setComments(cc => [...cc, ...comments])
            setCommentsEndCursor(endCursor)

            if (comments.length < pageSize) {
                setNoMoreComments(true)
            }
        }, err => {
            const msg = "could not fetch more comments: " + err.message
            console.error(msg)
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setLoadingMore(false)
        })
    }

    useEffect(() => {
        setFetching(true)
        Promise.all([
            fetchPost(postID).catch(setPostErr),
            fetchComments(postID).catch(setCommentsErr),
        ]).then(([post, { items: comments, endCursor }]) => {
            setPost(post)
            setComments(comments)
            setCommentsEndCursor(endCursor)

            if (comments.length < pageSize) {
                setNoMoreComments(true)
            }
        }).finally(() => {
            setFetching(false)
        })
    }, [postID])

    useEffect(() => subscribeToComments(postID, onNewCommentArrive), [postID])

    return html`
        <main>
            <div class="post-wrapper">
                <div class="container">
                    ${postErr !== null ? html`
                        <p class="error" role="alert">Could not fetch post: ${postErr.message}</p>
                    ` : fetching ? html`
                        <p class="loader" aria-busy="true" aria-live="polite">Loading post... please wait.</p>
                    ` : html`
                        <post-item .post=${post} .type=${"post"} @resource-deleted=${onPostDeleted}></post-item>
                    `}
                </div>
            </div>
            <div class="container comments-wrapper">
                <h2>Comments</h2>
                ${commentsErr !== null ? html`
                    <p class="error" role="alert">Could not fetch comments: ${commentsErr.message}</p>
                ` : fetching ? html`
                    <p class="loader" aria-busy="true" aria-live="polite">Loading comments... please wait.</p>
                ` : html`
                    ${comments.length === 0 ? html`
                        <p>0 comments</p>
                    ` : html`
                        ${!noMoreComments ? html`
                            <button class="load-more-comments-btn" .disabled=${loadingMore} @click=${onLoadMoreBtnClick}>
                                ${loadingMore ? "Loading previous..." : "Load previous"}
                            </button>
                        ` : null}
                        <div class="comments" role="feed">
                            ${repeat(comments.slice().reverse(), c => c.id, c => html`<post-item .post=${c} .type=${"comment"} @resource-deleted=${onCommentDeleted}></post-item>`)}
                        </div>
                    `}
                    ${auth !== null ? html`
                        <comment-form .postID=${postID} @comment-created=${onCommentCreated}></comment-form>
                    ` : null}
                    ${queue.length !== 0 ? html`
                        <button class="queue-btn" @click=${onQueueBtnClick}>${queue.length} new comments</button>
                ` : null}
                `}
            </div>
        </main>
        ${toast !== null ? html`<toast-item .toast=${toast}></toast-item>` : null}
    `
}

// @ts-ignore
customElements.define("post-page", component(PostPage, { useShadowDOM: false }))

const reMention = /\B@([\-+\w]*)$/

function CommentForm({ postID }) {
    const [auth] = useStore(authStore)
    const [content, setContent] = useState("")
    const [fetching, setFetching] = useState(false)
    const [initialTextAreaHeight, setInitialTextAreaHeight] = useState(0)
    const textAreaRef = /** @type {import("lit/directives/ref.js").Ref<HTMLTextAreaElement>} */(createRef())
    const [toast, setToast] = useState(null)

    const dispatchCommentCreated = payload => {
        this.dispatchEvent(new CustomEvent("comment-created", { bubbles: true, detail: payload }))
    }

    const onSubmit = ev => {
        ev.preventDefault()

        setFetching(true)
        createComment(postID, { content }).then(comment => {
            comment.user = auth.user
            dispatchCommentCreated(comment)
            setContent("")
            if (textAreaRef.value !== undefined) {
                textAreaRef.value.style.height = initialTextAreaHeight + "px"
            }
        }, err => {
            const msg = "could not create comment: " + err.message
            console.error(msg)
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setFetching(false)
        })
    }

    const onTextAreaInput = () => {
        if (textAreaRef.value === undefined) {
            return
        }

        const el = /** @type {HTMLTextAreaElement} */(textAreaRef.value)

        setContent(el.value)

        el.style.height = initialTextAreaHeight + "px"
        if (el.value !== "") {
            el.style.height = Math.max(el.scrollHeight, initialTextAreaHeight) + "px"
        }
    }

    useEffect(() => {
        if (textAreaRef.value === undefined) {
            return
        }

        const el = /** @type {HTMLTextAreaElement} */(textAreaRef.value)
        const editor = new TextareaEditor(el)
        const textcomplete = new Textcomplete(editor, [{
            match: reMention,
            search: async (term, cb) => {
                cb(await fetchUsernames(term).then(page => page.items, err => {
                    console.error("could not fetch mentions usernames:", err)
                    return []
                }))
            },
            replace: username => `@${username} `,
        }])

        setInitialTextAreaHeight(el.scrollHeight)

        return () => {
            textcomplete.destroy()
        }
    }, [textAreaRef.value])

    return html`
        <form class="comment-form${content !== "" ? " has-content" : ""}" name="comment-form" @submit=${onSubmit}>
            <textarea name="content" placeholder="Say something..." maxlenght="2048" aria-label="Content" required .disabled=${fetching} .value=${content} ${ref(textAreaRef)} @input=${onTextAreaInput}></textarea>
            ${content !== "" ? html`
                <button .disabled=${fetching}>
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="paper-plane"><rect width="24" height="24" opacity="0"/><path d="M21 4a1.31 1.31 0 0 0-.06-.27v-.09a1 1 0 0 0-.2-.3 1 1 0 0 0-.29-.19h-.09a.86.86 0 0 0-.31-.15H20a1 1 0 0 0-.3 0l-18 6a1 1 0 0 0 0 1.9l8.53 2.84 2.84 8.53a1 1 0 0 0 1.9 0l6-18A1 1 0 0 0 21 4zm-4.7 2.29l-5.57 5.57L5.16 10zM14 18.84l-1.86-5.57 5.57-5.57z"/></g></g></svg>
                    <span>Comment</span>
                </button>
            ` : null}
        </form>
        ${toast !== null ? html`<toast-item .toast=${toast}></toast-item>` : null}
    `
}

// @ts-ignore
customElements.define("comment-form", component(CommentForm, { useShadowDOM: false }))

function fetchPost(postID) {
    return request("GET", "/api/posts/" + encodeURIComponent(postID))
        .then(resp => resp.body)
        .then(post => {
            post.createdAt = new Date(post.createdAt)
            return post
        })
}

function fetchComments(postID, before = "", last = pageSize) {
    return request("GET", `/api/posts/${encodeURIComponent(postID)}/comments?last=${last}&before=${before}`)
        .then(resp => resp.body)
        .then(page => {
            page.items = page.items.map(c => ({
                ...c,
                createdAt: new Date(c.createdAt)
            }))
            return page
        })
}

function createComment(postID, { content }) {
    return request("POST", `/api/posts/${encodeURIComponent(postID)}/comments`, { body: { content } })
        .then(resp => resp.body)
        .then(c => {
            c.createdAt = new Date(c.createdAt)
            return c
        })
}

function subscribeToComments(postID, cb) {
    return subscribe(`/api/posts/${encodeURIComponent(postID)}/comments`, c => {
        c.createdAt = new Date(c.createdAt)
        cb(c)
    })
}

function fetchUsernames(startingWith = "", after = "", first = pageSize) {
    return request("GET", `/api/usernames?starting_with=${encodeURIComponent(startingWith)}&after=${encodeURIComponent(after)}&first=${encodeURIComponent(first)}`)
        .then(resp => resp.body)
}

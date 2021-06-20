import { Textcomplete } from "@textcomplete/core"
import { TextareaEditor } from "@textcomplete/textarea"
import { component, html, useCallback, useEffect, useRef, useState } from "haunted"
import { nothing } from "lit-html"
import { repeat } from "lit-html/directives/repeat.js"
import { authStore, useStore } from "../ctx.js"
import { ref } from "../directives/ref.js"
import { request, subscribe } from "../http.js"
import "./intersectable-comp.js"
import "./post-item.js"
import "./toast-item.js"

const pageSize = 10

export default function () {
    return html`<home-page></home-page>`
}

function HomePage() {
    const [timeline, setTimeline] = useState([])
    const [timelineEndCursor, setTimelineEndCursor] = useState(null)
    const [fetchingTimeline, setFetchingTimeline] = useState(timeline.length === 0)
    const [err, setErr] = useState(null)
    const [loadingMore, setLoadingMore] = useState(false)
    const [noMoreTimelineItems, setNoMoreTimelineItems] = useState(false)
    const [endReached, setEndReached] = useState(false)
    const [queue, setQueue] = useState([])
    const [toast, setToast] = useState(null)

    const onTimelineItemCreated = useCallback(ev => {
        const payload = ev.detail
        setTimeline(tt => [payload, ...queue, ...tt])
        setQueue([])
    }, [queue])

    const onNewTimelineItemArrive = useCallback(ti => {
        setQueue(tt => [ti, ...tt])
    }, [])

    const onRemovedFromTimeline = useCallback(ev => {
        const payload = ev.detail
        setTimeline(tt => tt.filter(ti => ti.id !== payload.timelineItemID))
    }, [])

    const onPostDeleted = useCallback(ev => {
        const payload = ev.detail
        setTimeline(tt => tt.filter(ti => ti.post.id !== payload.id))
    }, [])

    const onQueueBtnClick = useCallback(() => {
        setTimeline(tt => [...queue, ...tt])
        setQueue([])
    }, [queue])

    const loadMore = useCallback(() => {
        if (loadingMore || noMoreTimelineItems) {
            return
        }

        setLoadingMore(true)
        fetchTimeline(timelineEndCursor).then(({ items: timeline, endCursor }) => {
            setTimeline(tt => [...tt, ...timeline])
            setTimelineEndCursor(endCursor)

            if (timeline.length < pageSize) {
                setNoMoreTimelineItems(true)
                setEndReached(true)
            }
        }, err => {
            const msg = "could not fetch more timeline items: " + err.message
            console.error(msg)
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setLoadingMore(false)
        })
    }, [loadingMore, noMoreTimelineItems, timelineEndCursor])

    useEffect(() => {
        setFetchingTimeline(true)
        fetchTimeline().then(({ items: timeline, endCursor }) => {
            setTimeline(timeline)
            setTimelineEndCursor(endCursor)

            if (timeline.length < pageSize) {
                setNoMoreTimelineItems(true)
            }
        }, err => {
            console.error("could not fetch timeline:", err)
            setErr(err)
        }).finally(() => {
            setFetchingTimeline(false)
        })
    }, [])

    useEffect(() => subscribeToTimeline(onNewTimelineItemArrive), [])

    return html`
        <main class="container home-page">
            <h1>Timeline</h1>
            <post-form @timeline-item-created=${onTimelineItemCreated}></post-form>
            ${err !== null ? html`
                <p class="error" role="alert">Could not fetch timeline: ${err.message}</p>
            ` : fetchingTimeline ? html`
                <p class="loader" aria-busy="true" aria-live="polite">Loading timeline... please wait.<p>
            ` : html`
                ${queue.length !== 0 ? html`
                    <button class="queue-btn" @click=${onQueueBtnClick}>${queue.length} new timeline items</button>
                ` : nothing}
                ${timeline.length === 0 ? html`
                    <p>0 timeline items</p>
                ` : html`
                    <div class="posts" role="feed">
                        ${repeat(timeline, ti => ti.id, ti => html`<post-item .post=${ti.post} .type=${"timeline_item"} .timelineItemID=${ti.id} @removed-from-timeline=${onRemovedFromTimeline} @resource-deleted=${onPostDeleted}></post-item>`)}
                    </div>
                    ${!noMoreTimelineItems ? html`
                        <intersectable-comp @is-intersecting=${loadMore}></intersectable-comp>
                        <p class="loader" aria-busy="true" aria-live="polite">Loading timeline... please wait.<p>
                    ` : endReached ? html`
                        <p>End reached.</p>
                    ` : nothing}
                `}
            `}
        </main>
        ${toast !== null ? html`<toast-item .toast=${toast}></toast-item>` : nothing}
    `
}

customElements.define("home-page", component(HomePage, { useShadowDOM: false }))

const reMention = /\B@([\-+\w]*)$/

function PostForm() {
    const [auth] = useStore(authStore)
    const [content, setContent] = useState("")
    const [nsfw, setNSFW] = useState(false)
    const [isSpoiler, setIsSpoiler] = useState(false)
    const [spoilerOf, setSpoilerOf] = useState("")
    const spoilerOfDialogRef = useRef(null)
    const [initialTextAreaHeight, setInitialTextAreaHeight] = useState(0)
    const textAreaRef = useRef(null)
    const textcompleteRef = useRef(null)

    const dispatchTimelineItemCreated = post => {
        this.dispatchEvent(new CustomEvent("timeline-item-created", { bubbles: true, detail: post }))
    }

    const onSubmit = useCallback(ev => {
        ev.preventDefault()
        createTimelineItem({
            content,
            spoilerOf: spoilerOf.trim() === "" ? null : spoilerOf.trim(),
            nsfw,
        }).then(ti => {
            ti.post.user = auth.user
            setContent("")
            setNSFW(false)
            setIsSpoiler(false)
            setSpoilerOf("")
            textcompleteRef.current.hide()
            textAreaRef.current.style.height = initialTextAreaHeight + "px"

            dispatchTimelineItemCreated(ti)
        })
    }, [content, nsfw, spoilerOf, auth, textAreaRef, initialTextAreaHeight])

    const onTextAreaInput = useCallback(() => {
        setContent(textAreaRef.current.value)

        textAreaRef.current.style.height = initialTextAreaHeight + "px"
        if (textAreaRef.current.value !== "") {
            textAreaRef.current.style.height = Math.max(textAreaRef.current.scrollHeight, initialTextAreaHeight) + "px"
        }
    }, [content, initialTextAreaHeight, textAreaRef])

    const onNSFWInputChange = useCallback(ev => {
        const checked = ev.currentTarget.checked
        setNSFW(checked)
    }, [])

    const onIsSpoilerInputChange = useCallback(ev => {
        const checked = ev.currentTarget.checked
        setIsSpoiler(checked)
        if (!checked) {
            setSpoilerOf("")
        } else {
            spoilerOfDialogRef.current.showModal()
        }
    }, [spoilerOfDialogRef])

    const onSpoilerOfInput = useCallback(ev => {
        setSpoilerOf(ev.currentTarget.value)
    }, [])

    const onSpoilerOfDialogClose = useCallback(() => {
        if (spoilerOf.trim() === "") {
            setSpoilerOf("")
            setIsSpoiler(false)
        }
    }, [spoilerOf])

    const onSpoilerOfFormSubmit = useCallback(ev => {
        ev.preventDefault()
        spoilerOfDialogRef.current.close()
    }, [])

    const onSpoilerOfCancelBtnClick = useCallback(() => {
        setSpoilerOf("")
        setIsSpoiler(false)
        spoilerOfDialogRef.current.close()
    }, [spoilerOfDialogRef])

    useEffect(() => {
        if (textAreaRef.current === null) {
            return
        }

        const editor = new TextareaEditor(textAreaRef.current)
        textcompleteRef.current = new Textcomplete(editor, [{
            match: reMention,
            search: async (term, cb) => {
                cb(await fetchUsernames(term).then(page => page.items, err => {
                    console.error("could not fetch mentions usernames:", err)
                    return []
                }))
            },
            replace: username => `@${username} `,
        }])

        setInitialTextAreaHeight(textAreaRef.current.scrollHeight)

        return () => {
            textcompleteRef.current.destroy()
        }
    }, [])

    return html`
        <form class="post-form${content !== "" ? " has-content" : ""}" name="post-form" @submit=${onSubmit}>
            <textarea name="content" placeholder="Write something..." maxlenght="2048" aria-label="Content" required .value=${content} .ref=${ref(textAreaRef)} @input=${onTextAreaInput}></textarea>
            ${content !== "" ? html`
                <div class="post-form-controls">
                    <div class="post-form-options">
                        <label class="switch-wrapper">
                            <input type="checkbox" role="switch" name="nsfw" .checked=${nsfw} @change=${onNSFWInputChange}>
                            <span>NSFW</span>
                        </label>
                        <label class="switch-wrapper">
                            <input type="checkbox" role="switch" name="is_spoiler" .checked=${isSpoiler} @change=${onIsSpoilerInputChange}>
                            <span>Spoiler${spoilerOf.trim() !== "" ? ` of ${spoilerOf}` : ""}</span>
                        </label>
                    </div>
                    <button>
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="paper-plane"><rect width="24" height="24" opacity="0"/><path d="M21 4a1.31 1.31 0 0 0-.06-.27v-.09a1 1 0 0 0-.2-.3 1 1 0 0 0-.29-.19h-.09a.86.86 0 0 0-.31-.15H20a1 1 0 0 0-.3 0l-18 6a1 1 0 0 0 0 1.9l8.53 2.84 2.84 8.53a1 1 0 0 0 1.9 0l6-18A1 1 0 0 0 21 4zm-4.7 2.29l-5.57 5.57L5.16 10zM14 18.84l-1.86-5.57 5.57-5.57z"/></g></g></svg>
                        <span>Publish</button>
                    </button>
                </div>
            ` : nothing}
        </form>
        <dialog .ref=${ref(spoilerOfDialogRef)} @close=${onSpoilerOfDialogClose}>
            <form method="dialog" class="spoiler-of-form" @submit=${onSpoilerOfFormSubmit}>
                <label for="spoiler-of-input">Spoiler of:</label>
                <input type="text" id="spoiler-of-input" name="spoiler_of" placeholder="Spoiler of..." maxlenght="64" autocomplete="off" .value=${spoilerOf} ?required=${isSpoiler} @input=${onSpoilerOfInput}>
                <div class="spoiler-of-controls">
                    <button>OK</button>
                    <button type="reset" @click=${onSpoilerOfCancelBtnClick}>Cancel</button>
                </div>
            </form>
        </dialog>
    `
}

customElements.define("post-form", component(PostForm, { useShadowDOM: false }))

function createTimelineItem({ content, spoilerOf, nsfw }) {
    return request("POST", "/api/timeline", { body: { content, spoilerOf, nsfw } })
        .then(resp => resp.body)
        .then(ti => {
            ti.post.createdAt = new Date(ti.post.createdAt)
            return ti
        })
}

function subscribeToTimeline(cb) {
    return subscribe("/api/timeline", ti => {
        ti.post.createdAt = new Date(ti.post.createdAt)
        cb(ti)
    })
}

function fetchTimeline(before = "", last = pageSize) {
    return request("GET", `/api/timeline?last=${encodeURIComponent(last)}&before=${encodeURIComponent(before)}`)
        .then(resp => resp.body)
        .then(page => {
            page.items = page.items.map(ti => ({
                ...ti,
                post: {
                    ...ti.post,
                    createdAt: new Date(ti.post.createdAt),
                },
            }))
            return page
        })
}

function fetchUsernames(startingWith = "", after = "", first = pageSize) {
    return request("GET", `/api/usernames?starting_with=${encodeURIComponent(startingWith)}&after=${encodeURIComponent(after)}&first=${encodeURIComponent(first)}`)
        .then(resp => resp.body)
}

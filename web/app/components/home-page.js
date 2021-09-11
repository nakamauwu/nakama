import { Textcomplete } from "@textcomplete/core"
import { TextareaEditor } from "@textcomplete/textarea"
import { component, html, useCallback, useEffect, useRef, useState } from "haunted"
import { nothing } from "lit-html"
import { repeat } from "lit-html/directives/repeat.js"
import { get as getTranslation, translate } from "lit-translate"
import { setLocalAuth } from "../auth.js"
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
    const [_, setAuth] = useStore(authStore)
    const [mode, setMode] = useState("timeline")
    const [posts, setPosts] = useState([])
    const [endCursor, setEndCursor] = useState(null)
    const [fetching, setFetching] = useState(posts.length === 0)
    const [err, setErr] = useState(null)
    const [loadingMore, setLoadingMore] = useState(false)
    const [noMore, setNoMore] = useState(false)
    const [endReached, setEndReached] = useState(false)
    const [queue, setQueue] = useState([])
    const [toast, setToast] = useState(null)

    const onTimelineItemCreated = useCallback(ev => {
        const payload = ev.detail
        setPosts(pp => [payload, ...queue, ...pp])
        setQueue([])
    }, [queue])

    const onNewTimelineItemArrive = useCallback(ti => {
        setQueue(pp => [ti, ...pp])
    }, [])

    const onNewPostArrive = useCallback(p => {
        setQueue(pp => [p, ...pp])
    }, [])

    const onRemovedFromTimeline = useCallback(ev => {
        const payload = ev.detail
        setPosts(pp => pp.filter(p => p.timelineItemID !== payload.timelineItemID))
    }, [])

    const onPostDeleted = useCallback(ev => {
        const payload = ev.detail
        setPosts(pp => pp.filter(p => p.id !== payload.id))
    }, [])

    const onQueueBtnClick = useCallback(() => {
        setPosts(pp => [...queue, ...pp])
        setQueue([])
    }, [queue])

    const loadMore = useCallback(() => {
        if (loadingMore || noMore) {
            return
        }

        setLoadingMore(true)
        const promise = mode === "timeline" ? fetchTimeline(endCursor) : fetchPosts(endCursor)
        promise.then(({ items: posts, endCursor }) => {
            setPosts(tt => [...tt, ...posts])
            setEndCursor(endCursor)

            if (posts.length < pageSize) {
                setNoMore(true)
                setEndReached(true)
            }
        }, err => {
            const msg = mode === ("timeline" ? "could not fetch more timeline items: " : "could not fetch more posts: ") + err.message
            console.error(msg)
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setLoadingMore(false)
        })
    }, [mode, loadingMore, noMore, endCursor])

    const onTimelineModeClick = useCallback(() => {
        setMode("timeline")
    }, [])

    const onPostsModeClick = useCallback(() => {
        setMode("posts")
    }, [])

    useEffect(() => {
        setPosts([])
        setEndCursor(null)
        setQueue([])
        setNoMore(false)
        setEndReached(false)

        setFetching(true)
        const promise = mode === "timeline" ? fetchTimeline() : fetchPosts()
        promise.then(({ items: posts, endCursor }) => {
            setPosts(posts)
            setEndCursor(endCursor)

            if (posts.length < pageSize) {
                setNoMore(true)
            }
        }, err => {
            console.error(mode === "timeline" ? "could not fetch timeline:" : "could not fetch posts:", err)
            if (err.name === "UnauthenticatedError") {
                setAuth(null)
                setLocalAuth(null)
            }

            setErr(err)
        }).finally(() => {
            setFetching(false)
        })
    }, [mode])

    useEffect(() => {
        return mode === "timeline" ?
            subscribeToTimeline(onNewTimelineItemArrive) :
            subscribeToPosts(onNewPostArrive)
    }, [mode])

    return html`
        <main class="container home-page">
            <h1>${mode === "timeline" ? translate("homePage.title.timeline") : translate("homePage.title.posts")}</h1>
            <post-form @timeline-item-created=${onTimelineItemCreated}></post-form>
            ${queue.length !== 0 ? html`
                <button class="queue-btn" @click=${onQueueBtnClick}>${mode === "timeline"
                ? (queue.length === 1
                    ? translate("homePage.queueBtn.newTimelineItem")
                    : translate("homePage.queueBtn.newTimelineItems", { length: queue.length }))
                : (queue.length === 1
                    ? translate("homePage.queueBtn.newPost")
                    : translate("homePage.queueBtn.newPosts", { length: queue.length }))}
                </button>
            ` : nothing}
            <div role="tablist">
                <button role="tab" id="${mode}-tab" aria-controls="${mode}-tabpanel" aria-selected=${String(mode === "timeline")} @click=${onTimelineModeClick}>
                    ${translate("homePage.tabs.timeline")}
                </button>
                <button role="tab" id="${mode}-tab" aria-controls="${mode}-tabpanel" aria-selected=${String(mode === "posts")} @click=${onPostsModeClick}>
                    ${translate("homePage.tabs.posts")}
                </button>
            </div>
            ${err !== null ? html`
                <p class="error" role="alert">${mode === "timeline"
                ? translate("homePage.err.timeline")
                : translate("homePage.err.posts")}
                    ${translate(err.name)}
                </p>
            ` : fetching ? html`
                <p class="loader" aria-busy="true" aria-live="polite">${mode === "timeline"
                ? translate("homePage.loading.timeline")
                : translate("homePage.loading.posts")}
                <p>
            ` : html`
                <div role="tabpanel" id="${mode}-tabpanel" aria-labelledby="${mode}-tab">
                ${posts.length === 0 ? html`
                    <p>${mode === "timeline"
                    ? translate("homePage.empty.timeline")
                    : translate("homePage.empty.posts")}
                    </p>
                ` : html`
                    <div class="posts" role="feed">
                        ${repeat(posts, p => p.id, p => html`<post-item .post=${p} .type=${mode === "timeline" ? "timeline_item" : "post"}
                            @removed-from-timeline=${onRemovedFromTimeline}
                            @resource-deleted=${onPostDeleted}></post-item>`)}
                    </div>
                    ${!noMore ? html`
                        <intersectable-comp @is-intersecting=${loadMore}></intersectable-comp>
                        <p class="loader" aria-busy="true" aria-live="polite">${mode === "timeline"
                        ? translate("homePage.loading.timeline")
                        : translate("homePage.loading.posts")}
                        <p>
                    ` : endReached ? html`
                        <p>${translate("homePage.end")}</p>
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
    const [fetching, setFetching] = useState(false)
    const [nsfw, setNSFW] = useState(false)
    const [isSpoiler, setIsSpoiler] = useState(false)
    const [spoilerOf, setSpoilerOf] = useState("")
    const spoilerOfDialogRef = useRef(null)
    const [initialTextAreaHeight, setInitialTextAreaHeight] = useState(0)
    const textAreaRef = useRef(null)
    const textcompleteRef = useRef(null)
    const [toast, setToast] = useState(null)

    const dispatchTimelineItemCreated = post => {
        this.dispatchEvent(new CustomEvent("timeline-item-created", { bubbles: true, detail: post }))
    }

    const onSubmit = useCallback(ev => {
        ev.preventDefault()

        setFetching(true)
        createTimelineItem({
            content,
            spoilerOf: spoilerOf.trim() === "" ? null : spoilerOf.trim(),
            nsfw,
        }).then(ti => {
            ti.user = auth.user
            setContent("")
            setNSFW(false)
            setIsSpoiler(false)
            setSpoilerOf("")
            textcompleteRef.current.hide()

            dispatchTimelineItemCreated(ti)
        }, err => {
            const msg = getTranslation("postForm.err") + " " + translate(err.name)
            console.error(msg)
            setToast({ type: "error", content: msg })
        }).finally(() => {
            setFetching(false)
        })
    }, [content, nsfw, spoilerOf, auth, textAreaRef, initialTextAreaHeight])

    const onTextAreaInput = useCallback(() => {
        setContent(textAreaRef.current.value)
    }, [content, textAreaRef])

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
        if (spoilerOfDialogRef.current !== null && !("HTMLDialogElement" in window || "showModal" in spoilerOfDialogRef.current)) {
            import("dialog-polyfill").then(m => m.default).then(dialogPolyfill => {
                dialogPolyfill.registerDialog(spoilerOfDialogRef.current)
            }).catch(err => {
                console.error("could not import dialog polyfill:", err)
            })
        }
    }, [spoilerOfDialogRef.current])

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
    }, [textAreaRef, textcompleteRef])

    // Share Target.
    useEffect(() => {
        const params = new URLSearchParams(window.location.search.slice(1))
        const preContent = []
        let cleanup = false
        if (params.has("text")) {
            cleanup = true
            const text = decodeURIComponent(params.get("text")).trim()
            if (text !== "") {
                preContent.push(text)
            }
        }
        if (params.has("url")) {
            cleanup = true
            const url = decodeURIComponent(params.get("url"))
            if (url !== "") {
                preContent.push(url)
            }
        }

        if (preContent.length !== 0) {
            setContent(preContent.join(" "))
            if (textAreaRef.current !== null) {
                textAreaRef.current.focus()
            }
        }

        if (cleanup) {
            history.replaceState(history.state, document.title, "/")
        }
    }, [textAreaRef])

    useEffect(() => {
        if (textAreaRef.current === null) {
            return
        }

        textAreaRef.current.style.height = initialTextAreaHeight + "px"
        if (textAreaRef.current.value !== "") {
            textAreaRef.current.style.height = Math.max(textAreaRef.current.scrollHeight, initialTextAreaHeight) + "px"
        }
    }, [content, initialTextAreaHeight, textAreaRef])

    return html`
        <form class="post-form${content !== "" ? " has-content" : ""}" name="post-form" @submit=${onSubmit}>
            <textarea name="content"
                placeholder="${translate("postForm.placeholder")}"
                maxlenght="2048"
                aria-label="${translate("postForm.textAreaLabel")}"
                required
                .value=${content}
                .disabled=${fetching}
                .ref=${ref(textAreaRef)}
                @input=${onTextAreaInput}></textarea>
            ${content !== "" ? html`
            <div class="post-form-controls">
                <div class="post-form-options">
                    <label class="switch-wrapper">
                        <input type="checkbox" role="switch" name="nsfw" .disabled=${fetching} .checked=${nsfw} @change=${onNSFWInputChange}>
                        <span>${translate("postForm.nsfwLabel")}</span>
                    </label>
                    <label class="switch-wrapper">
                        <input type="checkbox" role="switch" name="is_spoiler" .disabled=${fetching} .checked=${isSpoiler} @change=${onIsSpoilerInputChange}>
                        <span>${spoilerOf.trim() === ""
                ? translate("postForm.spoilerLabel")
                : translate("postForm.spoilerOfLabel", { value: spoilerOf.trim() })}
                        </span>
                    </label>
                </div>
                <button class="submit-btn" .disabled=${fetching}>
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                        <g data-name="Layer 2">
                            <g data-name="paper-plane">
                                <rect width="24" height="24" opacity="0" />
                                <path
                                    d="M21 4a1.31 1.31 0 0 0-.06-.27v-.09a1 1 0 0 0-.2-.3 1 1 0 0 0-.29-.19h-.09a.86.86 0 0 0-.31-.15H20a1 1 0 0 0-.3 0l-18 6a1 1 0 0 0 0 1.9l8.53 2.84 2.84 8.53a1 1 0 0 0 1.9 0l6-18A1 1 0 0 0 21 4zm-4.7 2.29l-5.57 5.57L5.16 10zM14 18.84l-1.86-5.57 5.57-5.57z" />
                            </g>
                        </g>
                    </svg>
                    <span>${translate("postForm.submit")}</button>
                </button>
            </div>
            ` : nothing}
        </form>
        <dialog .ref=${ref(spoilerOfDialogRef)} @close=${onSpoilerOfDialogClose}>
            <form method="dialog" class="spoiler-of-form" @submit=${onSpoilerOfFormSubmit}>
                <label for="spoiler-of-input">${translate("postForm.dialog.spoilerOfLabel")}</label>
                <input type="text"
                    id="spoiler-of-input"
                    name="spoiler_of"
                    placeholder="${translate("postForm.dialog.spoilerOfPlaceholder")}"
                    maxlenght="64"
                    autocomplete="off"
                    .value=${spoilerOf}
                    ?required=${isSpoiler}
                    @input=${onSpoilerOfInput}>
                <div class="spoiler-of-controls">
                    <button>${translate("postForm.dialog.ok")}</button>
                    <button type="reset" @click=${onSpoilerOfCancelBtnClick}>${translate("postForm.dialog.cancel")}</button>
                </div>
            </form>
        </dialog>
        ${toast !== null ? html`<toast-item .toast=${toast}></toast-item>` : nothing}
    `
}

customElements.define("post-form", component(PostForm, { useShadowDOM: false }))

function createTimelineItem({ content, spoilerOf, nsfw }) {
    return request("POST", "/api/timeline", { body: { content, spoilerOf, nsfw } })
        .then(resp => resp.body)
        .then(ti => {
            ti.createdAt = new Date(ti.createdAt)
            return ti
        })
}

function subscribeToTimeline(cb) {
    return subscribe("/api/timeline", ti => {
        ti.createdAt = new Date(ti.createdAt)
        cb(ti)
    })
}

function fetchTimeline(before = "", last = pageSize) {
    return request("GET", `/api/timeline?last=${encodeURIComponent(last)}&before=${encodeURIComponent(before)}`)
        .then(resp => resp.body)
        .then(page => {
            page.items = page.items.map(ti => ({
                ...ti,
                createdAt: new Date(ti.createdAt),
            }))
            return page
        })
}

function subscribeToPosts(cb) {
    return subscribe("/api/posts", p => {
        p.createdAt = new Date(p.createdAt)
        cb(p)
    })
}

function fetchPosts(before = "", last = pageSize) {
    return request("GET", `/api/posts?last=${encodeURIComponent(last)}&before=${encodeURIComponent(before)}`)
        .then(resp => resp.body)
        .then(page => {
            page.items = page.items.map(p => ({
                ...p,
                createdAt: new Date(p.createdAt),
            }))
            return page
        })
}

function fetchUsernames(startingWith = "", after = "", first = pageSize) {
    return request("GET", `/api/usernames?starting_with=${encodeURIComponent(startingWith)}&after=${encodeURIComponent(after)}&first=${encodeURIComponent(first)}`)
        .then(resp => resp.body)
}

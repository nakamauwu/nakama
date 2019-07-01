import { getAuthUser } from "../auth.js"
import { doGet, doPost, subscribe } from "../http.js"
import renderList from "./list.js"
import renderPost from "./post.js"
import { smartTrim } from "../utils.js"

const PAGE_SIZE = 10
let timeline = /** @type {import("../types.js").TimelineItem[]} */ (null)

const template = document.createElement("template")
template.innerHTML = `
    <div class="container">
        <h1>Timeline</h1>
        <form id="post-form" class="post-form">
            <textarea name="content" placeholder="Write something..." maxlength="480" required></textarea>
            <button class="post-form-button" hidden>
                <svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g data-name="Layer 2"><g data-name="paper-plane"><rect width="24" height="24" opacity="0"/><path d="M21 4a1.31 1.31 0 0 0-.06-.27v-.09a1 1 0 0 0-.2-.3 1 1 0 0 0-.29-.19h-.09a.86.86 0 0 0-.31-.15H20a1 1 0 0 0-.3 0l-18 6a1 1 0 0 0 0 1.9l8.53 2.84 2.84 8.53a1 1 0 0 0 1.9 0l6-18A1 1 0 0 0 21 4zm-4.7 2.29l-5.57 5.57L5.16 10zM14 18.84l-1.86-5.57 5.57-5.57z"/></g></g></svg>
                <span>Publish</span>
            </button>
        </form>
        <div id="timeline-outlet" class="posts-wrapper"></div>
    </div>
`

export default async function renderHomePage() {
    if (timeline === null || timeline.length === 0) {
        timeline = await fetchTimeline()
    }
    const list = renderList({
        items: timeline,
        loadMoreFunc: fetchTimeline,
        pageSize: PAGE_SIZE,
        renderItem: renderTimelineItem,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const postForm = /** @type {HTMLFormElement} */ (page.getElementById("post-form"))
    const postFormTextArea = postForm.querySelector("textarea")
    const postFormButton = postForm.querySelector("button")
    const timelineOutlet = page.getElementById("timeline-outlet")
    let initialPostFormTextAreaHeight = /** @type {string=} */ (undefined)

    /**
     * @param {Event} ev
     */
    const onPostFormSubmit = async ev => {
        ev.preventDefault()
        const content = smartTrim(postFormTextArea.value)
        if (content === "") {
            postFormTextArea.setCustomValidity("Empty")
            postFormTextArea.reportValidity()
            return
        }

        postFormTextArea.disabled = true
        postFormButton.disabled = true

        try {
            const timelineItem = await publishPost({ content })

            list.enqueue(timelineItem)
            list.flush()

            postForm.reset()
            postFormButton.hidden = true
        } catch (err) {
            console.error(err)
            alert(err.message)
            setTimeout(() => {
                postFormTextArea.focus()
            })
        } finally {
            postFormTextArea.disabled = false
            postFormButton.disabled = false
        }
    }

    const onPostFormTextAreaInput = () => {
        postFormTextArea.setCustomValidity("")
        postFormButton.hidden = smartTrim(postFormTextArea.value) === ""
        if (initialPostFormTextAreaHeight === undefined) {
            initialPostFormTextAreaHeight = postFormTextArea.style.height
        }
        postFormTextArea.style.height = initialPostFormTextAreaHeight
        postFormTextArea.style.height = postFormTextArea.scrollHeight + "px"
    }

    const onTimelineItemArrive = list.enqueue

    const unsubscribeFromTimeline = subscribeToTimeline(onTimelineItemArrive)

    const onPageDisconnect = () => {
        unsubscribeFromTimeline()
        list.teardown()
    }

    timelineOutlet.appendChild(list.el)

    postForm.addEventListener("submit", onPostFormSubmit)
    postFormTextArea.addEventListener("input", onPostFormTextAreaInput)
    page.addEventListener("disconnect", onPageDisconnect)

    return page
}

/**
 * @param {import("../types.js").TimelineItem} timelineItem
 */
function renderTimelineItem(timelineItem) {
    return renderPost(timelineItem.post, timelineItem.id)
}

/**
 * @param {import("../types.js").CreatePostInput} input
 * @returns {Promise<import("../types.js").TimelineItem>}
 */
async function publishPost(input) {
    const timelineItem = await doPost("/api/posts", input)
    timelineItem.post.user = getAuthUser()
    return timelineItem
}

/**
 * @param {bigint=} before
 * @returns {Promise<import("../types.js").TimelineItem[]>}
 */
function fetchTimeline(before = 0n) {
    return doGet(`/api/timeline?before=${before}&last=${PAGE_SIZE}`)
}

/**
 * @param {function(import("../types.js").TimelineItem): any} cb
 */
function subscribeToTimeline(cb) {
    return subscribe("/api/timeline", cb)
}

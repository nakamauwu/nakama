import { getAuthUser } from '../auth.js';
import { doGet, doPost, subscribe } from '../http.js';
import renderList from './list.js';
import renderPost from './post.js';

const PAGE_SIZE = 10

const template = document.createElement('template')
template.innerHTML = `
    <div class="container">
        <h1>Timeline</h1>
        <form id="post-form" class="post-form">
            <textarea placeholder="Write something..." maxlength="480" required></textarea>
            <button class="post-form-button" hidden>Publish</button>
        </form>
        <div id="timeline-outlet" class="posts-wrapper"></div>
    </div>
`

export default async function renderHomePage() {
    const timeline = await fetchTimeline()
    const list = renderList({
        items: timeline,
        loadMoreFunc: fetchTimeline,
        pageSize: PAGE_SIZE,
        renderItem: renderTimelineItem,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const postForm = /** @type {HTMLFormElement} */ (page.getElementById('post-form'))
    const postFormTextArea = postForm.querySelector('textarea')
    const postFormButton = postForm.querySelector('button')
    const timelineOutlet = page.getElementById('timeline-outlet')

    /**
     * @param {Event} ev
     */
    const onPostFormSubmit = async ev => {
        ev.preventDefault()
        const content = postFormTextArea.value

        postFormTextArea.disabled = true
        postFormButton.disabled = true

        try {
            const timelineItem = await publishPost({ content })

            list.addItemToQueue(timelineItem)
            list.flushQueue()

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
        postFormButton.hidden = postFormTextArea.value === ''
    }

    const onTimelineItemArrive = list.addItemToQueue

    const unsubscribeFromTimeline = subscribeToTimeline(onTimelineItemArrive)

    const onPageDisconnect = () => {
        unsubscribeFromTimeline()
        list.teardown()
    }

    timelineOutlet.appendChild(list.el)

    postForm.addEventListener('submit', onPostFormSubmit)
    postFormTextArea.addEventListener('input', onPostFormTextAreaInput)
    page.addEventListener('disconnect', onPageDisconnect)

    return page
}

/**
 * @param {import('../types.js').TimelineItem} timelineItem
 */
function renderTimelineItem(timelineItem) {
    return renderPost(timelineItem.post, timelineItem.id)
}

/**
 * @param {import('../types.js').CreatePostInput} input
 * @returns {Promise<import('../types.js').TimelineItem>}
 */
async function publishPost(input) {
    const timelineItem = await doPost('/api/posts', input)
    timelineItem.post.user = getAuthUser()
    return timelineItem
}

/**
 * @param {bigint=} before
 * @returns {Promise<import('../types.js').TimelineItem[]>}
 */
function fetchTimeline(before = 0n) {
    return doGet(`/api/timeline?before=${before}&last=${PAGE_SIZE}`)
}

/**
 * @param {function(import('../types.js').TimelineItem): any} cb
 */
function subscribeToTimeline(cb) {
    return subscribe('/api/timeline', cb)
}

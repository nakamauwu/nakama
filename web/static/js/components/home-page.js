import { getAuthUser } from '../auth.js';
import { doGet, doPost, subscribe } from '../http.js';
import { makeInfiniteList } from './behaviors/feed.js';
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
        <button id="flush-queue-button"
            class="flush-posts-queue"
            aria-live="assertive"
            aria-atomic="true"
            hidden></button>
        <div id="timeline-div"></div>
    </div>
`

export default async function renderHomePage() {
    const timelineQueue = /** @type {import('../types.js').TimelineItem[]} */ ([])
    const timeline = await http.fetchTimeline()

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const postForm = /** @type {HTMLFormElement} */ (page.getElementById('post-form'))
    const postFormTextArea = postForm.querySelector('textarea')
    const postFormButton = postForm.querySelector('button')
    const flushQueueButton = /** @type {HTMLButtonElement} */ (page.getElementById('flush-queue-button'))
    const timelineDiv = /** @type {HTMLDivElement} */ (page.getElementById('timeline-div'))

    /**
     * @param {Event} ev
     */
    const onPostFormSubmit = async ev => {
        ev.preventDefault()
        const content = postFormTextArea.value

        postFormTextArea.disabled = true
        postFormButton.disabled = true

        try {
            const timelineItem = await http.publishPost({ content })

            flushQueue()

            timeline.unshift(timelineItem)
            timelineDiv.insertAdjacentElement('afterbegin', renderTimelineItem(timelineItem))

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

    const flushQueue = () => {
        let timelineItem = timelineQueue.pop()

        while (timelineItem !== undefined) {
            timeline.unshift(timelineItem)
            timelineDiv.insertAdjacentElement('afterbegin', renderTimelineItem(timelineItem))

            timelineItem = timelineQueue.pop()
        }

        flushQueueButton.hidden = true
    }

    const onFlushQueueButtonClick = flushQueue

    const timelineTeardown = makeInfiniteList(timelineDiv, {
        items: timeline,
        getMoreItems: http.fetchTimeline,
        renderItem: renderTimelineItem,
        pageSize: PAGE_SIZE,
    })

    /**
     * @param {import('../types.js').TimelineItem} timelineItem
     */
    const onTimelineItemArrive = timelineItem => {
        timelineQueue.unshift(timelineItem)

        flushQueueButton.textContent = timelineQueue.length + ' new posts'
        flushQueueButton.hidden = false
    }

    const unsubscribeFromTimeline = http.subscribeToTimeline(onTimelineItemArrive)

    const onPageDisconnect = () => {
        unsubscribeFromTimeline()
        timelineTeardown()
    }

    postForm.addEventListener('submit', onPostFormSubmit)
    postFormTextArea.addEventListener('input', onPostFormTextAreaInput)
    flushQueueButton.addEventListener('click', onFlushQueueButtonClick)
    page.addEventListener('disconnect', onPageDisconnect)

    return page
}

/**
 * @param {import('../types.js').TimelineItem} timelineItem
 */
function renderTimelineItem(timelineItem) {
    return renderPost(timelineItem.post, timelineItem.id)
}

const http = {
    /**
     * @param {import('../types.js').CreatePostInput} input
     * @returns {Promise<import('../types.js').TimelineItem>}
     */
    publishPost: input => doPost('/api/posts', input).then(timelineItem => {
        timelineItem.post.user = getAuthUser()
        return timelineItem
    }),

    /**
     * @param {bigint=} before
     * @returns {Promise<import('../types.js').TimelineItem[]>}
     */
    fetchTimeline: (before = 0n) => doGet(`/api/timeline?before=${before}&last=${PAGE_SIZE}`),

    /**
     * @param {function(import('../types.js').TimelineItem): any} cb
     */
    subscribeToTimeline: cb => subscribe('/api/timeline', cb),
}

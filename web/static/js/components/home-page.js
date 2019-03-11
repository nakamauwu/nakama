import { getAuthUser } from '../auth.js';
import { doGet, doPost, subscribe } from '../http.js';
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
        <ol id="timeline-list" class="post-list"></ol>
        <button id="load-more-button" class="load-more-posts-button" hidden>Load more</button>
    </div>
`

export default async function renderHomePage() {
    const timelineQueue = /** @type {import('../types.js').TimelineItem[]} */ ([])
    const timeline = await http.timeline()

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const postForm = /** @type {HTMLFormElement} */ (page.getElementById('post-form'))
    const postFormTextArea = postForm.querySelector('textarea')
    const postFormButton = postForm.querySelector('button')
    const flushQueueButton = /** @type {HTMLButtonElement} */ (page.getElementById('flush-queue-button'))
    const timelineList = /** @type {HTMLOListElement} */ (page.getElementById('timeline-list'))
    const loadMoreButton = /** @type {HTMLButtonElement} */ (page.getElementById('load-more-button'))

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
            timelineList.insertAdjacentElement('afterbegin', renderPost(timelineItem.post))

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
            timelineList.insertAdjacentElement('afterbegin', renderPost(timelineItem.post))

            timelineItem = timelineQueue.pop()
        }

        flushQueueButton.hidden = true
    }

    const onFlushQueueButtonClick = flushQueue

    const onLoadMoreButtonClick = async () => {
        loadMoreButton.disabled = true

        try {
            const lastTimelineItem = timeline[timeline.length - 1]
            const newTimelineItems = await http.timeline(lastTimelineItem.id)

            timeline.push(...newTimelineItems)
            for (const timelineItem of newTimelineItems) {
                timelineList.appendChild(renderPost(timelineItem.post))
            }

            if (newTimelineItems.length < PAGE_SIZE) {
                loadMoreButton.removeEventListener('click', onLoadMoreButtonClick)
                loadMoreButton.remove()
            }
        } catch (err) {
            console.error(err)
            alert(err.message)
        } finally {
            loadMoreButton.disabled = false
        }
    }

    /**
     * @param {import('../types.js').TimelineItem} timelineItem
     */
    const onTimelineItemArrive = timelineItem => {
        timelineQueue.unshift(timelineItem)

        flushQueueButton.textContent = timelineQueue.length + ' new posts'
        flushQueueButton.hidden = false
    }

    const unsubscribeFromTimeline = http.subscribeToTimeline(onTimelineItemArrive)

    const onPageDisconnect = unsubscribeFromTimeline

    for (const timelineItem of timeline) {
        timelineList.appendChild(renderPost(timelineItem.post))
    }

    postForm.addEventListener('submit', onPostFormSubmit)
    postFormTextArea.addEventListener('input', onPostFormTextAreaInput)
    flushQueueButton.addEventListener('click', onFlushQueueButtonClick)
    if (timeline.length == PAGE_SIZE) {
        loadMoreButton.hidden = false
        loadMoreButton.addEventListener('click', onLoadMoreButtonClick)
    }
    page.addEventListener('disconnect', onPageDisconnect)

    return page
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
    timeline: (before = 0n) => doGet(`/api/timeline?before=${before}&last=${PAGE_SIZE}`),

    /**
     * @param {function(import('../types.js').TimelineItem): any} cb
     */
    subscribeToTimeline: cb => subscribe('/api/timeline', cb),
}

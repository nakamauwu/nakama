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
        <div id="timeline-div" class="posts-wrapper"></div>
    </div>
`

export default async function renderHomePage() {
    const timeline = await http.fetchTimeline()
    const list = renderList({
        items: timeline,
        fetchMoreItems: http.fetchTimeline,
        pageSize: PAGE_SIZE,
        renderItem: renderTimelineItem,
    })

    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    const postForm = /** @type {HTMLFormElement} */ (page.getElementById('post-form'))
    const postFormTextArea = postForm.querySelector('textarea')
    const postFormButton = postForm.querySelector('button')
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

    const unsubscribeFromTimeline = http.subscribeToTimeline(onTimelineItemArrive)

    const onPageDisconnect = () => {
        unsubscribeFromTimeline()
        list.teardown()
    }

    timelineDiv.appendChild(list.el)

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

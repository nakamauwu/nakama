/**
 * @param {HTMLElement} feed
 * @param {Object} opts
 * @param {any[]} opts.items
 * @param {function(any):Element} opts.renderItem
 * @param {function(any):Promise<any[]>} opts.getMoreItems
 * @param {number} opts.pageSize
 * @param {function(any):any=} opts.getID
 * @param {function(Error):any=} opts.onError
 */
export function makeInfiniteList(feed, opts) {
    let index = 0
    const observer = new IntersectionObserver(([entry]) => {
        if (entry.isIntersecting) {
            loadMore()
        }
    }, { rootMargin: '25%' })

    const target = document.createElement('div')
    target.style.visibility = 'hidden'
    target.setAttribute('aria-hidden', 'true')

    const loadMore = async () => {
        feed.setAttribute('aria-busy', 'true')

        try {
            const lastItem = opts.items[opts.items.length - 1]
            const newItems = await opts.getMoreItems(lastItem === undefined ? undefined
                : typeof opts.getID === 'function'
                    ? opts.getID(lastItem)
                    : lastItem['id']
            )
            opts.items.push(...newItems)
            for (const item of newItems) {
                feed.appendChild(opts.renderItem(item))
            }
            if (newItems.length < opts.pageSize) {
                teardown()
            }
        } catch (err) {
            if (typeof opts.onError === 'function') {
                opts.onError(err)
            } else {
                console.error(err)
            }
        } finally {
            feed.setAttribute('aria-busy', 'false')
        }
    }

    /**
     * @param {KeyboardEvent} ev
     */
    const onFeedKeyDown = ev => {
        switch (ev.key) {
            case 'ArrowLeft':
            case 'ArrowUp':
                index = Math.max(0, index - 1)
                const prev = feed.children[index]
                if (prev instanceof HTMLElement) {
                    prev.setAttribute('tabindex', '-1')
                    prev.focus()
                }
                break
            case 'ArrowRight':
            case 'ArrowDown':
                index = Math.min(feed.children.length - 1, index + 1)
                const next = feed.children[index]
                if (next instanceof HTMLElement) {
                    next.setAttribute('tabindex', '-1')
                    next.focus()
                }
                break
        }
    }

    feed.setAttribute('role', 'feed')
    feed.setAttribute('aria-busy', 'false')

    for (const item of opts.items) {
        feed.appendChild(opts.renderItem(item))
    }
    feed.insertAdjacentElement('afterend', target)

    feed.addEventListener('keydown', onFeedKeyDown)
    if (opts.items.length === opts.pageSize) {
        setTimeout(() => {
            observer.observe(target)
        })
    }

    const teardown = () => {
        observer.unobserve(target)
        observer.disconnect()
        target.remove()
    }

    return teardown
}

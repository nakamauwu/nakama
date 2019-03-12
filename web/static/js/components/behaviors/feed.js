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
    let loading = false

    const target = document.createElement('div')
    target.style.visibility = 'hidden'
    target.setAttribute('aria-hidden', 'true')

    const mo = new MutationObserver(mutations => {
        for (const mutation of mutations) {
            for (const node of mutation.addedNodes) {
                if (node instanceof Element) {
                    node.setAttribute('tabindex', '-1')
                }
            }
        }
    })

    const io = new IntersectionObserver(entries => {
        for (const entry of entries) {
            if (entry.target === target && entry.isIntersecting) {
                loadMore()
            }
        }
    }, { rootMargin: '25%' })

    const addPagination = () => {
        feed.insertAdjacentElement('afterend', target)
        setTimeout(() => {
            io.observe(target)
        })
    }

    const removePagination = () => {
        io.unobserve(target)
        io.disconnect()
        target.remove()
    }

    const loadMore = async () => {
        if (loading) {
            return
        }

        loading = true
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
                removePagination()
            }
        } catch (err) {
            if (typeof opts.onError === 'function') {
                opts.onError(err)
            } else {
                console.error(err)
            }
        } finally {
            loading = false
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
                    prev.focus()
                }
                break
            case 'ArrowRight':
            case 'ArrowDown':
                index = Math.min(feed.children.length - 1, index + 1)
                const next = feed.children[index]
                if (next instanceof HTMLElement) {
                    next.focus()
                }
                break
        }
    }

    feed.setAttribute('role', 'feed')
    feed.setAttribute('aria-busy', 'false')

    for (const item of opts.items) {
        const child = opts.renderItem(item)
        child.setAttribute('tabindex', '-1')
        feed.appendChild(child)
    }

    feed.addEventListener('keydown', onFeedKeyDown)
    mo.observe(feed, { childList: true })
    if (opts.items.length === opts.pageSize) {
        addPagination()
    }

    return () => {
        mo.disconnect()
        removePagination()
    }
}

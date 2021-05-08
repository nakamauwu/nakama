function renderFeed() {
    let index = 0

    const feed = document.createElement("div")
    feed.setAttribute("role", "feed")
    feed.setAttribute("aria-busy", "false")

    /**
     * @param {MutationRecord[]} mutations
     */
    const mutationCallback = mutations => {
        for (const mutation of mutations) {
            for (const node of mutation.addedNodes) {
                if (node instanceof Element) {
                    node.setAttribute("tabindex", "-1")
                }
            }
        }
    }

    /**
     * @param {KeyboardEvent} ev
     */
    const onKeyDown = ev => {
        if (ev.key === "ArrowUp") {
            index = Math.max(0, index - 1)
        } else if (ev.key === "ArrowDown") {
            index = Math.min(feed.children.length - 1, index + 1)
        } else if (ev.ctrlKey && ev.key === "Home") {
            index = 0
        } else if (ev.ctrlKey && ev.key === "End") {
            index = feed.children.length - 1
        } else {
            return
        }

        const child = feed.children[index]
        if (child instanceof HTMLElement) {
            child.focus()
        }
    }

    const setLoading = val => {
        feed.setAttribute("aria-busy", String(Boolean(val)))
    }

    const teardown = () => {
        mo.disconnect()
        feed.removeAttribute("aria-busy")
        feed.removeEventListener("keydown", onKeyDown)
    }

    const mo = new MutationObserver(mutationCallback)
    mo.observe(feed, { childList: true })
    feed.addEventListener("keydown", onKeyDown)

    return { el: feed, setLoading, teardown }
}

/**
 * @param {Object} opts
 * @param {import("../types.js").Page<any>} opts.page
 * @param {function(any): HTMLElement} opts.renderItem
 * @param {function(any):Promise<import("../types.js").Page<any>>} opts.loadMoreFunc
 * @param {string=} opts.loadMoreText
 * @param {number} opts.pageSize
 * @param {function(number):string=} opts.newItemsMessageFunc
 * @param {boolean=} opts.reverse
 * @param {boolean=} opts.forward
 * @param {function(Error):any=} opts.onError
 * @param {Node=} opts.noContent
 */
export default function renderList(opts) {
    const queue = []
    const feed = renderFeed()
    let loadMoreButton = /** @type {HTMLButtonElement=} */ (null)
    let queueButton = /** @type {HTMLButtonElement=} */ (null)
    let noContentRendered = false

    const cleanupFeed = () => {
        while (feed.el.firstElementChild !== null) {
            feed.el.removeChild(feed.el.lastElementChild)
        }
    }

    const enqueue = item => {
        if (queueButton === null) {
            queueButton = document.createElement("button")
            queueButton.className = "queue-button"
            queueButton.setAttribute("aria-live", "assertive")
            queueButton.setAttribute("aria-atomic", "true")
            queueButton.addEventListener("click", flush)
            feed.el.insertAdjacentElement(opts.reverse ? "afterend" : "beforebegin", queueButton)
        }

        queue.unshift(item)

        queueButton.textContent = typeof opts.newItemsMessageFunc === "function"
            ? opts.newItemsMessageFunc(queue.length)
            : queue.length + " new item" + (queue.length === 1 ? "" : "s")
        queueButton.hidden = false
    }

    const flush = () => {
        if (noContentRendered) {
            cleanupFeed()
        }

        let item = queue.pop()
        while (item !== undefined) {
            opts.page.items.unshift(item)

            if (opts.reverse) {
                feed.el.appendChild(opts.renderItem(item))
            } else {
                feed.el.insertAdjacentElement("afterbegin", opts.renderItem(item))
            }

            item = queue.pop()
        }

        if (queueButton !== null) {
            queueButton.hidden = true
        }
    }

    const teardown = () => {
        feed.teardown()
        if (loadMoreButton !== null) {
            loadMoreButton.removeEventListener("click", onLoadMoreButtonClick)
        }
        if (queueButton !== null) {
            queueButton.removeEventListener("click", flush)
        }
    }

    const onLoadMoreButtonClick = async () => {
        feed.setLoading(true)
        loadMoreButton.disabled = true

        try {
            const cursor = opts.forward ? opts.page.startCursor : opts.page.endCursor
            const page = await opts.loadMoreFunc(cursor)

            opts.page.items.push(...page.items)
            opts.page.startCursor = page.startCursor
            opts.page.endCursor = page.endCursor

            for (const item of page.items) {
                if (opts.reverse) {
                    feed.el.insertAdjacentElement("afterbegin", opts.renderItem(item))
                } else {
                    feed.el.appendChild(opts.renderItem(item))
                }
            }

            if (page.items.length < opts.pageSize) {
                loadMoreButton.removeEventListener("click", onLoadMoreButtonClick)
                loadMoreButton.remove()
            }
        } catch (err) {
            if (typeof opts.onError === "function") {
                opts.onError(err)
            } else {
                console.error(err)
            }
        } finally {
            feed.setLoading(false)
            loadMoreButton.disabled = false
        }
    }

    if (opts.page.items === null || opts.page.items.length === 0) {
        if (opts.noContent instanceof Node) {
            feed.el.appendChild(opts.noContent)
            noContentRendered = true
        }
    } else {
        for (const item of opts.page.items) {
            if (opts.reverse) {
                feed.el.insertAdjacentElement("afterbegin", opts.renderItem(item))
            } else {
                feed.el.appendChild(opts.renderItem(item))
            }
        }
    }


    if (opts.page.items.length === opts.pageSize) {
        setTimeout(() => {
            loadMoreButton = document.createElement("button")
            loadMoreButton.className = "load-more-button"
            loadMoreButton.textContent = typeof opts.loadMoreText === "string" ? opts.loadMoreText : "Load more"
            loadMoreButton.addEventListener("click", onLoadMoreButtonClick)
            feed.el.insertAdjacentElement(opts.reverse ? "beforebegin" : "afterend", loadMoreButton)
        })
    }

    return { el: feed.el, enqueue, flush, teardown }
}

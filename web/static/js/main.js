import { guard } from './auth.js';
import renderErrorPage from './components/error-page.js';
import { createRouter } from './lib/router.js';

let currentPage
const disconnectEvent = new CustomEvent('disconnect')
const r = createRouter()
r.route('/', guard(view('home'), view('access')))
r.route(/\//, view('not-found'))
r.subscribe(renderInto(document.querySelector('main')))
r.install()

function view(name) {
    return (...args) => import(`/js/components/${name}-page.js`)
        .then(m => m.default(...args))
}

/**
 * @param {Element} target
 */
function renderInto(target) {
    return async result => {
        if (currentPage instanceof Node) {
            currentPage.dispatchEvent(disconnectEvent)
            target.innerHTML = ''
        }
        try {
            currentPage = await result
        } catch (err) {
            console.error(err)
            currentPage = renderErrorPage(err)
        }
        target.appendChild(currentPage)
        activateLinks()
    }
}

function activateLinks() {
    const { pathname } = location
    for (const link of Array.from(document.querySelectorAll('a'))) {
        if (link.pathname === pathname) {
            link.setAttribute('aria-current', 'page')
        } else {
            link.removeAttribute('aria-current')
        }
    }
}

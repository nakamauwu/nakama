import { guard } from './auth.js';
import { createRouter } from './lib/router.js';

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
        target.innerHTML = ''
        target.appendChild(await result)
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

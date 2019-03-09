const template = document.createElement('template')
template.innerHTML = `
    <div class="container">
        <h1>404 Not Found</h1>
        <p>Nothing to see here. <a href="/">Go home</a>.</p>
    </div>
`

export default function renderNotFoundPage() {
    return template.content.cloneNode(true)
}

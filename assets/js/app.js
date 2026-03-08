// Dark mode toggle
function toggleDarkMode() {
    const html = document.documentElement;
    if (html.classList.contains('dark')) {
        html.classList.remove('dark');
        localStorage.setItem('theme', 'light');
    } else {
        html.classList.add('dark');
        localStorage.setItem('theme', 'dark');
    }
}

// CSRF token helper for htmx
document.addEventListener('DOMContentLoaded', function () {
    // Set CSRF token header for all htmx requests
    document.body.addEventListener('htmx:configRequest', function (event) {
        const csrfMeta = document.querySelector('meta[name="csrf-token"]');
        if (csrfMeta) {
            event.detail.headers['X-CSRF-Token'] = csrfMeta.content;
        }
    });
});

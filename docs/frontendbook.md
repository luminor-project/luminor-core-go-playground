# Frontend Book

## Template Architecture

### templ Components

Templates are `.templ` files that compile to Go code. They support:

- Type-safe parameters
- Composition via `{ children... }`
- Conditional rendering with `if`/`for`
- Context access for auth state

### Layout System

Two base layouts in `internal/common/web/templates/layouts/`:

- **AppShell** — Sidebar navigation for authenticated pages
- **FullContent** — Full-width with header nav for public/auth pages

Usage:

```go
templ MyPage() {
    @layouts.AppShell("Page Title") {
        <div>Page content here</div>
    }
}
```

## htmx Patterns

htmx handles server-driven interactions without writing JavaScript:

```text
<!-- Form submission -->
<form action="/sign-in" method="post">
    <!-- fields -->
</form>

<!-- Partial page updates (future) -->
<div hx-get="/notifications" hx-trigger="every 30s" hx-swap="innerHTML"></div>
```

### CSRF Note

Server-rendered forms no longer include hidden CSRF token fields. CSRF protection is enforced centrally by Go's `net/http.CrossOriginProtection` middleware for unsafe methods.

## Alpine.js Patterns

Alpine.js handles client-only state:

```html
<!-- Mobile menu toggle -->
<div x-data="{ open: false }">
    <button @click="open = !open">Menu</button>
    <div x-show="open">...</div>
</div>

<!-- Dark mode (via vanilla JS function) -->
<button onclick="toggleDarkMode()">Toggle</button>
```

## Design System

CSS component classes prefixed with `lmn-*` in `assets/css/design-system.css`:

- **Navigation:** `lmn-nav-link`, `lmn-nav-link-active`
- **Typography:** `lmn-h1` through `lmn-h4`, `lmn-text`, `lmn-text-small`
- **Buttons:** `lmn-button-default`, `lmn-button-danger`, `lmn-button-secondary`
- **Forms:** `lmn-form-label`, `lmn-form-input`, `lmn-form-select`
- **Cards:** `lmn-card`, `lmn-card-title`, `lmn-card-content`
- **Tables:** `lmn-table`, `lmn-table-header`, `lmn-table-cell`
- **Badges:** `lmn-badge`, `lmn-badge-success`, `lmn-badge-warning`
- **Alerts:** `lmn-alert`, `lmn-alert-success`, `lmn-alert-danger`
- **Pagination:** `lmn-pagination-nav`, `lmn-pagination-item`
- **Links:** `lmn-link-default`, `lmn-link-danger`

Dark mode supported via Tailwind's `dark:` variants with `class` strategy.

## Asset Pipeline

- CSS: `npm exec tailwindcss -- -i assets/css/app.css -o static/css/app.css`
- JS:
    - `assets/js/app.js` is copied to `static/js/app.js`
    - `htmx` and `Alpine.js` are pinned in `package.json` and copied from `node_modules` to `static/js/` by `mise run prepare-assets`
- Static files served by `http.FileServer`

### Vendor JS Upgrade Workflow

1. Update pinned versions in `package.json` (`htmx.org`, `alpinejs`)
2. Refresh lockfile (`npm install`)
3. Rebuild/copy assets (`mise run prepare-assets`)
4. Verify pages still load `/static/js/htmx.min.js` and `/static/js/alpine.min.js`

## Frontend Quality Gates

- Lint TypeScript/JavaScript:
    - `npm run lint:frontend`
- Check formatting (JS/TS/CSS/JSON/YAML/Markdown):
    - `npm run format:check`
- Auto-fix formatting:
    - `npm run format:write`

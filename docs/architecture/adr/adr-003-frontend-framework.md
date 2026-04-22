# ADR-003: Frontend Framework

## Status: Accepted

## Context

The application needs a responsive web frontend that works on mobile browsers (320px and up) and on desktop. The developer has passing experience with JavaScript/TypeScript and used Angular ~10 years ago (explicitly: do not recommend returning to Angular). The developer's JS/TS experience is described as "passing" — not fluent.

The two candidate frameworks from the PRD are SvelteKit and Next.js (React).

The frontend must:
- Serve a login page (public).
- Serve role-appropriate dashboards (referee vs. assignor).
- Handle forms (profile, match editing, CSV upload).
- Display real-time-ish updates (availability toggle, assignment changes).
- Work on mobile.
- Be deployable as static files or within a Docker container.

Timeline is before August. Frontend work must not dominate the schedule — the developer's strength is on the backend.

## Decision

**SvelteKit** with the **static adapter**.

## Rationale

### Learning curve

Svelte's component model is closer to plain HTML/CSS/JavaScript than React's component model is. A Svelte component is a `.svelte` file that looks like:

```svelte
<script>
  let count = 0;
</script>

<button on:click={() => count++}>Clicked {count} times</button>
```

Reactivity is built into the language (variable assignment triggers re-render). There is no `useState`, `useEffect`, `useCallback`, or mental model of React's rendering lifecycle to learn. For a developer with limited modern JS/TS experience, Svelte's model is significantly easier to pick up than React's.

### Bundle size

Svelte is a compiler, not a runtime library. It compiles components to small, efficient JavaScript with no Svelte runtime in the output. React + ReactDOM adds ~45KB gzipped. This is minor at this scale, but Svelte's output is simpler to reason about.

### SvelteKit's conventions

SvelteKit provides file-based routing (`src/routes/`), server-side `load` functions, and form actions that work naturally with a REST backend. The conventions are simple and well-documented. There is no need to set up React Router, TanStack Query, or other ecosystem packages to achieve basic functionality.

### Static adapter

The static adapter (`@sveltejs/adapter-static`) builds the app as pure HTML/CSS/JS files with no Node.js runtime required at serving time. These files are served by Caddy (the reverse proxy), which is already in the container stack for TLS. This avoids running a Node.js process in production.

### Documentation

Svelte's documentation (svelte.dev, learn.svelte.dev) is among the best in the frontend ecosystem. The interactive tutorial at learn.svelte.dev covers everything needed for this app in a few hours.

### Limitations of this choice

- The ecosystem is smaller than React's. Component libraries for Svelte are less numerous than React. However, for this application's UI needs (forms, tables, a simple assignment panel), Tailwind CSS with base HTML elements is sufficient — no heavy component library is needed.
- Svelte's TypeScript support is good but the TypeScript integration in `.svelte` files has historically been slightly more awkward than in `.tsx` files. Given the developer's limited TS experience, this is less of a concern — basic typing is sufficient.

## Consequences

**Positive:**
- Faster learning curve for a developer with limited React/modern JS experience.
- No Node.js runtime in production (static adapter).
- Smaller, simpler output bundle.
- File-based routing keeps the frontend structure predictable.
- The compiler catches many errors at build time.

**Negative:**
- Smaller community and ecosystem than React. Fewer third-party component libraries. Fewer Stack Overflow answers.
- If a future developer is React-fluent, they'll need to learn Svelte.
- Some SvelteKit idioms (especially `load` functions and form actions) require reading the docs carefully; they differ from both Angular and React patterns.

## Alternatives Considered

**Next.js (React)**: Strong choice for React-experienced developers. The learning curve for React's hooks model (`useState`, `useEffect`, `useRef`, `useMemo`, `useCallback`) and its rendering modes (SSR, SSG, CSR, RSC) is steeper for someone with limited modern JS experience. The React ecosystem is larger but also more fragmented — choosing a state management approach, a data fetching library, and a routing configuration all require decisions that SvelteKit's opinions already make. For a developer hitting a tight deadline, fewer decisions is better.

**Angular**: Explicitly rejected per developer profile. The framework has changed substantially since the developer last used it (~v2 era vs. current standalone component model), and the additional relearning cost is not justified.

**Vanilla HTML + HTMX**: Could serve many of the app's needs (form submissions, simple page updates) with minimal JavaScript. However, the assignment interface (dynamic referee picker, real-time slot updates, conflict indicators) would be awkward to build with HTMX's attribute-based model. SvelteKit's interactivity model is more appropriate for the application's UI complexity.

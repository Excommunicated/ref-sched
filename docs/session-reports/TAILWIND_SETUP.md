# Tailwind CSS Setup Guide for Ref-Sched

## Quick Start

### 1. Install Dependencies
```bash
cd frontend
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

### 2. Configure Tailwind

Create or update `tailwind.config.js`:
```javascript
/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#2563eb',
          hover: '#1d4ed8',
        },
        success: '#16a34a',
        error: '#dc2626',
        warning: '#ea580c',
      },
      maxWidth: {
        '7xl': '1200px',
      },
    },
  },
  plugins: [],
}
```

### 3. Update app.css

Replace contents with:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;

/* Custom component classes */
@layer components {
  .btn {
    @apply inline-block px-4 py-2 font-medium rounded-md transition-all duration-200;
  }
  
  .btn-primary {
    @apply bg-blue-600 text-white hover:bg-blue-700;
  }
  
  .btn-secondary {
    @apply bg-white text-gray-700 border border-gray-300 hover:bg-gray-50;
  }
  
  .card {
    @apply bg-white rounded-lg p-6 shadow;
  }
}
```

### 4. Migration Example

**Before (Vanilla CSS):**
```svelte
<div class="container">
  <div class="header">
    <h1>Dashboard</h1>
    <button class="btn btn-primary">Sign Out</button>
  </div>
</div>

<style>
  .container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem 1rem;
  }
  
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
  }
  
  h1 {
    font-size: 2rem;
    font-weight: 700;
  }
</style>
```

**After (Tailwind):**
```svelte
<div class="max-w-7xl mx-auto px-4 py-8">
  <div class="flex justify-between items-center mb-8">
    <h1 class="text-3xl font-bold">Dashboard</h1>
    <button class="btn btn-primary">Sign Out</button>
  </div>
</div>
```

## Gradual Migration Strategy

### Phase 1: Setup & Utilities
1. Install Tailwind
2. Keep all existing CSS
3. Use Tailwind for spacing, colors only (mx-auto, p-4, text-blue-600)

### Phase 2: New Components
1. Build new components with Tailwind only
2. Don't touch existing components yet

### Phase 3: Systematic Migration
1. Migrate one page per day
2. Start with simpler pages (login, pending)
3. End with complex pages (assignor/matches)

### Phase 4: Cleanup
1. Remove unused CSS
2. Consolidate theme config
3. Run production build to verify size reduction

## Cheat Sheet - Common Conversions

| Vanilla CSS | Tailwind Equivalent |
|-------------|-------------------|
| `display: flex` | `flex` |
| `justify-content: space-between` | `justify-between` |
| `align-items: center` | `items-center` |
| `margin: 0 auto` | `mx-auto` |
| `padding: 1rem` | `p-4` |
| `padding: 0.5rem 1rem` | `px-4 py-2` |
| `background-color: #2563eb` | `bg-blue-600` |
| `color: white` | `text-white` |
| `font-weight: 600` | `font-semibold` |
| `border-radius: 0.5rem` | `rounded-lg` |
| `max-width: 1200px` | `max-w-7xl` |
| `gap: 1rem` | `gap-4` |
| `margin-bottom: 2rem` | `mb-8` |

## Estimated Timeline

- **Setup**: 30 minutes
- **Login page**: 1 hour
- **Dashboard**: 2 hours
- **Referee pages**: 4 hours
- **Assignor pages**: 8 hours
- **Testing & refinement**: 4 hours
- **Total**: ~20 hours

## Benefits for This Project

1. **Responsive design**: Built-in breakpoints (sm, md, lg, xl)
2. **Dark mode**: Easy to add later with `dark:` prefix
3. **Consistency**: No more wondering about spacing values
4. **Smaller bundle**: Tree-shaking removes unused styles
5. **Faster development**: No context switching between files

## When NOT to Use Tailwind

- ❌ Very custom, artistic designs
- ❌ Team unfamiliar with utility-first CSS
- ❌ Tight deadline on current project
- ❌ Project is almost finished

## Recommendation for Ref-Sched

**Current Phase**: MVP/Beta
**Recommendation**: **Wait until v2.0 or major refactor**

**Reasons**:
- Current CSS is clean and maintainable
- Would slow down feature development
- No immediate benefit justifies 20+ hour migration
- Team would need to learn new syntax

**Alternative**: Use Tailwind's principles (spacing scale, color palette) in your current CSS to stay organized.

## If You Decide to Proceed

1. Create a new branch: `git checkout -b feature/tailwind-migration`
2. Install dependencies
3. Migrate one page as proof of concept
4. Review with team
5. If approved, continue migration
6. Thoroughly test before merging

## Resources

- [Tailwind Docs](https://tailwindcss.com/docs)
- [Tailwind + SvelteKit](https://tailwindcss.com/docs/guides/sveltekit)
- [Tailwind Play](https://play.tailwindcss.com/) - Online playground
- [Tailwind Cheat Sheet](https://nerdcave.com/tailwind-cheat-sheet)

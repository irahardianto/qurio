# Implementation Plan - Frontend Design Refresh (MVP)

**Feature:** Frontend Design Refresh - "The Sage" Aesthetic
**Date:** 2025-12-28
**Sequence:** 1
**Status:** Planned

## 1. Feature Overview

**Goal:** Transform the current frontend into a "Sage" archetype interface: technical, precise, and grounded. Implement the "Void Black" and "Cognitive Blue" brand identity, ensuring a high-fidelity, developer-native look and feel.

**Scope:**
- Global Design System (Colors, Typography, Reset).
- Layout Architecture (Sidebar, Main Content).
- Core UI Components (Buttons, Inputs, Cards).
- Feature Views (Jobs Monitor, Source Library).
- Micro-interactions & Transitions.

**Out of Scope:**
- changing business logic or backend APIs.
- Adding new features.

**Gap Analysis:**
- **Nouns:**
  - `Color Palette`: Mismatched. Needs unification to `#0F172A` (bg) and `#3B82F6` (primary).
  - `Typography`: Needs `Inter` (UI) and `JetBrains Mono` (Data).
  - `Layout`: Needs "sharp lines" and "geometric" structure.
- **Verbs:**
  - `Navigate`: Sidebar needs active state polish.
  - `View Data`: Tables/Lists need "technical data" styling.
  - `Interact`: Buttons/Inputs need precise feedback (hover/focus glow).

## 2. Implementation Tasks

### Task 1: Foundation & Design System

**Files:**
- Modify: `apps/frontend/tailwind.config.js`
- Modify: `apps/frontend/src/style.css`
- Modify: `apps/frontend/index.html` (for fonts)

**Requirements:**
- **Acceptance Criteria**
  1. `style.css` CSS variables are updated to match Brand Colors (converted to HSL).
     - `--background` -> Void Black (`222.2 47.4% 11.2%` for #0F172A)
     - `--primary` -> Cognitive Blue (`217.2 91.2% 59.8%` for #3B82F6)
     - `--secondary` -> Grounded Greenish/Gray (`149.3 80% 39%` or similar for accent)
  2. Tailwind config extends `fontFamily` with `sans` (Inter) and `mono` (JetBrains Mono).
  3. Inter and JetBrains Mono fonts are loaded (via Google Fonts CDN).

- **Functional Requirements**
  1. Application renders with the new dark background and blue primary buttons by default (inherited by shadcn components).

- **Non-Functional Requirements**
  - None for this task.

- **Test Coverage**
  - Manual verification: Open app, inspect `<body>` and `<button>` computed styles to verify HSL values.

**Step 1: Add Fonts to index.html**
```html
<!-- apps/frontend/index.html -->
<head>
  <!-- ... existing tags -->
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;700&family=JetBrains+Mono:wght@400;700&display=swap" rel="stylesheet">
</head>
```

**Step 2: Update Global Styles (Theming shadcn)**
```css
/* apps/frontend/src/style.css */
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  /* Default to Dark Mode (The Sage) aesthetics even in root if desired, or strictly manage via .dark */
  :root {
    /* "Sage" Light Mode (Optional - inverse of dark) */
    --background: 0 0% 100%;
    --foreground: 222.2 47.4% 11.2%;
    /* ... other light vars ... */
    --radius: 0.5rem;
  }
 
  .dark {
    /* Brand: Void Black #0F172A -> hsl(222.2, 47.4%, 11.2%) */
    --background: 222.2 47.4% 11.2%;
    --foreground: 210 40% 98%;

    /* Surface/Card: Slightly lighter than void #1E293B -> hsl(215, 25%, 27%) */
    --card: 217.2 32.6% 17.5%;
    --card-foreground: 210 40% 98%;
 
    --popover: 222.2 47.4% 11.2%;
    --popover-foreground: 210 40% 98%;
 
    /* Primary: Cognitive Blue #3B82F6 -> hsl(217.2, 91.2%, 59.8%) */
    --primary: 217.2 91.2% 59.8%;
    --primary-foreground: 222.2 47.4% 11.2%;
 
    /* Secondary: Context Gray #64748B -> hsl(215, 16%, 47%) */
    --secondary: 217.2 32.6% 17.5%;
    --secondary-foreground: 210 40% 98%;
 
    --muted: 217.2 32.6% 17.5%;
    --muted-foreground: 215 20.2% 65.1%;
 
    --accent: 217.2 32.6% 17.5%;
    --accent-foreground: 210 40% 98%;
 
    /* Destructive/Error */
    --destructive: 0 62.8% 30.6%;
    --destructive-foreground: 210 40% 98%;
 
    /* Borders: #334155 -> hsl(215, 25%, 27%) */
    --border: 217.2 32.6% 17.5%;
    --input: 217.2 32.6% 17.5%;
    --ring: 212.7 26.8% 83.9%;
  }
}

@layer base {
  * {
    @apply border-border;
  }
  body {
    @apply bg-background text-foreground font-sans antialiased selection:bg-primary selection:text-primary-foreground;
  }
}
```

**Step 3: Update Tailwind Config**
```javascript
// apps/frontend/tailwind.config.js
/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class', // Ensure class strategy is enabled
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'monospace'],
      },
      // Keep shadcn default extends (colors mapped to CSS vars)
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
      },
      // ... keep animations/keyframes
    },
  },
  plugins: [],
}
```

**Step 4: Verify**
- Run `npm run dev` in `apps/frontend`.
- Check browser: Background is `#0F172A` (Void Black), Buttons are `#3B82F6` (Cognitive Blue).

---

### Task 2: Layout Architecture (Sidebar & Shell)

**Files:**
- Modify: `apps/frontend/src/components/layout/Sidebar.vue`
- Modify: `apps/frontend/src/components/layout/AppLayout.vue`

**Requirements:**
- **Acceptance Criteria**
  1. Sidebar has a distinct but subtle border (right).
  2. Sidebar background matches or is slightly offset from body.
  3. Navigation links use `muted-foreground` (inactive) and `primary` (active/hover) with a "glow" or marker.
  4. AppLayout provides a consistent container with "sharp" aesthetics.

- **Functional Requirements**
  1. Navigation remains functional.

- **Non-Functional Requirements**
  - Responsive design (sidebar collapses or works on mobile - preserve existing behavior but style it).

- **Test Coverage**
  - Visual verification of layout structure.

**Step 1: Style AppLayout**
```vue
<!-- apps/frontend/src/components/layout/AppLayout.vue -->
<template>
  <div class="flex h-screen w-full bg-background overflow-hidden">
    <Sidebar />
    <main class="flex-1 flex flex-col min-w-0 overflow-hidden relative">
      <!-- Optional: Grid overlay for "technical" texture -->
      <div class="absolute inset-0 bg-[url('/grid-pattern.svg')] opacity-[0.02] pointer-events-none"></div>
      
      <div class="flex-1 overflow-y-auto p-6 md:p-8 scroll-smooth">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
```

**Step 2: Style Sidebar**
```vue
<!-- apps/frontend/src/components/layout/Sidebar.vue -->
<template>
  <aside class="w-64 flex-shrink-0 border-r border-border bg-card/30 backdrop-blur-sm flex flex-col">
    <div class="h-16 flex items-center px-6 border-b border-border">
       <!-- Logo Area -->
       <span class="font-mono font-bold text-xl tracking-tight text-foreground">
         <span class="text-primary">&lt;</span>Qurio<span class="text-primary">/&gt;</span>
       </span>
    </div>

    <nav class="flex-1 px-4 py-6 space-y-1">
      <router-link 
        v-for="item in navigation" 
        :key="item.name" 
        :to="item.href"
        class="group flex items-center px-3 py-2 text-sm font-medium rounded-md transition-all duration-200"
        :class="[
          $route.path === item.href 
            ? 'bg-primary/10 text-primary shadow-[0_0_10px_rgba(59,130,246,0.15)] border-l-2 border-primary' 
            : 'text-muted-foreground hover:bg-secondary/50 hover:text-foreground border-l-2 border-transparent'
        ]"
      >
        <component :is="item.icon" class="mr-3 h-5 w-5 flex-shrink-0" />
        {{ item.name }}
      </router-link>
    </nav>
    
    <!-- Optional: System Status or Version -->
    <div class="p-4 border-t border-border">
      <div class="flex items-center gap-2">
        <div class="h-2 w-2 rounded-full bg-emerald-500 animate-pulse"></div>
        <span class="text-xs font-mono text-muted-foreground">System Online</span>
      </div>
    </div>
  </aside>
</template>
```

**Step 3: Verify**
- Check sidebar styles: Dark, sharp borders, blue glow on active items.
- Check page transition: Smooth fade between routes.

---

### Task 3: Aesthetic Enhancements (Glows & Typography)

**Files:**
- Modify: `apps/frontend/src/components/ui/button/Button.vue`
- Modify: `apps/frontend/src/components/ui/input/Input.vue`

**Requirements:**
- **Acceptance Criteria**
  1. **Button:** Primary variant gets a subtle glow on hover (`shadow-[0_0_15px_rgba(59,130,246,0.5)]`).
  2. **Input:** Uses `font-mono` for code/data precision. Focus ring color uses `ring-primary`.

- **Functional Requirements**
  - Component functionality (clicks, input) remains unchanged.

**Step 1: Enhance Button Styles**
```vue
<!-- Modify Button cva classes to add the unique glow effect for 'default' (primary) variant -->
<!-- Note: In shadcn-vue, this is usually in the variants object or class string -->
variant: {
  default: "bg-primary text-primary-foreground hover:bg-primary/90 hover:shadow-[0_0_15px_rgba(59,130,246,0.4)]",
  // ... other variants
}
```

**Step 2: Enhance Input Styles**
```vue
<!-- Modify Input class string -->
class="... font-mono focus-visible:ring-primary ..."
```

**Step 3: Verify**
- Button: Hover over a primary button -> see glow.
- Input: Type text -> see monospaced font.

---

### Task 4: Feature Polish (Jobs Monitor)

**Files:**
- Modify: `apps/frontend/src/views/JobsView.vue` (or features/job/components/...)

**Requirements:**
- **Acceptance Criteria**
  1. Job list looks like a system log/monitor.
  2. Statuses (Completed, Failed) use clear colors (Green, Red) and mono font.
  3. Layout uses a table or grid with `border-slate-800` separators.

- **Functional Requirements**
  - Data display remains accurate.

**Step 1: Update Job List Styling**
- Use `font-mono` for Job IDs and timestamps.
- Use `Badge` component for status (Green for success, Red for failure).
- Add specific table styles:
  ```html
  <table class="w-full text-sm text-left">
    <thead class="text-xs text-slate-400 uppercase bg-slate-900/50 border-b border-slate-800">
      <!-- headers -->
    </thead>
    <tbody class="divide-y divide-slate-800">
      <!-- rows -->
    </tbody>
  </table>
  ```

**Step 2: Verify**
- Navigate to `/jobs`.
- Verify the "System Monitor" aesthetic.

---

### Task 5: Feature Polish (Sources Library)

**Files:**
- Modify: `apps/frontend/src/views/SourcesView.vue` (or similar)
- Modify: `apps/frontend/src/features/source/components/SourceList.vue`

**Requirements:**
- **Acceptance Criteria**
  1. Source cards/list items use the "Card" component style.
  2. "Add Source" button uses the new "Primary Button" style with glow.
  3. Icons (Lucide) are used to distinguish source types (PDF, Web, etc.).

**Step 1: Update Source List**
- Convert list items to use the new `Card` aesthetic or a clean list with `border-slate-800`.
- Ensure "Type" indicators (e.g., PDF icon) are prominent (`text-brand-blue`).

**Step 2: Verify**
- Navigate to `/sources` (or home).
- Verify "Library" aesthetic.


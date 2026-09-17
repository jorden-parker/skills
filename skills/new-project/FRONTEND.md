# Frontend setup

The UI branch of [`new-project`](SKILL.md). Three pieces that only work well together: **shadcn/ui** supplies the components, **Geist** supplies the values they render with, and **`@shadcn/lint`** stops the two from drifting apart.

For a new frontend, follow this setup before writing components. For an existing frontend, inspect its components, styling, tokens, and lint configuration first. Reuse existing shadcn/ui and Geist wiring, merge missing compatible configuration, and follow the project’s paths and package manager. Preserve an established design system; adopting this stack where it would replace that system requires a separate migration decision. Run `shadcn init` only when shadcn is absent and compatible with the existing stack.

The examples below assume a new Next.js frontend. Adapt framework-specific steps to the actual project; use `next/font` only in Next.js.

Requires Node 20.19+ and ESLint 9.30+. Check both before starting - `create-next-app` will happily scaffold on an older Node and fail at the lint step.

## 1. shadcn/ui

```sh
pnpm dlx shadcn@latest init
```

Take the defaults. It writes `components.json`, the CSS variable blocks in `app/globals.css`, and a `lib/utils.ts` with `cn`.

Add components as they are needed, never up front:

```sh
pnpm dlx shadcn@latest add button card input
```

These land in `components/ui/` and belong to the project. Editing one is normal and expected; that is the whole point of shadcn/ui.

Add the Tailwind plugin to the Prettier setup from `SKILL.md`, so class order is decided by the formatter instead of by argument:

```sh
pnpm add -D prettier-plugin-tailwindcss
```

```json
{ "plugins": ["prettier-plugin-tailwindcss"] }
```

It sorts `className` strings and the class arguments inside `cn` and `cva`, which keeps diffs on a restyled component down to the classes that actually changed.

## 2. Geist

Invoke the `geist` skill for the tokens, the type scale, and the step-number rules. Copy `geist-tokens.css` and `geist-type.css` into `app/`, and import them at the top of `app/globals.css`.

Then point shadcn's variables at Geist tokens. `init` wrote a `:root` block of raw colour values - replace those values with `var(--ds-*)`:

```css
:root {
  --background: var(--ds-background-100);
  --foreground: var(--ds-gray-1000);
  --card: var(--ds-background-200);
  --card-foreground: var(--ds-gray-1000);
  --popover: var(--ds-background-200);
  --popover-foreground: var(--ds-gray-1000);
  --primary: var(--ds-gray-1000);
  --primary-foreground: var(--ds-background-100);
  --secondary: var(--ds-gray-100);
  --secondary-foreground: var(--ds-gray-1000);
  --muted: var(--ds-gray-100);
  --muted-foreground: var(--ds-gray-900);
  --accent: var(--ds-gray-200);
  --accent-foreground: var(--ds-gray-1000);
  --destructive: var(--ds-red-700);
  --border: var(--ds-gray-400);
  --input: var(--ds-gray-400);
  --ring: var(--ds-blue-700);
  --radius: var(--ds-radius-base);
}
```

**One block covers both themes.** The Geist tokens swap underneath on `data-theme` and on the system preference, so the mapping resolves correctly in dark mode with no second block. Delete the `.dark` block `init` generated - a duplicated mapping there is the thing that goes stale.

Keep the `@theme inline` block exactly as `init` wrote it. It turns these variables into Tailwind utilities, and every shadcn component depends on those names.

Align the dark variant with how Geist switches, so `dark:` utilities still fire:

```css
@custom-variant dark (&:is([data-theme="dark"] *));
```

Set the fonts through `next/font` from the `geist` package rather than a stylesheet link - it self-hosts and skips the network round-trip:

```sh
pnpm add geist
```

## 3. @shadcn/lint

Read <https://github.com/shadcn-ui/lint/blob/main/SETUP.md> and follow it. For existing frontends, follow its adoption scope guidance and merge rules with the existing lint configuration. For new projects, two things change what it tells you:

- **Its scope rule is written for existing repositories**, where enabling rules would flood the user with findings. Here there is no code yet, so turn the rules on now. Starting strict costs nothing; retrofitting them costs a cleanup pass.
- **It picks Oxlint when no linter exists.** `create-next-app --eslint` already wrote one, so take the ESLint path and register the plugin into the config that is already there - the `eslint.config.ts` renamed from `.mjs` in `SKILL.md`.

```sh
pnpm add -D @shadcn/lint
```

In `eslint.config.ts`, keep Next's existing entries and append:

```ts
import { plugin as shadcn } from "@shadcn/lint";

// ...append to the exported array:
{
  files: ["**/*.{js,jsx,ts,tsx}"],
  plugins: { shadcn },
  rules: {
    "shadcn/no-restyle": ["error", { allow: ["layout"] }],
    "shadcn/no-raw-colors": "error",
    "shadcn/no-arbitrary-values": "error",
    "shadcn/no-inline-styles": "error",
    "shadcn/no-unknown-classes": "error",
    "shadcn/require-static-classes": "error",
  },
},
{
  files: ["components/ui/**"],
  rules: { "shadcn/no-restyle": "off" },
},
```

Next's config already parses TSX, so leave its parser setup alone.

The `components/ui/**` override matters: those files are the design system, so they are allowed to style themselves. Everything outside them consumes the system instead.

`components.json` gives the plugin automatic component and theme discovery, so no `settings.shadcn` block is needed in a single-package project. A monorepo needs one - `SETUP.md` covers the shape.

## 4. Hand it to the next agent

Add one more line to the `AGENTS.md` written in `SKILL.md`:

```md
- UI and styling use Vercel's Geist Design System through the shadcn/ui variables in `app/globals.css`. Invoke the `geist` skill before choosing a colour, radius, or type size.
```

## Green for a frontend

On top of the build, lint, and dev checks in `SKILL.md`, confirm the wiring actually took:

- A `<Button>` on the page renders with Geist's radius and grey, not shadcn's defaults.
- Toggling `data-theme="dark"` on `<html>` flips the whole page, components included.
- `pnpm lint` reports zero `shadcn/*` findings.

A green build with an unwired theme is the failure worth catching here - nothing errors, the colours are just quietly wrong.

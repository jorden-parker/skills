---
name: geist
description: Vercel's Geist Design System - real design tokens, type scale, and component recipes. Use whenever you build or restyle a UI, write an HTML artifact, pick colours or fonts, build a landing page or dashboard, or are about to invent an ad-hoc palette.
---

# Geist Design System

Use Geist as the default visual language. It gives you a decided-upon palette, type scale, and radii, so no design work is spent re-deriving greys and blues.

The token values in this skill were extracted from the computed styles of the live `vercel.com/geist` stylesheets, both themes. Do not edit them by hand and do not substitute approximations.

## When this applies

**Always** for standalone HTML artifacts, one-off pages, prototypes, and demos.

**For a project repo**, first check whether a design system is already present:

```sh
ls tailwind.config.* 2>/dev/null
grep -rl "@theme\|--color-\|design-tokens" --include="*.css" . 2>/dev/null | head
ls src/components/ui components/ui 2>/dev/null
```

- **Nothing found** - use Geist.
- **Something found** - that system wins. Do not introduce Geist alongside it. Say what you found and move on.

Never migrate an existing project onto Geist unless explicitly asked.

## Using the tokens

Copy [`geist-tokens.css`](geist-tokens.css) and [`geist-type.css`](geist-type.css) into the project, or inline them into a single-file artifact.

Load the fonts:

```html
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link
  href="https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600&family=Geist+Mono:wght@400;500&display=swap"
  rel="stylesheet"
/>
```

In a Next.js or bundled project, prefer the package - it self-hosts and avoids the network round-trip:

```sh
npm i geist
```

## The scales

Every hue runs `100` to `1000`, and the step number means the same thing in every scale. This is the part to internalise - it removes almost all colour guesswork:

| Steps           | Role                                                   |
| --------------- | ------------------------------------------------------ |
| 100 / 200 / 300 | Background: default, hover, active                     |
| 400 / 500 / 600 | Border: default, hover, active                         |
| 700 / 800       | High-contrast background (a solid fill), and its hover |
| 900             | Secondary text and icons                               |
| 1000            | Primary text and icons                                 |

Hues: `gray`, `blue`, `red`, `amber`, `green`, `teal`, `purple`, `pink`. Plus `--ds-gray-alpha-*` for overlays on unknown backgrounds, and `--ds-background-100` / `--ds-background-200` for page and raised surfaces.

Semantic mapping: blue = informational and focus, red = error and destructive, amber = warning, green = success.

### Rules that keep it looking right

- **Text on a surface is `900` or `1000`.** Never use `400`-`600` for text - those are border steps. `gray-600` on the page background is 2.37:1 in light mode, well under the 4.5:1 AA floor. It happens to pass in dark mode; use `900` in both anyway, so a theme switch cannot break you.
- **A solid coloured button is `700` background with white text**, hovering to `800`.
- **A subtle status chip is `100` background, `400` border, `900` text** of the same hue.
- **Borders are `--ds-gray-400`**, not a hand-rolled grey.
- **Never hardcode a hex.** If a value you need is missing, pick the nearest step rather than inventing one.

## Radii, shadows, sizes

- Radius: `--ds-radius-base` (6px) for controls and inputs, `--ds-radius-medium` (12px) for cards, menus, and modals, `--ds-radius-fullscreen` (16px) for full-bleed panels.
- Elevation: `--ds-shadow-border` for a flat bordered surface, then `--ds-shadow-small` / `--ds-shadow-medium` / `--ds-shadow-large`, and `--ds-shadow-menu` / `--ds-shadow-modal` for floating layers.
- Control heights: `--ds-size-small` 32px, `--ds-size-medium` 36px, `--ds-size-large` 40px.
- Focus: `box-shadow: var(--ds-focus-ring)` or `outline: var(--ds-focus-ring-outline)`. Never remove focus styling without replacing it.

## Typography

Use the type classes from `geist-type.css` rather than raw font sizes: `.text-heading-32`, `.text-copy-16`, `.text-label-13`, `.text-button-14`.

- Headings carry negative letter-spacing that scales with size. It is what makes Geist look like Geist, so do not flatten it.
- Body prose is `copy`. Dense UI text is `label`. They differ in line-height, not size.
- `--ds-font-mono` for code, IDs, hashes, and numeric tables.

## Component recipes

See [`RECIPES.md`](RECIPES.md) for button, input, card, badge, table, and code-block CSS built only from these tokens. Copy and adapt - they are a starting point, not a library.

### Verified

Every pairing above was checked against WCAG AA in both themes: `gray-1000` and `gray-900` on both backgrounds, and the `900`-on-`100` badge pattern for all seven hues. The lowest was teal at 4.63:1; all 22 pass. If you invent a pairing outside these rules, check it yourself.

## Dark mode

`geist-tokens.css` handles it. The page follows the system preference; setting `data-theme="dark"` or `data-theme="light"` on `<html>` overrides it in either direction.

Always give `body` an explicit `background: var(--ds-background-100)` and `color: var(--ds-gray-1000)`. A transparent body inherits the host's theme and breaks in one of the two modes.

## Making it stick in a project

The first time you use Geist in a project, offer to record it so future sessions do not have to rediscover it. Ask, then add one line to the project's `CLAUDE.md` or `AGENTS.md`:

```md
- UI and styling use Vercel's Geist Design System. Invoke the `geist` skill before writing any CSS or choosing colours.
```

Only the project's own file. Never write to a global or user-level config.

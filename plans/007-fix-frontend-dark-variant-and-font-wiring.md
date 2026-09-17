# Plan 007: Make `new-project/FRONTEND.md` produce a theme that actually flips, and stop telling agents to use `npm`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 067367d..HEAD -- skills/new-project/FRONTEND.md skills/geist/SKILL.md skills/geist/geist-tokens.css`
> If any of these changed since this plan was written, compare the "Current
> state" excerpts against the live files before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW (documentation only, but it is documentation agents execute)
- **Depends on**: none
- **Category**: docs
- **Planned at**: commit `067367d`, 2026-09-15

## Why this matters

These skills are instructions an agent follows literally, so a wrong line is a
wrong project. Three lines are wrong today:

1. `FRONTEND.md` tells the agent to define Tailwind's `dark:` variant as
   `[data-theme="dark"] *`. But the Geist tokens it just imported also flip to
   dark under the operating system's `prefers-color-scheme: dark` when no
   `data-theme` attribute is set. Result on a system-dark machine with no
   attribute: every token is dark, every `dark:` utility is inert. That is
   precisely the failure `FRONTEND.md` itself warns about in its last line —
   "nothing errors, the colours are just quietly wrong."
2. `FRONTEND.md` says to load fonts through `next/font` from the `geist`
   package, then stops at `pnpm add geist`. The tokens read the font from
   `--ds-font-sans`, which the package does not set. Without wiring, the page
   silently falls back to the system font stack.
3. `skills/geist/SKILL.md` tells the agent `npm i geist`. The `new-project`
   skill in the same repo says "Never `npm`", and a `package-lock.json` is
   the first thing it would create.

After this plan the theme attribute is always present (so the simple variant
is correct), the font is wired, and the package-manager instruction matches
the rest of the repo.

## Current state

- `skills/new-project/FRONTEND.md` — the UI branch of the `new-project`
  skill. Excerpts:

```md
<!-- FRONTEND.md:67 -->

**One block covers both themes.** The Geist tokens swap underneath on `data-theme` and on the system preference, so the mapping resolves correctly in dark mode with no second block. Delete the `.dark` block `init` generated - a duplicated mapping there is the thing that goes stale.
```

````md
<!-- FRONTEND.md:71-75 -->

Align the dark variant with how Geist switches, so `dark:` utilities still fire:

```css
@custom-variant dark (&:is([data-theme="dark"] *));
```
````

````

```md
<!-- FRONTEND.md:77-81 -->
Set the fonts through `next/font` from the `geist` package rather than a stylesheet link - it self-hosts and skips the network round-trip:

```sh
pnpm add geist
````

````

```md
<!-- FRONTEND.md:132-140 -->
## Green for a frontend
...
- Toggling `data-theme="dark"` on `<html>` flips the whole page, components included.
...
A green build with an unwired theme is the failure worth catching here - nothing errors, the colours are just quietly wrong.
````

- `skills/geist/geist-tokens.css` — how the tokens switch:

```css
/* geist-tokens.css:7-8 (header comment) */
 *   prefers-color-scheme: dark             -> dark, unless [data-theme="light"]
 *   [data-theme="dark"] / [data-theme="light"] -> explicit override, wins both ways
```

```css
/* geist-tokens.css:184-185 and :309 */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
...
:root[data-theme="dark"] {
```

```css
/* geist-tokens.css:15-18 */
--ds-font-sans: ... (a font stack beginning with "Geist") --ds-font-mono: ...
  (a font stack beginning with "Geist Mono");
```

Open the file and read the exact values of lines 15–18 before Step 2; do not
change them.

- `skills/geist/SKILL.md:63-67`:

````md
In a Next.js or bundled project, prefer the package - it self-hosts and avoids the network round-trip:

```sh
npm i geist
```
````

`````

- `skills/new-project/SKILL.md:61`: "`pnpm`, except on a project that runs on
  Bun … Never `npm`; a `package-lock.json` in a new project is a mistake to
  correct, not a default to accept." — the rule the geist line violates.
- The `geist` npm package exports `GeistSans` and `GeistMono` from
  `geist/font/sans` and `geist/font/mono`; each exposes `.variable` (a class
  that defines a CSS variable, `--font-geist-sans` / `--font-geist-mono`) and
  `.className`. This is the package's documented API as of the 1.x line.
- `next-themes` is the standard way to manage a theme attribute in Next.js
  App Router. With `attribute="data-theme"` and `enableSystem`, it resolves
  the OS preference into an explicit `data-theme="light"`/`"dark"` on
  `<html>` before paint, so the attribute is always present.
- Vocabulary (`CONTEXT.md`): **Token** (a named design value from the Geist
  stylesheets), **Design System Present** (the test that decides whether
  Geist applies). Use "token" not "variable" in prose you add.
- Style: these files are prose with fenced code blocks, formatted by
  Prettier (`printWidth: 80` does not wrap Markdown prose; Prettier leaves
  paragraphs alone but normalises lists and code fences).

## Commands you will need

| Purpose | Command                          | Expected on success                         |
| ------- | -------------------------------- | ------------------------------------------- |
| Format  | `pnpm exec prettier --check skills/` | `All matched files use Prettier code style!` |
| Grep    | see Done criteria                |                                             |

There is no way to execute these instructions inside this repo; Step 4 is a
manual scaffold check and is optional but recommended.

## Scope

**In scope**:

- `skills/new-project/FRONTEND.md` — sections 2 ("Geist") and "Green for a frontend"
- `skills/geist/SKILL.md` — the one `npm i geist` line

**Out of scope**:

- `skills/geist/geist-tokens.css`, `geist-type.css`, `RECIPES.md` — the
  tokens are extracted from live stylesheets and must not be hand-edited
  (`skills/geist/SKILL.md:9-10`).
- `skills/new-project/SKILL.md` — nothing there is wrong.
- The shadcn variable mapping block in `FRONTEND.md:43-65` — correct as is.

## Git workflow

- Branch: `docs/frontend-theme-wiring`
- Conventional Commits (hook-enforced). Suggested single commit:
  `docs: wire the theme attribute and Geist fonts in the frontend setup`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Replace the dark-variant paragraph in `FRONTEND.md`

Replace lines 71–75 (from "Align the dark variant" through the closing code
fence) with:

````md
The tokens also switch on the operating system's `prefers-color-scheme` when
`<html>` carries no `data-theme` attribute, and Tailwind's `dark:` variant
cannot see that media query and the attribute at the same time. Give the
attribute a single owner so it is always present: `next-themes` resolves the
system preference into an explicit `data-theme` before first paint.

```sh
pnpm add next-themes
`````

In `app/layout.tsx`, wrap the body and add `suppressHydrationWarning` to
`<html>` (the attribute is set on the client):

```tsx
import { ThemeProvider } from "next-themes";

<html lang="en" suppressHydrationWarning>
  <body>
    <ThemeProvider attribute="data-theme" defaultTheme="system" enableSystem>
      {children}
    </ThemeProvider>
  </body>
</html>;
```

Then align the dark variant with the attribute, so `dark:` utilities fire
whenever the tokens are dark:

```css
@custom-variant dark (&:is([data-theme="dark"] *));
```

`````

**Verify**: `grep -n 'next-themes' skills/new-project/FRONTEND.md` → at
least 2 lines. `grep -c '@custom-variant dark' skills/new-project/FRONTEND.md` → `1`.

### Step 2: Complete the font wiring in `FRONTEND.md`

Replace lines 77–81 (the `next/font` paragraph and its `pnpm add geist`
block) with:

````md
Set the fonts through `next/font` from the `geist` package rather than a
stylesheet link - it self-hosts and skips the network round-trip:

```sh
pnpm add geist
```

The tokens read the typeface from `--ds-font-sans` and `--ds-font-mono`,
and the package defines `--font-geist-sans` and `--font-geist-mono` on
whichever element carries its `.variable` classes. Put the classes on
`<html>` and point the tokens at them, in `app/globals.css` after the token
imports:

```tsx
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";

<html lang="en" suppressHydrationWarning className={`${GeistSans.variable} ${GeistMono.variable}`}>
```

```css
:root {
  --ds-font-sans: var(--font-geist-sans), var(--ds-font-sans);
  --ds-font-mono: var(--font-geist-mono), var(--ds-font-mono);
}
```

Remove the Google Fonts `<link>` if the scaffolder or the `geist` skill added
one; the package replaces it.
`````

Note the `<html>` line now appears in both Step 1 and Step 2. Merge them into
one `<html>` example if you prefer, but both `suppressHydrationWarning` and
the `className` must survive.

**Verify**: `grep -n 'font-geist-sans' skills/new-project/FRONTEND.md` → at
least 2 lines.

### Step 3: Extend "Green for a frontend"

In the bullet list under `## Green for a frontend`, add after the
`data-theme` bullet:

```md
- With no `data-theme` set by hand, switching the operating system to dark mode flips the page too, and `dark:` utilities apply. If tokens go dark but a `dark:` class does not, the `ThemeProvider` is not mounted.
- The rendered font is Geist, not the system sans. Check computed `font-family` on `<body>` in devtools.
```

**Verify**: `pnpm exec prettier --check skills/new-project/FRONTEND.md` → passes.

### Step 4: Fix the package manager line in `skills/geist/SKILL.md`

Change `npm i geist` to `pnpm add geist`.

**Verify**: `grep -rn 'npm i ' skills/` → no output.
`grep -rn 'npm install\|npm ci' skills/` → no output (the `new-project`
skill mentions `npm` only to forbid it, in the phrase "Never `npm`").

### Step 5 (optional but recommended): Scaffold once and check

If `pnpm`, Node 20.19+, and network access are available, in a temp
directory outside this repo, follow `FRONTEND.md` sections 1–2 on a fresh
`create-next-app` scaffold and confirm the three "Green for a frontend"
theme/font bullets hold. Do not commit the scaffold anywhere. If you skip
this step, say so in your report.

## Test plan

No automated tests: these are Markdown instructions. Verification is the
grep checks above plus Prettier. Step 5 is the behavioural check.

## Done criteria

- [ ] `grep -c 'next-themes' skills/new-project/FRONTEND.md` ≥ 2
- [ ] `grep -c 'font-geist-sans' skills/new-project/FRONTEND.md` ≥ 2
- [ ] `grep -rn 'npm i \|npm install\|npm ci' skills/` prints nothing
- [ ] `pnpm exec prettier --check skills/` passes
- [ ] `git status` shows only `skills/new-project/FRONTEND.md` and `skills/geist/SKILL.md` modified
- [ ] `plans/README.md` status row for 007 updated

## STOP conditions

Stop and report back (do not improvise) if:

- `FRONTEND.md` lines 67–81 or `skills/geist/SKILL.md:63-67` do not match
  the excerpts.
- `geist-tokens.css:15-18` do not define `--ds-font-sans` and
  `--ds-font-mono` (the CSS in Step 2 depends on those names).
- Step 5 shows `dark:` utilities still not firing with the `ThemeProvider`
  mounted — report; do not switch to a media-query variant on your own.

## Maintenance notes

- If Geist tokens ever stop switching on `prefers-color-scheme` (a change to
  `geist-tokens.css` from a re-extraction), the `next-themes` step becomes
  optional rather than wrong; leave it.
- The `geist` package's export paths (`geist/font/sans`) are its 1.x API. A
  reviewer of a future bump should re-check them against the package README.

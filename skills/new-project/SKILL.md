---
name: new-project
description: Scaffold a new project from zero - brainstorm the name, pick the framework, create the directory, and get it building and linting clean. Use when starting a new app, CLI, library, or service, or when the user has an idea but no repository yet.
argument-hint: "optional: the idea, in a sentence"
---

# New project

Take an idea to a repository that is **green**: it runs, it builds, it lints clean, and its first commit is in.

Work the steps in order. Each one ends on a user decision or a command that exits zero - do not carry an open question into the next step.

## 1. Pin the shape

The **shape** is the kind of thing being built. It decides the framework, the layout, and half the naming vocabulary, so settle it first.

Ask whatever you cannot infer from the user's sentence, then write a brief of at most one paragraph covering:

- Shape: web app, CLI, library, service, or script.
- Who runs it, and where.
- The one thing it must do to be worth existing.
- Anything already decided: a language, a host, an existing repo it must sit beside.

**Done when** the user has confirmed that paragraph. Their "yes" is the gate; an assumed shape is a rewrite later.

## 2. Name it

Names come from the brief, not from thin air.

1. Pull the concrete words out of the brief: the domain objects, the verb it performs, the metaphor hiding in it.
2. Generate across four registers, so the list is not five variations of one idea:
   - **Literal compound** - `linkwarden`, `passkit`
   - **Metaphor** - `harbor`, `anvil`, `lantern`
   - **Root word** - Latin or Greek for the verb - `iterum`, `pharos`
   - **Short invented** - pronounceable, 5-7 letters, no digits or hyphens
3. Screen the survivors:

```sh
pnpm view <name> version 2>&1 | head -1   # "404" means the registry name is free
gh search repos "<name>" --limit 5       # collisions worth knowing about
```

4. Present exactly five, each with a one-line rationale and its screening result. Say which one you would pick and why.

**Done when** the user picks one. Stop and wait for that answer - naming is theirs, and the directory name, the package name, and the module path all inherit from it.

## 3. Pick the framework

Take the default for the shape unless the brief names a reason to move off it. State the default and the reason in one line, then go.

| Shape           | Default                                                                     | Move off it when                                                    |
| --------------- | --------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| Web app, any UI | Next.js App Router, TypeScript, Tailwind v4                                 | It must be a static page with no build - then plain HTML            |
| CLI             | TypeScript on Bun, `clipanion` to parse, `bun build --compile` for a binary | It must run under a Node the user already has - then Node and `tsx` |
| Library         | TypeScript, `tsdown` to build, `vitest` to test                             | It is Python - then `uv` and `pytest`                               |
| Service or API  | Hono on Node                                                                | It is already going to be a Next.js app - then route handlers       |
| Script          | One file, `uv run` for Python or `tsx` for TypeScript                       | It grows a second file - then it is a CLI, go back a row            |

### Package manager

`pnpm`, except on a project that runs on Bun - a single-binary CLI, a Bun-native service - where it is `bun`. Never `npm`; a `package-lock.json` in a new project is a mistake to correct, not a default to accept.

Every command from here on is written in `pnpm` form. On a Bun project, substitute:

| Job                   | pnpm            | bun             |
| --------------------- | --------------- | --------------- |
| Add a dev dependency  | `pnpm add -D x` | `bun add -d x`  |
| Run a one-off         | `pnpm dlx x`    | `bunx x`        |
| Run a local binary    | `pnpm exec x`   | `bun x`         |
| Run a script          | `pnpm build`    | `bun run build` |
| Install from lockfile | `pnpm install`  | `bun install`   |

`pnpm` keeps `node_modules` strict, so anything a config imports has to be a direct dependency of the project. That is why `eslint`, `jiti`, and every plugin below get installed explicitly rather than leaned on transitively.

Anything JavaScript-flavoured is TypeScript, config files included: `eslint.config.ts`, not `eslint.config.mjs`. Rename what a scaffolder writes in plain JavaScript rather than leaving it.

## 4. Scaffold

Default location is `~/src/<name>`. Confirm it, then create there.

Run the framework's own scaffolder rather than hand-building a tree. For the Next.js default:

```sh
pnpm create next-app@latest <name> --typescript --tailwind --app --eslint --use-pnpm
```

Then, in the new directory:

```sh
git init
```

### Formatting

Set this up before the first commit, so there is never a reformat-the-world diff later.

```sh
pnpm add -D prettier husky lint-staged
pnpm exec husky init
```

`husky init` needs the repository to exist, which is why `git init` comes first. It also writes a placeholder `pre-commit` and a `prepare` script - the placeholder gets replaced below, and the script is what installs the hook on a fresh clone.

`.husky/pre-commit`:

```sh
pnpm exec lint-staged
pnpm typecheck
pnpm test
```

Drop the lines whose scripts the project does not have yet, and say which ones you dropped. `lint-staged` runs first because it only touches staged files and finishes fast; a full typecheck and test run afterwards is what catches the breakage staged files alone cannot show.

`.prettierrc`:

```json
{
  "useTabs": false,
  "tabWidth": 2,
  "printWidth": 80,
  "singleQuote": false,
  "trailingComma": "es5",
  "semi": true,
  "arrowParens": "always"
}
```

`.lintstagedrc`:

```json
{
  "*": "prettier --ignore-unknown --write"
}
```

The `*` glob with `--ignore-unknown` covers Markdown, JSON, and YAML alongside the code, and skips anything Prettier has no parser for.

Add both scripts to `package.json`:

```json
"format": "prettier --write .",
"format:check": "prettier --check ."
```

The hook and the check do different jobs. The hook rewrites **staged** files on the way into a commit, so nobody has to remember to format. `format:check` reads the **whole tree** and exits non-zero, which is what catches a file that reached the repository around the hook - a `--no-verify` commit, a merge, a web edit.

### Linting

Every project gets ESLint, whatever its shape. Copy [`assets/no-comments.ts`](assets/no-comments.ts) to `eslint-rules/no-comments.ts` in the new project - it is a local rule that requires code to explain itself instead of carrying comments that drift out of date.

```sh
pnpm add -D eslint jiti typescript
```

`jiti` is what lets ESLint load a TypeScript config and a TypeScript rule. Without it, `eslint.config.ts` fails to resolve.

`eslint.config.ts`, or the entries to append when the scaffolder already wrote a config - rename its `.mjs` to `.ts` first:

```ts
import noComments from "./eslint-rules/no-comments.ts";

{ ignores: ["eslint-rules/**"] },
{
  files: ["**/*.{js,jsx,ts,tsx}"],
  plugins: { local: { rules: { "no-comments": noComments } } },
  rules: { "local/no-comments": "error" },
},
```

The rule exempts directive comments - `eslint-disable`, `@ts-expect-error`, `prettier-ignore`, shebangs, and triple-slash references - because the escape hatches have to survive the rule. Reasoning that a reader genuinely needs goes in the commit message, the `AGENTS.md`, or a name, and stays true there.

Keep whatever `lint` script the scaffolder wrote, or add `"lint": "eslint ."` when there is none.

### Commit messages

Conventional Commits, enforced rather than remembered:

```sh
pnpm add -D @commitlint/cli @commitlint/config-conventional
echo 'pnpm exec commitlint --edit "$1"' > .husky/commit-msg
```

`.commitlintrc.json`:

```json
{
  "extends": ["@commitlint/config-conventional"]
}
```

JSON rather than TypeScript here, matching `.prettierrc` and `.lintstagedrc` - a config with no logic in it has nothing for types to check.

Check both directions before moving on. A bad message must be rejected and a good one accepted:

```sh
echo "add stuff" | pnpm exec commitlint          # must report type-empty
echo "feat: add stuff" | pnpm exec commitlint    # must print nothing
```

### AGENTS.md

Write it while the decisions are still fresh. It carries what the next agent cannot read off the config: the shape, the one thing it must do, and the conventions chosen here. Include these lines verbatim:

```md
- Run `pnpm lint` and `pnpm format:check` after making changes, and fix everything they report.
- Fix any error you notice - a failing type check, a lint error, a failing test, a bug in code you read - even when it sits outside the task you were given. Say what you fixed. If a fix would be large or risky, finish the task first, then describe the problem and ask.
- Code carries no comments. The `local/no-comments` rule enforces it.
- Commit messages are Conventional Commits (`feat:`, `fix:`, `chore:`). The `commit-msg` hook rejects anything else.
```

### First commit

```sh
git add -A
git commit -m "chore: initial commit"
```

This commit runs both hooks, which makes it the smoke test: it proves formatting is applied on the way in and that the message passes commitlint. A clean `git status` afterwards is the signal.

**Building a UI?** Read [`FRONTEND.md`](FRONTEND.md) and follow it before writing a single component. It wires shadcn/ui, the Geist Design System, and `@shadcn/lint` together, and the wiring is much cheaper before components exist than after.

## 5. Make it green

Run the project's own commands and read the output:

```sh
pnpm build
pnpm lint
pnpm format:check
pnpm dev     # boot it, confirm it serves, stop it
```

Fix everything they report, including findings in code the scaffolder wrote rather than you. A template's output is not automatically clean - unused imports, a missing env var, and the comments `create-next-app` sprinkles through its boilerplate, which `local/no-comments` will now flag.

**Done when** all four exit clean and the tree is committed. Report the path, the framework, and the commands the user runs day to day.

## Stop and ask

- The directory already exists and is not empty.
- The user's idea is close enough to an existing repository of theirs that the answer might be a branch, not a new project.
- A scaffolder wants to overwrite something.

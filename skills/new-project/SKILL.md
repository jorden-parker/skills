---
name: new-project
description: Initialise project tooling in an existing project or scaffold a new app, CLI, library, or service. Use when setting up a project or adding its formatting, linting, Git hooks, and agent instructions; infer the path from the current directory.
argument-hint: "optional: the idea, project path, or setup needed"
---

# New project

Bring a new or existing project to **green**: its applicable build, lint, formatting, typecheck, test, and runtime checks pass.

## 0. Infer the target

Start in the current directory unless the user specifies another path. Read its instructions, manifests, lockfiles, source layout, and Git status; inspect parent directories to identify the project or workspace root.

- **Existing project:** source, a manifest, or project configuration identifies a project here or in a containing directory. Initialise missing tooling in place. Retain its name, language, framework, package manager, and workspace layout. In a monorepo, use the current package as the target and respect shared root configuration.
- **New project in place:** the directory is empty or contains only starter material such as Git metadata, a README, or a licence. Use this directory and infer the name from it, preserving those files.
- **New project in a child directory:** the current directory is a collection of projects or a general location such as `~/src` or the home directory. Create `<name>` beneath it.

An explicit request for a separate new project takes precedence over an existing project in the current directory. Ask only when the target remains ambiguous or creating it would overwrite existing files. State the selected path and mode, then proceed.

**Done when** the target path, mode, existing tooling, and pre-existing changes are identified. For an existing project, infer the brief from its files, skip naming and framework selection, and continue at step 4.

## 1. Pin the shape

The **shape** is the kind of thing being built. It decides the framework, the layout, and half the naming vocabulary, so settle it first.

Ask whatever you cannot infer from the user's sentence, then write a brief of at most one paragraph covering:

- Shape: web app, CLI, library, service, or script.
- Who runs it, and where.
- The one thing it must do to be worth existing.
- Anything already decided: a language, a host, an existing repo it must sit beside.

**Done when** the brief identifies the shape and purpose, with only decisions that materially affect scaffolding clarified with the user.

## 2. Name it

Use a name supplied by the user or inferred from the target directory. When neither supplies one, names come from the brief:

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

**Done when** the name is supplied, inferred from the target directory, or selected by the user. If you presented candidates, wait for their choice before creating the directory.

## 3. Pick the framework

For a new project, take the default for the shape unless the brief names a reason to move off it. State the default and the reason in one line, then go.

| Shape           | Default                                                                     | Move off it when                                                    |
| --------------- | --------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| Web app, any UI | Next.js App Router, TypeScript, Tailwind v4                                 | It must be a static page with no build - then plain HTML            |
| CLI             | TypeScript on Bun, `clipanion` to parse, `bun build --compile` for a binary | It must run under a Node the user already has - then Node and `tsx` |
| Library         | TypeScript, `tsdown` to build, `vitest` to test                             | It is Python - then `uv` and `pytest`                               |
| Service or API  | Hono on Node                                                                | It is already going to be a Next.js app - then route handlers       |
| Script          | One file, `uv run` for Python or `tsx` for TypeScript                       | It grows a second file - then it is a CLI, go back a row            |

### Package manager

For new JavaScript/TypeScript projects, use `pnpm`, except on a project that runs on Bun, where it is `bun`. For existing projects, infer the package manager from its declared version, lockfile, and instructions; preserve it. Adapt the commands below to that manager and the project’s language.

Every command from here on is written in `pnpm` form. On a Bun project, substitute:

| Job                   | pnpm            | bun             |
| --------------------- | --------------- | --------------- |
| Add a dev dependency  | `pnpm add -D x` | `bun add -d x`  |
| Run a one-off         | `pnpm dlx x`    | `bunx x`        |
| Run a local binary    | `pnpm exec x`   | `bun x`         |
| Run a script          | `pnpm build`    | `bun run build` |
| Install from lockfile | `pnpm install`  | `bun install`   |

`pnpm` keeps `node_modules` strict, so anything a config imports has to be a direct dependency of the project. That is why `eslint`, `jiti`, and every plugin below get installed explicitly rather than leaned on transitively.

In new projects, anything JavaScript-flavoured is TypeScript, config files included: `eslint.config.ts`, not `eslint.config.mjs`. Rename what a scaffolder writes in plain JavaScript rather than leaving it.

## 4. Initialise

For an existing project, first run available checks to establish a baseline. Use its existing package manager for all commands below. Work in place: compare each setup section below with what is installed, retain equivalent working tooling, and add what is missing. Merge configuration, scripts, hooks, and agent instructions into their existing owners. A repeat run should leave already-complete setup unchanged. Preserve existing source and user changes; a tooling setup is not a framework, package-manager, or design-system migration.

For a new project, scaffold into the path selected in step 0. Use `.` as the destination when already inside that directory. Run the framework's own scaffolder when available; if it cannot preserve starter files, generate in a temporary directory and merge the scaffold. For the Next.js default:

```sh
pnpm create next-app@latest <name> --typescript --tailwind --app --eslint --use-pnpm
```

In the target directory, initialise Git only if it is not already inside a repository:

```sh
git init
```

### Formatting

For a new project, use the defaults below. For an existing project, retain its formatting choices and hook runner; add missing checks to its hooks and preserve existing commands, including `prepare`. Run `husky init` only when Husky is absent, and preserve any existing hooks it would replace.

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

Add missing formatting scripts to `package.json`, retaining existing equivalents:

```json
"format": "prettier --write .",
"format:check": "prettier --check ."
```

The hook and the check do different jobs. The hook rewrites **staged** files on the way into a commit, so nobody has to remember to format. `format:check` reads the **whole tree** and exits non-zero, which is what catches a file that reached the repository around the hook - a `--no-verify` commit, a merge, a web edit.

### Linting

For JavaScript/TypeScript projects, add ESLint when absent and extend the existing lint setup. For other languages, use their native lint and formatting tools; omit JavaScript-only rules unless the project also contains JavaScript/TypeScript. Copy [`assets/no-comments.ts`](assets/no-comments.ts) to `eslint-rules/no-comments.ts` in the target project if absent - it is a local rule that requires code to explain itself instead of carrying comments that drift out of date.

```sh
pnpm add -D eslint jiti typescript
```

`jiti` is what lets ESLint load a TypeScript config and a TypeScript rule. Without it, `eslint.config.ts` fails to resolve.

`eslint.config.ts`, or the entries to merge into an existing config without duplicating the rule. Keep an existing project’s config format; new scaffolds use `.ts`:

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

Add Conventional Commits enforcement when absent, using the existing hook runner. Preserve existing commitlint configuration and merge the hook command rather than replacing other checks. For a new Husky setup:

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

Create or update the applicable `AGENTS.md`, preserving existing instructions and avoiding duplicates. Record the shape, purpose, and conventions established here. Adapt these lines to the actual package manager and installed checks; include the no-comments and commit-message instructions only when their enforcement is configured:

```md
- Run `pnpm lint` and `pnpm format:check` after making changes, and fix everything they report.
- Fix any error you notice - a failing type check, a lint error, a failing test, a bug in code you read - even when it sits outside the task you were given. Say what you fixed. If a fix would be large or risky, finish the task first, then describe the problem and ask.
- Code carries no comments. The `local/no-comments` rule enforces it.
- Commit messages are Conventional Commits (`feat:`, `fix:`, `chore:`). The `commit-msg` hook rejects anything else.
```

**Building a UI?** Read [`FRONTEND.md`](FRONTEND.md). Apply its setup to a new frontend before writing components; for an existing frontend, follow its compatibility guidance.

**Done when** every applicable setup section is configured or already satisfied, with any incompatible or deferred additions explained.

## 5. Make it green

Run the project's own commands and read the output:

```sh
pnpm build
pnpm lint
pnpm format:check
pnpm dev     # boot it, confirm it serves, stop it
```

Use the scripts actually available, including typecheck and tests when present. For a CLI, library, or script, use an appropriate smoke check instead of starting a web server. A dev server is verified when it serves successfully and is then stopped.

Fix failures introduced by the setup. In an existing project, compare failures against the original state; report pre-existing failures separately. If a new rule requires widespread source cleanup, scope adoption explicitly or ask before undertaking that migration. Never describe failing checks as green.

For a new project, stage the generated setup and make `chore: initial commit` after verification, provided there are no unrelated pre-existing changes. For an existing project, leave changes ready for review unless the user requested a commit; when committing, stage only the task’s changes. Exercise configured hooks and check that commitlint rejects an invalid message and accepts a valid one.

**Done when** the applicable checks pass, or remaining failures are clearly identified with their cause and blocker. Review the diff for unrelated changes. Report the target path, what was initialised or retained, and verification results.

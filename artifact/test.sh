#!/usr/bin/env bash
# Smoke tests for artifact/render. Never opens a browser.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RENDER="$REPO/artifact/render"
pass=0; fail=0; n=0
ok()   { n=$((n+1)); pass=$((pass+1)); printf 'ok %d - %s\n' "$n" "$1"; }
nok()  { n=$((n+1)); fail=$((fail+1)); printf 'not ok %d - %s\n    %s\n' "$n" "$1" "$2"; }
assert_eq() { if [ "$2" = "$3" ]; then ok "$1"; else nok "$1" "expected [$2] got [$3]"; fi; }
assert_contains() { if grep -qF -- "$2" <<<"$3"; then ok "$1"; else nok "$1" "missing [$2]"; fi; }
assert_not_contains() { if grep -qF -- "$2" <<<"$3"; then nok "$1" "unexpected [$2]"; else ok "$1"; fi; }

run_render() { set +e; out="$("$RENDER" "$@" --no-open 2>&1)"; rc=$?; set -e; }
T="$(cd "$(mktemp -d "${TMPDIR:-/tmp}/artifact-test.XXXXXX")" && pwd -P)" # physical path: render prints a resolved one

cat > "$T/good.md" <<'MD'
## Answer

Node 24 is the LTS line [1]. It bundles npm 11 [1][2].

## Findings

Node 24 entered LTS in October 2025 [1].

> "Node.js 24 is now LTS"

- npm 11 ships with it [2].

```js
const x = arr[9]; // [9] is code, not a marker
```

## Sources

1. Release schedule - Node.js, published 2025-10-28. https://nodejs.org/en/about/releases (accessed 2026-09-17)
   > "Node.js 24 is now LTS"
2. npm 11 notes - GitHub, undated. https://github.com/npm/cli/releases (accessed 2026-09-17)
MD

# green: renders, escapes, embeds
run_render "$T/good.md" --title 'x</title><script>' --eyebrow "Web research" --cited-section Findings --output "$T/o.html"
assert_eq "green exits 0" 0 "$rc"
assert_contains "prints output path" "$T/o.html" "$out"
page="$(cat "$T/o.html")"
assert_contains "title escaped" "x&lt;/title&gt;&lt;script&gt;" "$page"
assert_contains "eyebrow embedded" "Web research" "$page"
assert_contains "body embedded" "## Findings" "$page"
assert_contains "geist tokens inlined" "--ds-gray-1000" "$page"

# existing output is never overwritten
run_render "$T/good.md" --title t --output "$T/o.html"
assert_eq "existing output exits 1" 1 "$rc"
assert_contains "existing output message" "File exists" "$out"

# no --output: a fresh file in the temp directory
run_render "$T/good.md" --title "Which Node is LTS?"
assert_eq "default output exits 0" 0 "$rc"
assert_contains "default output is slugged" "/artifacts/which-node-is-lts-" "$out"
rm -f "$out"

# red: each kind of citation problem is named, and nothing is written
sed -e 's/ships with it \[2\]/ships with it/' -e 's/bundles npm 11 \[1\]\[2\]/bundles npm 11 [1][7]/' \
    -e 's| https://nodejs.org/en/about/releases||' "$T/good.md" > "$T/bad.md"
run_render "$T/bad.md" --title t --cited-section Findings --output "$T/bad.html"
assert_eq "red exits 2" 2 "$rc"
assert_contains "missing URL named" "source 1 has no URL" "$out"
assert_contains "dangling marker named" "marker [7] has no Sources item" "$out"
assert_contains "unused source named" "source 2 is never cited" "$out"
assert_contains "uncited block named" "no marker under \`Findings\`: - npm 11 ships with it." "$out"
assert_not_contains "code is not a marker" "[9]" "$out"
if [ -e "$T/bad.html" ]; then nok "no file when red" "bad.html exists"; else ok "no file when red"; fi

# red: no Sources heading at all
printf '# Notes\n\nA claim [1].\n' > "$T/none.md"
run_render "$T/none.md" --title t --citations
assert_eq "no Sources exits 2" 2 "$rc"
assert_contains "no Sources message" "no \`Sources\` heading" "$out"

# without --citations the same file renders
run_render "$T/none.md" --title t --output "$T/none.html"
assert_eq "unchecked render exits 0" 0 "$rc"

# missing body
run_render "$T/nope.md" --title t
assert_eq "missing body exits 1" 1 "$rc"

rm -rf "$T"
printf '1..%d\n' "$n"
if [ "$fail" -eq 0 ]; then printf '# all %d tests passed\n' "$n"; else printf '# %d of %d tests FAILED\n' "$fail" "$n"; exit 1; fi

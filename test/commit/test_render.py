import html
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

RENDER = Path(__file__).resolve().parents[2] / "skills" / "commit" / "render"
TAG_PATTERN = re.compile(r"<[^>]+>")


def text_of(page):
    return html.unescape(TAG_PATTERN.sub("", page))


class RenderTestCase(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.repo = Path(temporary.name, "repo")
        self.repo.mkdir()
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Test")
        self.git("config", "user.email", "test@example.com")
        self.write("README.md", "start\n")
        self.git("add", "README.md")
        self.git("commit", "-q", "-m", "chore: start")
        self.base = self.git("rev-parse", "HEAD").strip()
        self.git_dir = Path(self.git("rev-parse", "--absolute-git-dir").strip())
        self.plan_dir = self.git_dir / "commit-plan"
        self.plan_dir.mkdir()

    def git(self, *arguments):
        result = subprocess.run(["git", *arguments], cwd=self.repo, capture_output=True, text=True, check=False)
        if result.returncode:
            raise AssertionError(result.stderr)
        return result.stdout

    def write(self, name, text):
        path = self.repo / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")

    def render(self, plan, *extra):
        plan_path = self.plan_dir / "plan.json"
        head = self.git("rev-parse", "HEAD").strip()
        plan_path.write_text(json.dumps({"base": head, **plan}), encoding="utf-8")
        return subprocess.run(
            [str(RENDER), "--no-open", str(plan_path), *extra],
            cwd=self.repo,
            capture_output=True,
            text=True,
            check=False,
        )

    def page(self):
        return (self.plan_dir / "index.html").read_text(encoding="utf-8")

    def commit(self, subject, paths, **extra):
        return {"type": "feat", "subject": subject, "paths": paths, **extra}

    def write_hunk_patches(self):
        numbered = "".join(f"line {number}\n" for number in range(1, 21))
        self.write("notes.txt", numbered)
        self.git("add", "notes.txt")
        self.git("commit", "-q", "-m", "chore: add notes")
        edited = numbered.replace("line 1\n", "first\n").replace("line 20\n", "last\n")
        self.write("notes.txt", edited)
        header, first, second = split_diff(self.git("diff", "--", "notes.txt"))
        (self.plan_dir / "patch-01.diff").write_text(header + first, encoding="utf-8")
        (self.plan_dir / "patch-02.diff").write_text(header + second, encoding="utf-8")


def split_diff(diff):
    lines = diff.splitlines(keepends=True)
    hunk_starts = [index for index, line in enumerate(lines) if line.startswith("@@")]
    header = "".join(lines[: hunk_starts[0]])
    first = "".join(lines[hunk_starts[0] : hunk_starts[1]])
    second = "".join(lines[hunk_starts[1] :])
    return header, first, second


class HappyPathTest(RenderTestCase):
    def test_writes_one_card_per_commit_and_the_full_message(self):
        self.write("cli.py", "run\n")
        self.write("test_cli.py", "test\n")
        result = self.render(
            {
                "commits": [
                    self.commit("add --dry-run flag", ["cli.py"], scope="cli", body="Why.", footers=["Refs: #42"], rationale="The flag and its caller."),
                    {"type": "test", "subject": "cover dry run", "paths": ["test_cli.py"], "rationale": "Test only."},
                ]
            }
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), str(self.plan_dir / "index.html"))
        page = text_of(self.page())
        self.assertIn("feat(cli): add --dry-run flag", page)
        self.assertIn("test: cover dry run", page)
        self.assertEqual((self.plan_dir / "msg-01.txt").read_text(encoding="utf-8"), "feat(cli): add --dry-run flag\n\nWhy.\n\nRefs: #42\n")


class CoverageTest(RenderTestCase):
    def test_a_changed_file_in_no_commit_is_named_and_rejected(self):
        self.write("README.md", "changed\n")
        self.write("cli.py", "run\n")
        result = self.render({"commits": [self.commit("add cli", ["cli.py"])]})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("README.md", result.stderr)

    def test_the_same_file_in_two_commits_is_rejected(self):
        self.write("cli.py", "run\n")
        result = self.render(
            {"commits": [self.commit("add cli", ["cli.py"]), self.commit("tweak cli", ["cli.py"])]}
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("cli.py", result.stderr)

    def test_untracked_deleted_and_moved_files_are_committed(self):
        self.write("old_name.txt", "moved\n")
        self.write("gone.txt", "deleted\n")
        self.git("add", "old_name.txt", "gone.txt")
        self.git("commit", "-q", "-m", "chore: add files")
        os.rename(self.repo / "old_name.txt", self.repo / "new_name.txt")
        (self.repo / "gone.txt").unlink()
        self.write("fresh.txt", "new\n")
        result = self.render(
            {"commits": [self.commit("reshuffle files", ["old_name.txt", "new_name.txt", "gone.txt", "fresh.txt"])]}
        )
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_an_excluded_file_is_left_out_with_its_reason(self):
        self.write("secrets.txt", "token\n")
        self.write("cli.py", "run\n")
        result = self.render(
            {
                "commits": [self.commit("add cli", ["cli.py"])],
                "excluded": [{"path": "secrets.txt", "reason": "looks like a secret"}],
            }
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("Left out", text_of(self.page()))
        self.assertIn("looks like a secret", text_of(self.page()))


class HeaderTest(RenderTestCase):
    def test_unknown_type_is_rejected(self):
        self.write("cli.py", "run\n")
        result = self.render({"commits": [{"type": "wip", "subject": "add cli", "paths": ["cli.py"]}]})
        self.assertNotEqual(result.returncode, 0)

    def test_long_subject_is_rejected(self):
        self.write("cli.py", "run\n")
        result = self.render({"commits": [self.commit("x" * 80, ["cli.py"])]})
        self.assertNotEqual(result.returncode, 0)

    def test_subject_ending_in_a_full_stop_is_rejected(self):
        self.write("cli.py", "run\n")
        result = self.render({"commits": [self.commit("add cli.", ["cli.py"])]})
        self.assertNotEqual(result.returncode, 0)

    def test_breaking_change_is_written_with_a_bang(self):
        self.write("cli.py", "run\n")
        result = self.render({"commits": [self.commit("drop old flag", ["cli.py"], breaking=True)]})
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue((self.plan_dir / "msg-01.txt").read_text(encoding="utf-8").startswith("feat!: drop old flag\n"))

    def test_subject_is_escaped_in_the_page(self):
        self.write("cli.py", "run\n")
        result = self.render({"commits": [self.commit("guard <script>alert(1)</script>", ["cli.py"])]})
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("guard &lt;script&gt;alert(1)&lt;/script&gt;", self.page())
        self.assertNotIn("<script>alert(1)</script>", self.page())


class HunkTest(RenderTestCase):
    def test_two_patches_for_one_file_replay_to_the_working_tree(self):
        self.write_hunk_patches()
        result = self.render(
            {
                "commits": [
                    {"type": "fix", "subject": "fix first line", "patch": "patch-01.diff"},
                    {"type": "fix", "subject": "fix last line", "patch": "patch-02.diff"},
                ]
            }
        )
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_a_patch_that_does_not_apply_leaves_the_index_untouched(self):
        self.write_hunk_patches()
        bogus = "diff --git a/notes.txt b/notes.txt\n--- a/notes.txt\n+++ b/notes.txt\n@@ -1 +1 @@\n-missing\n+changed\n"
        (self.plan_dir / "patch-01.diff").write_text(bogus, encoding="utf-8")
        index_before = (self.git_dir / "index").read_bytes()
        result = self.render({"commits": [{"type": "fix", "subject": "fix first line", "patch": "patch-01.diff"}]})
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual((self.git_dir / "index").read_bytes(), index_before)


class ReadOnlyTest(RenderTestCase):
    def test_the_repository_is_unchanged_after_a_run(self):
        self.write("cli.py", "run\n")
        self.write("README.md", "changed\n")
        self.git("status", "--porcelain")
        index_before = (self.git_dir / "index").read_bytes()
        objects_before = sorted(str(path) for path in (self.git_dir / "objects").rglob("*"))
        head_before = self.git("rev-parse", "HEAD")
        status_before = self.git("status", "--porcelain")
        result = self.render({"commits": [self.commit("add cli", ["cli.py"]), self.commit("update readme", ["README.md"])]})
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.git_dir / "index").read_bytes(), index_before)
        self.assertEqual(sorted(str(path) for path in (self.git_dir / "objects").rglob("*")), objects_before)
        self.assertEqual(self.git("rev-parse", "HEAD"), head_before)
        self.assertEqual(self.git("status", "--porcelain"), status_before)


class StaleBaseTest(RenderTestCase):
    def test_a_plan_made_against_an_older_head_is_rejected(self):
        self.write("cli.py", "run\n")
        self.git("commit", "-q", "--allow-empty", "-m", "chore: later")
        result = self.render({"commits": [self.commit("add cli", ["cli.py"])], "base": self.base})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("HEAD", result.stderr)


class IgnoreTest(RenderTestCase):
    def ignore_plan(self, **ignore):
        return {
            "ignore": {"subject": "ignore debug logs", "patterns": ["*.log"], "reasons": {"*.log": "debug output"}, **ignore},
            "commits": [self.commit("add cli", ["cli.py"])],
        }

    def test_the_ignore_commit_comes_first(self):
        self.write("cli.py", "run\n")
        self.write("debug.log", "noise\n")
        result = self.render(self.ignore_plan())
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.plan_dir / "msg-01.txt").read_text(encoding="utf-8"), "chore: ignore debug logs\n")
        self.assertIn("git add .gitignore", text_of(self.page()))

    def test_gitignore_without_a_final_newline_starts_the_append_with_a_newline(self):
        self.write(".gitignore", "dist/")
        self.git("add", ".gitignore")
        self.git("commit", "-q", "-m", "chore: add gitignore")
        self.write("cli.py", "run\n")
        self.write("debug.log", "noise\n")
        result = self.render(self.ignore_plan())
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("printf '%s\\n' '' '*.log' >> .gitignore", text_of(self.page()))

    def test_gitignore_that_is_missing_needs_no_leading_newline(self):
        self.write("cli.py", "run\n")
        self.write("debug.log", "noise\n")
        result = self.render(self.ignore_plan())
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("printf '%s\\n' '*.log' >> .gitignore", text_of(self.page()))

    def test_a_tracked_match_is_removed_from_the_index_when_listed(self):
        self.write("build.log", "old\n")
        self.git("add", "build.log")
        self.git("commit", "-q", "-m", "chore: add log")
        self.write("cli.py", "run\n")
        result = self.render(self.ignore_plan(untrack=["build.log"]))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("git rm -r --cached --quiet -- build.log", text_of(self.page()))

    def test_a_tracked_match_missing_from_untrack_is_rejected(self):
        self.write("build.log", "old\n")
        self.git("add", "build.log")
        self.git("commit", "-q", "-m", "chore: add log")
        self.write("cli.py", "run\n")
        result = self.render(self.ignore_plan())
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("build.log", result.stderr)

    def test_a_pattern_already_in_gitignore_is_rejected(self):
        self.write(".gitignore", "*.log\n")
        self.git("add", ".gitignore")
        self.git("commit", "-q", "-m", "chore: add gitignore")
        self.write("cli.py", "run\n")
        self.write("debug.log", "noise\n")
        result = self.render(self.ignore_plan())
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("*.log", result.stderr)


if __name__ == "__main__":
    unittest.main()

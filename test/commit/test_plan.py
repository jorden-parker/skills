import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

PLAN = Path(__file__).resolve().parents[2] / "skills" / "jorden" / "commit" / "plan"


class PlanTestCase(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.repo = Path(temporary.name, "repo")
        self.repo.mkdir()
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Test")
        self.git("config", "user.email", "test@example.com")
        (self.repo / "README.md").write_text("start\n", encoding="utf-8")
        self.git("add", "README.md")
        self.git("commit", "-q", "-m", "chore: start")
        self.head = self.git("rev-parse", "HEAD").strip()
        self.git_dir = Path(self.git("rev-parse", "--absolute-git-dir").strip())
        self.skills_tmpdir = Path(temporary.name, "skills-tmp")
        (self.repo / "a.txt").write_text("a\n", encoding="utf-8")

    def git(self, *arguments):
        result = subprocess.run(["git", *arguments], cwd=self.repo, capture_output=True, text=True, check=False)
        if result.returncode:
            raise AssertionError(result.stderr)
        return result.stdout

    def plan(self, *arguments, document=None):
        environment = {**os.environ, "SKILLS_TMPDIR": str(self.skills_tmpdir)}
        stdin = json.dumps(document) if document is not None else ""
        return subprocess.run([str(PLAN), *arguments], cwd=self.repo, input=stdin, capture_output=True, text=True, check=False, env=environment)

    def build(self, document):
        result = self.plan("build", document=document)
        self.assertEqual(result.returncode, 0, result.stderr)
        page = Path(result.stdout.strip())
        return page, json.loads((page.parent / "plan.json").read_text(encoding="utf-8"))

    def one_commit(self, **extra):
        return {"commits": [{"header": "feat: add a", "why": "new file", "paths": ["a.txt"]}], **extra}

    def test_plan_lives_outside_the_git_dir(self):
        before = sorted(str(path) for path in self.git_dir.rglob("*"))
        page, _ = self.build(self.one_commit())
        self.assertEqual(page.name, "index.html")
        self.assertTrue(str(page).startswith(str(self.skills_tmpdir.resolve() / "commit") + "/"))
        self.assertEqual(sorted(str(path) for path in self.git_dir.rglob("*")), before)
        self.assertEqual((self.skills_tmpdir / "commit").stat().st_mode & 0o777, 0o700)

    def test_build_records_head_and_splits_header(self):
        document = {"commits": [{"header": "feat(cli)!: add x", "why": "w", "paths": ["a.txt"]}]}
        _, saved = self.build(document)
        self.assertEqual(saved["base"], self.head)
        commit = saved["commits"][0]
        self.assertEqual(
            (commit["type"], commit["scope"], commit["breaking"], commit["subject"], commit["paths"], commit["rationale"]),
            ("feat", "cli", True, "add x", ["a.txt"], "w"),
        )

    def test_bad_header_is_rejected_and_plan_unchanged(self):
        page, _ = self.build(self.one_commit())
        before = (page.parent / "plan.json").read_text(encoding="utf-8")
        result = self.plan("build", document={"commits": [{"header": "Add x", "why": "w", "paths": ["a.txt"]}]})
        self.assertEqual(result.returncode, 1)
        self.assertIn("header must look like", result.stderr)
        self.assertEqual((page.parent / "plan.json").read_text(encoding="utf-8"), before)

    def test_build_needs_commits(self):
        result = self.plan("build", document={"commits": []})
        self.assertEqual(result.returncode, 1)
        self.assertIn("non-empty commits list", result.stderr)

    def test_commit_needs_paths_or_patch(self):
        result = self.plan("build", document={"commits": [{"header": "feat: x", "why": "w"}]})
        self.assertEqual(result.returncode, 1)
        self.assertIn("commit 1 needs paths or patch", result.stderr)

    def test_ignore_and_excluded_are_saved(self):
        (self.repo / "secret.txt").write_text("token\n", encoding="utf-8")
        ignore = {"patterns": {"dist/": "build output"}, "untrack": []}
        excluded = [{"path": "secret.txt", "reason": "looks like a secret"}]
        _, saved = self.build(self.one_commit(ignore=ignore, excluded=excluded))
        self.assertEqual(saved["ignore"], {"subject": "ignore local files", "patterns": ["dist/"], "reasons": {"dist/": "build output"}, "untrack": []})
        self.assertEqual(saved["excluded"], excluded)

    def test_gather_shows_untracked_contents(self):
        result = self.plan("gather")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("### a.txt\na\n", result.stdout)


if __name__ == "__main__":
    unittest.main()

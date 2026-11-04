import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SKILL_DIRECTORY = Path(__file__).resolve().parents[2] / "skills" / "commit"
PLAN = SKILL_DIRECTORY / "plan"
RENDER = SKILL_DIRECTORY / "render"


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

    def git(self, *arguments):
        result = subprocess.run(["git", *arguments], cwd=self.repo, capture_output=True, text=True, check=False)
        if result.returncode:
            raise AssertionError(result.stderr)
        return result.stdout

    def run_tool(self, tool, *arguments):
        environment = {**os.environ, "SKILLS_TMPDIR": str(self.skills_tmpdir)}
        return subprocess.run([str(tool), *arguments], cwd=self.repo, capture_output=True, text=True, check=False, env=environment)

    def plan(self, *arguments):
        return self.run_tool(PLAN, *arguments)

    @property
    def plan_file(self):
        return Path(self.plan("dir").stdout.strip(), "plan.json")

    def saved(self):
        return json.loads(self.plan_file.read_text(encoding="utf-8"))

    def test_plan_lives_outside_the_git_dir(self):
        before = sorted(str(path) for path in self.git_dir.rglob("*"))
        result = self.plan("new")
        self.assertEqual(result.returncode, 0, result.stderr)
        directory = self.plan("dir").stdout.strip()
        self.assertEqual(result.stdout.strip(), directory + "/plan.json")
        self.assertTrue(directory.startswith(str(self.skills_tmpdir / "commit") + "/"))
        self.assertEqual(sorted(str(path) for path in self.git_dir.rglob("*")), before)
        self.assertEqual((self.skills_tmpdir / "commit").stat().st_mode & 0o777, 0o700)

    def test_new_records_head_and_empty_commits(self):
        result = self.plan("new")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), str(self.plan_file))
        self.assertEqual(self.saved(), {"base": self.head, "commits": []})

    def test_commit_splits_header(self):
        self.plan("new")
        result = self.plan("commit", "feat(cli)!: add x", "--why", "w", "a.txt")
        self.assertEqual(result.returncode, 0, result.stderr)
        commit = self.saved()["commits"][0]
        self.assertEqual(
            (commit["type"], commit["scope"], commit["breaking"], commit["subject"], commit["paths"]),
            ("feat", "cli", True, "add x", ["a.txt"]),
        )

    def test_bad_header_is_rejected_and_plan_unchanged(self):
        self.plan("new")
        before = self.plan_file.read_text(encoding="utf-8")
        result = self.plan("commit", "Add x", "--why", "w", "a.txt")
        self.assertEqual(result.returncode, 1)
        self.assertEqual(self.plan_file.read_text(encoding="utf-8"), before)

    def test_missing_plan_asks_for_new(self):
        result = self.plan("commit", "feat: x", "--why", "w", "a.txt")
        self.assertEqual(result.returncode, 1)
        self.assertIn("plan new", result.stderr)

    def test_ignore_calls_build_one_block(self):
        self.plan("new")
        self.plan("ignore", "dist/", "--reason", "build output")
        self.plan("ignore", ".env.local", "--reason", "secrets", "--subject", "ignore local files", "--untrack", "old.log")
        ignore = self.saved()["ignore"]
        self.assertEqual(ignore["patterns"], ["dist/", ".env.local"])
        self.assertEqual(ignore["reasons"], {"dist/": "build output", ".env.local": "secrets"})
        self.assertEqual(ignore["untrack"], ["old.log"])

    def test_exclude_appends(self):
        self.plan("new")
        self.plan("exclude", "secret.txt", "--reason", "looks like a secret")
        self.assertEqual(self.saved()["excluded"], [{"path": "secret.txt", "reason": "looks like a secret"}])

    def test_plan_renders(self):
        (self.repo / "a.txt").write_text("a\n", encoding="utf-8")
        self.plan("new")
        self.plan("commit", "feat: add a", "--why", "new file", "a.txt")
        result = self.run_tool(RENDER, str(self.plan_file))
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()

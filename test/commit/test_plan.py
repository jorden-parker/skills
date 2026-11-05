import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

PLAN = Path(__file__).resolve().parents[2] / "catalog" / "jorden" / "commit" / "plan"
RENDER = Path(__file__).resolve().parents[2] / "catalog" / "jorden" / "commit" / "render"


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

    def test_gather_on_a_clean_tree_says_nothing_to_commit(self):
        (self.repo / "a.txt").unlink()
        result = self.plan("gather")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("nothing to commit", result.stdout)

    def gather(self):
        result = self.plan("gather")
        self.assertEqual(result.returncode, 0, result.stderr)
        return result.stdout

    def test_gather_convention_reads_commitlint_enum(self):
        rules = {"type-enum": [2, "always", ["feat", "fix"]], "scope-enum": [2, "always", ["cli"]], "header-max-length": [2, "always", 50]}
        (self.repo / ".commitlintrc.json").write_text(json.dumps({"rules": rules}), encoding="utf-8")
        output = self.gather()
        self.assertIn("## convention", output)
        self.assertIn("types: feat fix", output)
        self.assertIn("scopes: cli", output)
        self.assertIn("max header length: 50", output)

    def test_gather_convention_defaults(self):
        output = self.gather()
        self.assertIn("scopes: any", output)
        self.assertIn("max header length: 72", output)
        self.assertIn("feat", output.split("types:")[1].splitlines()[0])

    def test_gather_lists_untracked_ignore_candidates(self):
        (self.repo / "dist").mkdir()
        (self.repo / "dist" / "out.js").write_text("x\n", encoding="utf-8")
        section = self.gather().split("## ignore candidates")[1].split("\n## ")[0]
        self.assertIn("dist/", section)
        self.assertNotIn("untrack", section)

    def test_gather_marks_tracked_candidates_untrack(self):
        (self.repo / "debug.log").write_text("x\n", encoding="utf-8")
        self.git("add", "-f", "debug.log")
        self.git("commit", "-q", "-m", "chore: log")
        (self.repo / "debug.log").write_text("y\n", encoding="utf-8")
        section = self.gather().split("## ignore candidates")[1].split("\n## ")[0]
        self.assertIn("*.log", section)
        self.assertIn("untrack", section)

    def test_gather_leaves_out_patterns_already_ignored(self):
        (self.repo / ".gitignore").write_text("dist/\n", encoding="utf-8")
        (self.repo / "dist").mkdir()
        (self.repo / "dist" / "out.js").write_text("x\n", encoding="utf-8")
        (self.repo / "run.log").write_text("x\n", encoding="utf-8")
        section = self.gather().split("## ignore candidates")[1].split("\n## ")[0]
        self.assertIn("run.log", section)
        self.assertNotIn("dist/", section)

    def skeleton(self):
        result = self.plan("skeleton")
        self.assertEqual(result.returncode, 0, result.stderr)
        return json.loads(result.stdout)

    def test_skeleton_holds_changed_paths_and_ignore(self):
        (self.repo / "dist").mkdir()
        (self.repo / "dist" / "out.js").write_text("x\n", encoding="utf-8")
        skeleton = self.skeleton()
        self.assertEqual(skeleton["commits"][0]["paths"], ["a.txt"])
        self.assertEqual(skeleton["commits"][0]["header"], "TODO")
        self.assertIn("dist/", skeleton["ignore"]["patterns"])

    def test_skeleton_is_rejected_by_build_until_header_is_set(self):
        skeleton = self.skeleton()
        result = self.plan("build", document=skeleton)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("header must look like", result.stderr)
        skeleton["commits"][0]["header"] = "feat: add a"
        self.build(skeleton)

    def test_build_rejects_the_no_open_flag(self):
        result = self.plan("build", "--no-open", document=self.one_commit())
        self.assertEqual(result.returncode, 2)
        self.assertIn("unrecognized arguments: --no-open", result.stderr)

    def test_render_rejects_the_no_open_flag(self):
        result = subprocess.run([str(RENDER), "--no-open", "plan.json"], cwd=self.repo, capture_output=True, text=True, check=False)
        self.assertEqual(result.returncode, 2)
        self.assertIn("unrecognized arguments: --no-open", result.stderr)


if __name__ == "__main__":
    unittest.main()

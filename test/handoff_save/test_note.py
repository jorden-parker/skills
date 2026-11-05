import os
from pathlib import Path
import re
import stat
import subprocess
import tempfile
import unittest

NOTE = Path(__file__).resolve().parents[2] / "catalog/jorden/handoff-save/note"
HEADINGS = [
    "Goal",
    "State",
    "Progress",
    "Relevant files",
    "Decisions",
    "Open questions / blockers",
    "Next steps",
    "Verification",
    "Suggested skills",
]


class NoteTestCase(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        self.repo = self.root / "my-repo"
        self.repo.mkdir()
        self.tmpdir = self.root / "skills-tmp"

    def git(self, *arguments):
        subprocess.run(["git", *arguments], cwd=self.repo, capture_output=True, check=True)

    def init_repo(self):
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Test")
        self.git("config", "user.email", "test@example.com")
        (self.repo / "a.txt").write_text("a\n", encoding="utf-8")
        self.git("add", "a.txt")
        self.git("commit", "-q", "-m", "chore: start")

    def note(self, *arguments, cwd=None):
        environment = {**os.environ, "SKILLS_TMPDIR": str(self.tmpdir)}
        return subprocess.run([str(NOTE), *arguments], cwd=cwd or self.repo, capture_output=True, text=True, check=False, env=environment)

    def valid_note(self, extra=""):
        body = "\n".join(f"## {heading}\n\ntext\n" for heading in HEADINGS)
        path = self.root / "note.md"
        path.write_text(body + extra, encoding="utf-8")
        return path

    def test_facts_prints_path_with_repo_name_and_timestamp(self):
        self.init_repo()
        result = self.note("facts")
        self.assertEqual(result.returncode, 0, result.stderr)
        path = result.stdout.split("## note path\n")[1].splitlines()[0]
        self.assertRegex(path, re.escape(str(self.tmpdir / "handoff" / "my-repo-")) + r"\d{8}-\d{6}\.md$")

    def test_facts_creates_directory_with_mode_700(self):
        self.init_repo()
        self.note("facts")
        mode = stat.S_IMODE((self.tmpdir / "handoff").stat().st_mode)
        self.assertEqual(mode, 0o700)

    def test_facts_reports_git_state(self):
        self.init_repo()
        (self.repo / "b.txt").write_text("b\n", encoding="utf-8")
        output = self.note("facts").stdout
        self.assertIn("main", output)
        self.assertIn("?? b.txt", output)
        self.assertIn("chore: start", output)
        self.assertIn("no upstream", output)

    def test_facts_outside_a_repo_skips_git(self):
        result = self.note("facts")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertRegex(result.stdout, r"my-repo-\d{8}-\d{6}\.md")
        self.assertNotIn("no upstream", result.stdout)

    def test_check_passes_for_a_valid_note(self):
        result = self.note("check", str(self.valid_note()))
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_check_fails_on_missing_heading(self):
        path = self.valid_note()
        path.write_text(path.read_text(encoding="utf-8").replace("## Decisions\n", ""), encoding="utf-8")
        result = self.note("check", str(path))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Decisions", result.stderr)

    def test_check_fails_on_headings_out_of_order(self):
        path = self.valid_note()
        text = path.read_text(encoding="utf-8").replace("## Goal", "## TMP").replace("## State", "## Goal").replace("## TMP", "## State")
        path.write_text(text, encoding="utf-8")
        self.assertNotEqual(self.note("check", str(path)).returncode, 0)

    def test_check_fails_on_secret_like_lines(self):
        for secret in ["key sk-abc123", "ghp_abcdef", "AKIAABCDEFG", "-----BEGIN PRIVATE KEY-----", "password=hunter2", "token=abc"]:
            with self.subTest(secret=secret):
                result = self.note("check", str(self.valid_note(secret + "\n")))
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("secret", result.stderr)

    def test_check_allows_password_without_a_value(self):
        result = self.note("check", str(self.valid_note("rotate the password= setting\n")))
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()

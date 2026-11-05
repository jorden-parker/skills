import os
from pathlib import Path
import subprocess
import tempfile
import unittest

NOTES = Path(__file__).resolve().parents[2] / "catalog/jorden/handoff-resume/notes"


class NotesTestCase(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        self.repo = self.root / "my-repo"
        self.repo.mkdir()
        subprocess.run(["git", "init", "-q", "-b", "main"], cwd=self.repo, check=True)
        self.directory = self.root / "skills-tmp" / "handoff"
        self.directory.mkdir(parents=True)

    def add_note(self, stamp, text="note", repo="my-repo"):
        path = self.directory / f"{repo}-{stamp}.md"
        path.write_text(text, encoding="utf-8")
        return path

    def notes(self, *arguments):
        environment = {**os.environ, "SKILLS_TMPDIR": str(self.root / "skills-tmp")}
        return subprocess.run([str(NOTES), *arguments], cwd=self.repo, capture_output=True, text=True, check=False, env=environment).stdout

    def chosen(self, output):
        return output.split("## chosen\n")[1].splitlines()[0]

    def test_no_notes(self):
        self.assertEqual(self.notes().strip(), "no notes")

    def test_newest_note_is_chosen_first(self):
        self.add_note("20260101-000000")
        newest = self.add_note("20260102-000000")
        output = self.notes()
        self.assertEqual(self.chosen(output), str(newest))
        self.assertLess(output.index("20260102-000000"), output.index("20260101-000000"))

    def test_word_picks_newest_matching_note_case_insensitively(self):
        older = self.add_note("20260101-000000", "Fix the Parser")
        self.add_note("20260102-000000", "other work")
        self.assertEqual(self.chosen(self.notes("parser")), str(older))

    def test_other_repos_notes_are_ignored(self):
        self.add_note("20260102-000000", repo="elsewhere")
        self.assertEqual(self.notes().strip(), "no notes")

    def test_lists_at_most_five_notes(self):
        for day in range(1, 8):
            self.add_note(f"2026010{day}-000000")
        listing = self.notes().split("## chosen")[0]
        self.assertEqual(listing.count(".md"), 5)

    def test_prints_repo_state(self):
        self.add_note("20260101-000000")
        output = self.notes()
        self.assertIn("main", output)
        self.assertIn("## status", output)


if __name__ == "__main__":
    unittest.main()

import subprocess
import tempfile
import unittest
from pathlib import Path

PREVIEW = Path(__file__).resolve().parents[2] / "skills/pr-draft/preview"


class PreviewTest(unittest.TestCase):
    def run_preview(self, *arguments):
        return subprocess.run([str(PREVIEW), *arguments], capture_output=True, text=True, check=False)

    def test_prints_only_the_resolved_path(self):
        with tempfile.TemporaryDirectory() as directory:
            body = Path(directory, "body.md")
            body.write_text("## Summary\n\nbody\n", encoding="utf-8")
            output = Path(directory, "page.html")
            result = self.run_preview(str(body), "--output", str(output))
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), str(output.resolve()))
            self.assertEqual(result.stderr, "")

    def test_rejects_no_open_flag(self):
        result = self.run_preview("--output", "unused.html", "--no-open")
        self.assertEqual(result.returncode, 2)


if __name__ == "__main__":
    unittest.main()

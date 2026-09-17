import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


class RefreshTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name).resolve()
        self.repo = self.base / 'repo with spaces'
        self.repo.mkdir()
        shutil.copy2(Path(__file__).resolve().parents[1] / 'refresh', self.repo)
        self.source = self.repo / 'skills/example'
        self.source.mkdir(parents=True)
        (self.source / 'SKILL.md').write_text('old instructions')
        self.env = dict(os.environ, HOME=str(self.base / 'home'),
                        CODEX_HOME=str(self.base / 'codex'),
                        CLAUDE_CONFIG_DIR=str(self.base / 'claude'))
        self.git('init')
        self.git('add', '.')
        self.git('-c', 'user.name=Test', '-c', 'user.email=test@example.test',
                 '-c', 'core.hooksPath=/dev/null', 'commit', '-m', 'initial')
        (self.source / 'SKILL.md').write_text('current instructions')
        (self.source / 'asset.txt').write_text('current asset')
        self.target = self.base / 'home/.agents/skills/example'
        self.target.mkdir(parents=True)
        (self.target / 'SKILL.md').write_text('old instructions')
        (self.target / 'local.txt').write_text('preserve this')

    def git(self, *args):
        subprocess.run(['git', '-C', str(self.repo), *args], check=True,
                       capture_output=True, env=self.env)

    def run_refresh(self, *args):
        return subprocess.run([str(self.repo / 'refresh'), *args],
                              capture_output=True, text=True, env=self.env)

    def test_historical_copy_backup_and_repeat(self):
        result = self.run_refresh()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.target.resolve(), self.source)
        self.assertEqual((self.target / 'asset.txt').read_text(), 'current asset')
        backups = list((self.base / 'home/.agents/skill-backups').glob('*/example'))
        self.assertEqual(len(backups), 1)
        self.assertEqual((backups[0] / 'local.txt').read_text(), 'preserve this')
        self.assertEqual(self.run_refresh('--check').returncode, 0)
        self.assertEqual(self.run_refresh().returncode, 0)
        self.assertEqual(list((self.base / 'home/.agents/skill-backups').glob('*/example')), backups)
        self.assertFalse((self.base / 'codex/skills/example').exists())

    def test_check_is_read_only(self):
        self.assertEqual(self.run_refresh('--check').returncode, 1)
        self.assertFalse(self.target.is_symlink())
        self.assertFalse((self.base / 'home/.agents/skill-backups').exists())

    def test_unrecognised_copy_and_broken_link_are_preserved(self):
        (self.target / 'SKILL.md').write_text('someone else owns this')
        broken = self.base / 'codex/skills/example'
        broken.parent.mkdir(parents=True)
        broken.symlink_to(self.base / 'missing')
        self.assertEqual(self.run_refresh().returncode, 1)
        self.assertEqual((self.target / 'SKILL.md').read_text(), 'someone else owns this')
        self.assertTrue(broken.is_symlink())
        self.assertEqual(os.readlink(broken), str(self.base / 'missing'))

    def test_matching_copy_and_outdated_link_in_custom_roots(self):
        for root in ['codex', 'claude']:
            target = self.base / root / 'skills/example'
            target.parent.mkdir(parents=True)
            if root == 'codex':
                shutil.copytree(self.source, target)
            else:
                old = self.base / 'old-checkout/example'
                shutil.copytree(self.target, old)
                target.symlink_to(old, target_is_directory=True)
        result = self.run_refresh('example')
        self.assertEqual(result.returncode, 0, result.stderr)
        for root in ['codex', 'claude']:
            self.assertEqual((self.base / root / 'skills/example').resolve(), self.source)
        self.assertEqual(self.run_refresh('unknown').returncode, 2)


if __name__ == '__main__':
    unittest.main()

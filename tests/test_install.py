import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


class InstallTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name).resolve()
        self.repo = self.base / 'repo with spaces'
        self.repo.mkdir()
        shutil.copy2(Path(__file__).resolve().parents[1] / 'install', self.repo)
        for name in ['alpha', 'beta']:
            source = self.repo / 'skills' / name
            source.mkdir(parents=True)
            (source / 'SKILL.md').write_text(f'---\nname: {name}\ndescription: {name} skill\n---\n')
        self.claude = self.base / 'claude'
        self.links = self.base / 'home/.agents/skills'
        self.claude_links = self.claude / 'skills'
        self.settings = self.claude / 'settings.json'
        self.env = dict(os.environ, HOME=str(self.base / 'home'), CLAUDE_CONFIG_DIR=str(self.claude))

    def run_install(self, *args):
        return subprocess.run([str(self.repo / 'install'), *args], capture_output=True,
                              text=True, env=self.env, stdin=subprocess.DEVNULL)

    def overrides(self):
        if not self.settings.exists():
            return None
        return json.loads(self.settings.read_text()).get('skillOverrides')

    def states(self):
        out = self.run_install('--list').stdout
        found = {}
        for line in out.splitlines():
            for name in ['alpha', 'beta']:
                if f' {name} ' in line + ' ':
                    for state in ['user-invocable-only', 'name-only', 'off', 'on']:
                        if state in line:
                            found[name] = state
                            break
        return found

    def test_add_creates_link_and_remove_deletes_it(self):
        self.assertEqual(self.states(), {'alpha': 'off', 'beta': 'off'})
        result = self.run_install('--add', 'alpha')
        self.assertEqual(result.returncode, 0, result.stderr)
        for directory in [self.links, self.claude_links]:
            self.assertTrue((directory / 'alpha').is_symlink())
            self.assertEqual((directory / 'alpha').resolve(), self.repo / 'skills/alpha')
        self.assertFalse((self.links / 'beta').is_symlink())
        self.assertEqual(self.states(), {'alpha': 'on', 'beta': 'off'})
        self.assertIsNone(self.overrides())
        result = self.run_install('--remove', 'alpha')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse((self.links / 'alpha').is_symlink())
        self.assertFalse((self.claude_links / 'alpha').is_symlink())
        self.assertEqual(self.states(), {'alpha': 'off', 'beta': 'off'})

    def test_partial_states_keep_link_and_write_override(self):
        self.run_install('--set', 'alpha=name-only', '--set', 'beta=user-invocable-only')
        self.assertTrue((self.links / 'alpha').is_symlink())
        self.assertTrue((self.links / 'beta').is_symlink())
        self.assertTrue((self.claude_links / 'beta').is_symlink())
        self.assertEqual(self.overrides(), {'alpha': 'name-only', 'beta': 'user-invocable-only'})
        self.assertEqual(self.states(), {'alpha': 'name-only', 'beta': 'user-invocable-only'})
        self.run_install('--set', 'alpha=off', '--set', 'beta=on')
        self.assertFalse((self.links / 'alpha').is_symlink())
        self.assertFalse((self.claude_links / 'alpha').is_symlink())
        self.assertTrue((self.links / 'beta').is_symlink())
        self.assertTrue((self.claude_links / 'beta').is_symlink())
        self.assertIsNone(self.overrides())
        self.assertEqual(self.states(), {'alpha': 'off', 'beta': 'on'})

    def test_claude_only_link_does_not_control_state_and_is_removed(self):
        self.claude_links.mkdir(parents=True)
        link = self.claude_links / 'alpha'
        link.symlink_to(self.repo / 'skills/alpha', target_is_directory=True)
        self.assertEqual(self.states(), {'alpha': 'off', 'beta': 'off'})
        result = self.run_install('--remove', 'alpha')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(link.is_symlink())

    def test_default_claude_directory(self):
        self.env.pop('CLAUDE_CONFIG_DIR')
        link = self.base / 'home/.claude/skills/alpha'
        result = self.run_install('--add', 'alpha')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(link.is_symlink())
        self.assertEqual(link.resolve(), self.repo / 'skills/alpha')
        result = self.run_install('--remove', 'alpha')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(link.is_symlink())

    def test_real_claude_directory_is_preserved(self):
        directory = self.claude_links / 'alpha'
        directory.mkdir(parents=True)
        sentinel = directory / 'local.txt'
        sentinel.write_text('local skill')
        for args in [('--add', 'alpha'), ('--sync',), ('--remove', 'alpha')]:
            result = self.run_install(*args)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertFalse(directory.is_symlink())
            self.assertEqual(sentinel.read_text(), 'local skill')
            self.assertIn(str(directory), result.stdout)
        self.assertFalse((self.links / 'alpha').is_symlink())

    def test_sync_adds_missing_claude_link_for_enabled_skill_only(self):
        self.links.mkdir(parents=True)
        (self.links / 'alpha').symlink_to(self.repo / 'skills/alpha')
        result = self.run_install('--sync')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue((self.claude_links / 'alpha').is_symlink())
        self.assertEqual((self.claude_links / 'alpha').resolve(), self.repo / 'skills/alpha')
        self.assertFalse((self.claude_links / 'beta').exists())

    def test_real_directory_is_not_deleted(self):
        (self.links / 'alpha').mkdir(parents=True)
        result = self.run_install('--remove', 'alpha')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue((self.links / 'alpha').is_dir())
        self.assertIn('not deleting', result.stdout)

    def test_sync_repairs_stale_links_and_prunes_dangling_ones(self):
        self.links.mkdir(parents=True)
        old = self.base / 'old-checkout/skills'
        (old / 'alpha').mkdir(parents=True)
        (self.links / 'alpha').symlink_to(old / 'alpha', target_is_directory=True)
        (self.links / 'gone').symlink_to(self.repo / 'skills/gone', target_is_directory=True)
        (self.links / 'other').symlink_to(self.base / 'elsewhere', target_is_directory=True)
        self.claude_links.mkdir(parents=True)
        (self.claude_links / 'alpha').symlink_to(old / 'alpha')
        (self.claude_links / 'gone').symlink_to(self.repo / 'skills/gone')
        (self.claude_links / 'other').symlink_to(self.base / 'elsewhere')
        result = self.run_install('--sync')
        self.assertEqual(result.returncode, 0, result.stderr)
        for directory in [self.links, self.claude_links]:
            self.assertTrue((directory / 'alpha').is_symlink())
            self.assertEqual((directory / 'alpha').resolve(), self.repo / 'skills/alpha')
        self.assertFalse((self.links / 'gone').is_symlink())
        self.assertFalse((self.claude_links / 'gone').is_symlink())
        self.assertTrue((self.claude_links / 'other').is_symlink())
        self.assertFalse((self.claude_links / 'beta').is_symlink())
        self.assertTrue((self.links / 'other').is_symlink())
        self.assertFalse((self.links / 'beta').is_symlink())

    def test_bad_input_is_rejected(self):
        self.assertEqual(self.run_install('--add', 'nope').returncode, 1)
        self.assertEqual(self.run_install('--set', 'alpha=maybe').returncode, 1)
        self.assertEqual(self.run_install('--set', 'alpha').returncode, 1)
        self.assertEqual(self.run_install().returncode, 1)
        self.assertFalse(self.links.exists())


if __name__ == '__main__':
    unittest.main()

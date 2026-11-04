import json
import os
import subprocess
import sys
from pathlib import Path

import pytest

PLAN_SCRIPT = Path(__file__).resolve().parents[1] / "plan"
RENDER_SCRIPT = Path(__file__).resolve().parents[1] / "render"


def git(repo, *arguments):
    subprocess.run(["git", *arguments], cwd=repo, check=True, capture_output=True)


@pytest.fixture
def repo(tmp_path):
    repository = tmp_path / "repo"
    repository.mkdir()
    git(repository, "init", "-q", "-b", "main")
    git(repository, "config", "user.email", "test@example.com")
    git(repository, "config", "user.name", "Test")
    (repository / "a.go").write_text("package a\n")
    git(repository, "add", ".")
    git(repository, "commit", "-q", "-m", "feat: initial")
    return repository


def run_plan(repo, tmp_path, *arguments, stdin=""):
    environment = {**os.environ, "SKILLS_TMPDIR": str(tmp_path / "out")}
    return subprocess.run(
        [sys.executable, str(PLAN_SCRIPT), *arguments],
        cwd=repo,
        input=stdin,
        capture_output=True,
        text=True,
        env=environment,
    )


def dirty(repo):
    (repo / "a.go").write_text("package a\n\nvar changed = 1\n")
    (repo / "notes.txt").write_text("untracked body\n")


def test_gather_reports_clean_tree(repo, tmp_path):
    result = run_plan(repo, tmp_path, "gather")
    assert result.returncode == 0
    assert "nothing to commit" in result.stdout


def test_gather_includes_diff_and_untracked_contents(repo, tmp_path):
    dirty(repo)
    result = run_plan(repo, tmp_path, "gather")
    assert result.returncode == 0
    assert "+var changed = 1" in result.stdout
    assert "untracked body" in result.stdout


def test_build_writes_plan_and_page(repo, tmp_path):
    dirty(repo)
    document = {
        "commits": [
            {"header": "feat(cli)!: add thing", "why": "because", "paths": ["a.go", "notes.txt"]}
        ]
    }
    result = run_plan(repo, tmp_path, "build", stdin=json.dumps(document))
    assert result.returncode == 0, result.stderr
    page = Path(result.stdout.strip())
    assert page.name == "index.html" and page.exists()
    plan = json.loads((page.parent / "plan.json").read_text())
    commit = plan["commits"][0]
    assert (commit["type"], commit["scope"], commit["breaking"], commit["subject"]) == (
        "feat",
        "cli",
        True,
        "add thing",
    )


def test_build_rejects_invalid_header(repo, tmp_path):
    dirty(repo)
    document = {"commits": [{"header": "nonsense", "why": "x", "paths": ["a.go"]}]}
    result = run_plan(repo, tmp_path, "build", stdin=json.dumps(document))
    assert result.returncode != 0
    assert "header must look like" in result.stderr


def test_build_rejects_no_open_flag(repo, tmp_path):
    dirty(repo)
    result = run_plan(repo, tmp_path, "build", "--no-open", stdin=json.dumps({"commits": []}))
    assert result.returncode == 2


def test_render_rejects_no_open_flag():
    result = subprocess.run([sys.executable, str(RENDER_SCRIPT), "--no-open", "plan.json"], capture_output=True, text=True)
    assert result.returncode == 2

"""Build local-only Graphify profiles from filtered source snapshots."""

import argparse
import fnmatch
import json
import os
import shutil
import subprocess
import tempfile
from pathlib import Path
from typing import Any, Dict, Protocol

from tools.graphify.validate import make_manifest, provenance_stamp, validate_directory


def load_profile(config_path: Path, name: str) -> Dict[str, Any]:
    """Return the named profile from a Graphify profiles JSON file."""
    config = json.loads(config_path.read_text(encoding="utf-8"))
    try:
        return {"name": name, "version": config["version"], **config["profiles"][name]}
    except KeyError as error:
        raise ValueError("unknown graph profile: {}".format(name)) from error


def _matches(pattern: str, relative_path: str) -> bool:
    """Match repository-relative paths, including root files for ** patterns."""
    return fnmatch.fnmatchcase(relative_path, pattern) or (
        pattern.startswith("**/")
        and fnmatch.fnmatchcase(relative_path, pattern[3:])
    )


def profile_includes(profile: Dict[str, Any], relative_path: str) -> bool:
    """Whether a repository-relative path belongs in a profile's graph."""
    normalized = relative_path.replace("\\", "/").lstrip("/")
    if normalized in profile["include_files"]:
        return True
    if any(_matches(pattern, normalized) for pattern in profile["exclude_globs"]):
        return False
    return any(
        normalized == root or normalized.startswith(root + "/")
        for root in profile["include_roots"]
    )


def _is_within(path: Path, root: Path) -> bool:
    """Whether a resolved path is contained by a resolved root."""
    try:
        path.relative_to(root)
    except ValueError:
        return False
    return True


def materialize_profile(
    repo_root: Path, snapshot: Path, profile: Dict[str, Any]
) -> Dict[str, int]:
    """Copy profile files into a repository-relative snapshot."""
    repo_root = repo_root.resolve()
    snapshot = snapshot.absolute()
    try:
        repo_root.relative_to(snapshot)
    except ValueError:
        pass
    else:
        raise ValueError("snapshot must be below repository root")
    if not _is_within(snapshot.parent.resolve(), repo_root):
        raise ValueError("snapshot must resolve below repository root")
    if snapshot.is_symlink() or snapshot.is_file():
        snapshot.unlink()
    if not _is_within(snapshot.resolve(), repo_root):
        raise ValueError("snapshot must resolve below repository root")
    elif snapshot.exists():
        shutil.rmtree(snapshot)
    counts = {"included": 0, "excluded": 0}
    for source in repo_root.rglob("*"):
        if source.is_symlink() or not source.is_file() or ".git" in source.parts:
            continue
        try:
            source.relative_to(snapshot)
            continue
        except ValueError:
            pass
        relative_path = source.relative_to(repo_root).as_posix()
        if not profile_includes(profile, relative_path):
            counts["excluded"] += 1
            continue
        target = snapshot / relative_path
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)
        counts["included"] += 1
    return counts


class Runner(Protocol):
    """Produces Graphify artifacts for a materialized source snapshot."""

    def extract(self, source: Path, artifacts: Path) -> None:
        """Write graph.json, report, HTML, and wiki into artifacts."""

    def version(self) -> str:
        """Return the Graphify version used for the extraction."""


class GraphifyRunner:
    """Run the installed Graphify CLI without writing into the repository."""

    def __init__(self, command: str = "graphify") -> None:
        self.command = command

    def extract(self, source: Path, artifacts: Path) -> None:
        """Create a code-only graph plus HTML and wiki exports."""
        output_root = artifacts.parent
        subprocess.run(
            [
                self.command,
                "extract",
                str(source),
                "--out",
                str(output_root),
                "--code-only",
            ],
            check=True,
        )
        if not artifacts.is_dir():
            raise ValueError("Graphify CLI did not create graphify-out")
        graph = artifacts / "graph.json"
        subprocess.run(
            [
                self.command,
                "cluster-only",
                str(output_root),
                "--graph",
                str(graph),
            ],
            check=True,
        )
        subprocess.run(
            [self.command, "export", "wiki", "--graph", str(graph)], check=True
        )

    def version(self) -> str:
        """Return the installed Graphify version for artifact provenance."""
        result = subprocess.run(
            [self.command, "version"],
            check=True,
            stdout=subprocess.PIPE,
            text=True,
        )
        return result.stdout.strip()


def _git_sha(repo_root: Path) -> str:
    """Return the checkout SHA; use an explicit local marker in temporary tests."""
    result = subprocess.run(
        ["git", "-C", str(repo_root), "rev-parse", "HEAD"],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
    )
    return result.stdout.strip() if result.returncode == 0 else "local-uncommitted"


def _stamp(path: Path, manifest: Dict[str, Any]) -> None:
    """Add a machine-searchable build ID to a human-readable artifact."""
    path.write_text(
        path.read_text(encoding="utf-8")
        + "\n{}\n".format(provenance_stamp(manifest)),
        encoding="utf-8",
    )


def _write_manifest_and_stamps(artifacts: Path, manifest: Dict[str, Any]) -> None:
    """Write shared provenance before validating a staged artifact set."""
    (artifacts / "manifest.json").write_text(
        json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )
    _stamp(artifacts / "GRAPH_REPORT.md", manifest)
    _stamp(artifacts / "wiki/index.md", manifest)


def _validate_destination(repo_root: Path, destination: Path, profile: Dict[str, Any]) -> Path:
    """Require the local profile destination to stay below a real graphify-out dir."""
    output_root = repo_root / "graphify-out"
    expected = output_root / profile["name"]
    if destination.absolute() != expected.absolute():
        raise ValueError("destination must be the local graphify-out profile path")
    if output_root.is_symlink():
        raise ValueError("graphify-out must not be a symlink")
    if expected.is_symlink():
        raise ValueError("graphify-out profile destination must not be a symlink")
    if output_root.exists() and not output_root.is_dir():
        raise ValueError("graphify-out must be a directory")
    if not _is_within(output_root.resolve(), repo_root):
        raise ValueError("graphify-out must resolve below repository root")
    return expected


def _publish(artifacts: Path, destination: Path) -> None:
    """Swap validated artifacts into place, restoring the old profile on error."""
    previous = destination.parent / ".{}.previous-{}".format(
        destination.name, os.urandom(8).hex()
    )
    had_destination = destination.exists() or destination.is_symlink()
    if had_destination:
        destination.replace(previous)
    try:
        artifacts.replace(destination)
    except Exception:
        if had_destination and previous.exists():
            previous.replace(destination)
        raise
    if previous.exists() or previous.is_symlink():
        if previous.is_dir() and not previous.is_symlink():
            shutil.rmtree(previous)
        else:
            previous.unlink()


def refresh_profile(
    repo_root: Path, destination: Path, profile: Dict[str, Any], runner: Runner
) -> Dict[str, Any]:
    """Atomically refresh one local-only Graphify knowledge profile."""
    repo_root = repo_root.resolve()
    destination = _validate_destination(repo_root, destination, profile)
    destination.parent.mkdir(parents=True, exist_ok=True)
    staging = Path(
        tempfile.mkdtemp(prefix=".{}-".format(profile["name"]), dir=destination.parent)
    )
    try:
        snapshot = staging / "source"
        file_counts = materialize_profile(repo_root, snapshot, profile)
        artifacts = staging / "graphify-out"
        runner.extract(snapshot, artifacts)
        graph = json.loads((artifacts / "graph.json").read_text(encoding="utf-8"))
        manifest = make_manifest(
            profile["name"],
            profile["version"],
            runner.version(),
            _git_sha(repo_root),
            file_counts,
            graph,
        )
        _write_manifest_and_stamps(artifacts, manifest)
        validate_directory(artifacts, manifest, profile)
        _publish(artifacts, destination)
        return manifest
    finally:
        shutil.rmtree(staging, ignore_errors=True)


def main() -> None:
    """Refresh the requested profile beneath this checkout's ignored graph output."""
    parser = argparse.ArgumentParser(description="Atomically refresh local Graphify")
    parser.add_argument("--profile", default="coding")
    parser.add_argument("--repo-root", type=Path, default=Path.cwd())
    arguments = parser.parse_args()
    repo_root = arguments.repo_root.resolve()
    profile = load_profile(Path(__file__).with_name("profiles.json"), arguments.profile)
    destination = repo_root / "graphify-out" / profile["name"]
    manifest = refresh_profile(repo_root, destination, profile, GraphifyRunner())
    print("Graphify profile refreshed: {}".format(manifest["build_id"]))


if __name__ == "__main__":
    main()

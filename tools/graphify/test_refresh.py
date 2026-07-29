"""Tests for local Graphify artifact policy and knowledge profiles."""

import subprocess
import sys
import tempfile
import unittest
import json
import shutil
from pathlib import Path

from tools.graphify import refresh
from tools.graphify.refresh import (
    GraphifyRunner,
    load_profile,
    materialize_profile,
    profile_includes,
)
from tools.graphify.validate import (
    make_manifest,
    provenance_stamp,
    validate_build,
    validate_profile_health,
)



class IgnorePolicyTests(unittest.TestCase):
    def test_coding_graph_artifact_is_ignored(self) -> None:
        repository_root = Path(__file__).resolve().parents[2]
        result = subprocess.run(
            ["git", "check-ignore", "-q", "graphify-out/coding/graph.json"],
            cwd=repository_root,
            check=False,
        )

        self.assertEqual(result.returncode, 0)

    def test_graphify_python_caches_are_ignored(self) -> None:
        repository_root = Path(__file__).resolve().parents[2]
        result = subprocess.run(
            ["git", "check-ignore", "-q", "tools/graphify/__pycache__/refresh.cpython-313.pyc"],
            cwd=repository_root,
            check=False,
        )

        self.assertEqual(result.returncode, 0)


class ProfileTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repository_root = Path(__file__).resolve().parents[2]
        cls.config_path = cls.repository_root / "tools/graphify/profiles.json"

    def test_coding_profile_keeps_feature_implementation_and_filters_generated_noise(self) -> None:
        profile = load_profile(self.config_path, "coding")

        self.assertEqual(profile["name"], "coding")
        self.assertEqual(profile["version"], 1)
        self.assertEqual(profile["max_generated_node_ratio"], 0.05)
        self.assertEqual(profile["max_dangling_edge_ratio"], 0.05)
        self.assertTrue(profile_includes(profile, "app/app.go"))
        self.assertTrue(profile_includes(profile, "x/tax/keeper/keeper.go"))
        self.assertFalse(profile_includes(profile, "x/tax/types/tx.pb.go"))
        self.assertFalse(profile_includes(profile, "client/docs/swagger-ui/swagger-ui-bundle.js"))

    def test_domain_api_profile_keeps_specs_and_schema_but_filters_generated_go(self) -> None:
        profile = load_profile(self.config_path, "domain-api")

        self.assertEqual(profile["name"], "domain-api")
        self.assertEqual(profile["version"], 1)
        self.assertEqual(profile["max_generated_node_ratio"], 0.05)
        self.assertEqual(profile["max_dangling_edge_ratio"], 0.05)
        self.assertTrue(profile_includes(profile, "x/oracle/spec/03_end_block.md"))
        self.assertTrue(profile_includes(profile, "proto/terra/oracle/v1beta1/query.proto"))
        self.assertTrue(
            profile_includes(profile, "client/docs/swagger-ui/swagger.yaml")
        )
        self.assertFalse(profile_includes(profile, "x/oracle/types/query.pb.go"))
        self.assertFalse(
            profile_includes(profile, "client/docs/swagger-ui/swagger-ui-bundle.js")
        )

    def test_profiles_use_the_required_schema(self) -> None:
        coding = load_profile(self.config_path, "coding")
        domain_api = load_profile(self.config_path, "domain-api")

        required_fields = {
            "include_roots",
            "include_files",
            "exclude_globs",
            "max_generated_node_ratio",
            "max_dangling_edge_ratio",
        }
        for profile in (coding, domain_api):
            self.assertTrue(required_fields.issubset(profile))

    def test_unknown_profile_has_a_clear_error(self) -> None:
        with self.assertRaisesRegex(
            ValueError, "^unknown graph profile: missing$"
        ):
            load_profile(self.config_path, "missing")


class SnapshotTests(unittest.TestCase):
    @staticmethod
    def coding_profile():
        return load_profile(
            Path(__file__).resolve().parents[2] / "tools/graphify/profiles.json",
            "coding",
        )

    def test_materialize_profile_copies_allowed_files_and_skips_generated_files(self) -> None:
        profile = self.coding_profile()
        with tempfile.TemporaryDirectory() as temporary_directory:
            repository_root = Path(temporary_directory) / "repo"
            snapshot = repository_root / "graphify-out/coding/snapshot"
            allowed_code = repository_root / "app/app.go"
            generated_code = repository_root / "x/tax/types/tx.pb.go"
            allowed_spec = repository_root / "x/tax/spec/01_concepts.md"
            for path in (allowed_code, generated_code, allowed_spec):
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(path.name, encoding="utf-8")
            (repository_root / ".git").mkdir(parents=True)
            (repository_root / ".git/config").write_text("ignored", encoding="utf-8")

            result = materialize_profile(repository_root, snapshot, profile)

            self.assertEqual(result, {"included": 2, "excluded": 1})
            self.assertEqual((snapshot / "app/app.go").read_text(encoding="utf-8"), "app.go")
            self.assertEqual(
                (snapshot / "x/tax/spec/01_concepts.md").read_text(encoding="utf-8"),
                "01_concepts.md",
            )
            self.assertFalse((snapshot / "x/tax/types/tx.pb.go").exists())
            self.assertFalse((snapshot / "graphify-out/coding/snapshot").exists())

    def test_materialize_profile_replaces_a_stale_snapshot(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            repository_root = Path(temporary_directory) / "repo"
            snapshot = repository_root / "snapshot"
            (repository_root / "app/app.go").parent.mkdir(parents=True)
            (repository_root / "app/app.go").write_text("current", encoding="utf-8")
            (snapshot / "app/stale.go").parent.mkdir(parents=True)
            (snapshot / "app/stale.go").write_text("stale", encoding="utf-8")

            result = materialize_profile(repository_root, snapshot, self.coding_profile())

            self.assertEqual(result, {"included": 1, "excluded": 0})
            self.assertFalse((snapshot / "app/stale.go").exists())
            self.assertEqual((snapshot / "app/app.go").read_text(encoding="utf-8"), "current")

    def test_materialize_profile_skips_symlink_sources(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            repository_root = Path(temporary_directory) / "repo"
            source = repository_root / "app/app.go"
            linked_source = repository_root / "app/linked.go"
            source.parent.mkdir(parents=True)
            source.write_text("package app", encoding="utf-8")
            linked_source.symlink_to(source)

            result = materialize_profile(
                repository_root, repository_root / "snapshot", self.coding_profile()
            )

            self.assertEqual(result, {"included": 1, "excluded": 0})
            self.assertFalse((repository_root / "snapshot/app/linked.go").exists())

    def test_materialize_profile_unlinks_a_snapshot_symlink_without_touching_its_target(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_root = Path(temporary_directory)
            repository_root = temporary_root / "repo"
            external_target = temporary_root / "external"
            snapshot = repository_root / "snapshot"
            (repository_root / "app").mkdir(parents=True)
            (repository_root / "app/app.go").write_text("package app", encoding="utf-8")
            external_target.mkdir()
            sentinel = external_target / "sentinel.txt"
            sentinel.write_text("must survive", encoding="utf-8")
            snapshot.symlink_to(external_target, target_is_directory=True)

            materialize_profile(repository_root, snapshot, self.coding_profile())

            self.assertEqual(sentinel.read_text(encoding="utf-8"), "must survive")
            self.assertFalse(snapshot.is_symlink())
            self.assertEqual((snapshot / "app/app.go").read_text(encoding="utf-8"), "package app")

    def test_materialize_profile_rejects_snapshot_below_a_parent_symlink(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_root = Path(temporary_directory)
            repository_root = temporary_root / "repo"
            external_target = temporary_root / "external"
            snapshot = repository_root / "graphify-out/coding"
            (repository_root / "app").mkdir(parents=True)
            (repository_root / "app/app.go").write_text("package app", encoding="utf-8")
            external_target.mkdir()
            sentinel = external_target / "sentinel.txt"
            sentinel.write_text("must survive", encoding="utf-8")
            (repository_root / "graphify-out").symlink_to(
                external_target, target_is_directory=True
            )

            with self.assertRaisesRegex(ValueError, "snapshot.*repository root"):
                materialize_profile(repository_root, snapshot, self.coding_profile())

            self.assertEqual(sentinel.read_text(encoding="utf-8"), "must survive")

    def test_materialize_profile_rejects_parent_symlink_before_unlinking_external_file(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_root = Path(temporary_directory)
            repository_root = temporary_root / "repo"
            external_target = temporary_root / "external"
            snapshot = repository_root / "graphify-out/coding"
            (repository_root / "app").mkdir(parents=True)
            (repository_root / "app/app.go").write_text("package app", encoding="utf-8")
            external_target.mkdir()
            sentinel = external_target / "coding"
            sentinel.write_text("must survive", encoding="utf-8")
            (repository_root / "graphify-out").symlink_to(
                external_target, target_is_directory=True
            )

            with self.assertRaisesRegex(ValueError, "snapshot.*repository root"):
                materialize_profile(repository_root, snapshot, self.coding_profile())

            self.assertEqual(sentinel.read_text(encoding="utf-8"), "must survive")

    def test_materialize_profile_rejects_snapshot_at_or_above_repository_root(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            repository_root = Path(temporary_directory) / "repo"
            repository_root.mkdir()
            for invalid_snapshot in (repository_root, repository_root.parent):
                with self.subTest(snapshot=invalid_snapshot):
                    with self.assertRaisesRegex(ValueError, "snapshot.*below repository root"):
                        materialize_profile(
                            repository_root, invalid_snapshot, self.coding_profile()
                        )


class ValidationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.graph = {
            "nodes": [{"id": "app"}, {"id": "keeper"}],
            "edges": [{"source": "app", "target": "keeper"}],
        }
        self.manifest = make_manifest(
            profile="coding",
            profile_version=1,
            graphify_version="graphify 0.9.17",
            git_sha="0123456789abcdef0123456789abcdef01234567",
            file_counts={"included": 12, "excluded": 3},
            graph=self.graph,
            timestamp="2026-07-29T00:00:00+00:00",
        )

    def test_make_manifest_records_canonical_graph_provenance(self) -> None:
        reordered_graph = {
            "edges": [{"target": "keeper", "source": "app"}],
            "nodes": [{"id": "app"}, {"id": "keeper"}],
        }

        reordered_manifest = make_manifest(
            profile="coding",
            profile_version=1,
            graphify_version="graphify 0.9.17",
            git_sha="0123456789abcdef0123456789abcdef01234567",
            file_counts={"included": 12, "excluded": 3},
            graph=reordered_graph,
            timestamp="2026-07-29T00:00:00+00:00",
        )

        self.assertEqual(
            self.manifest["build_id"],
            "coding-0123456789ab-" + self.manifest["graph_sha256"][:12],
        )
        self.assertEqual(self.manifest["profile"], "coding")
        self.assertEqual(self.manifest["profile_version"], 1)
        self.assertEqual(self.manifest["graphify_version"], "graphify 0.9.17")
        self.assertEqual(self.manifest["timestamp"], "2026-07-29T00:00:00+00:00")
        self.assertEqual(
            self.manifest["git_sha"], "0123456789abcdef0123456789abcdef01234567"
        )
        self.assertEqual(self.manifest["node_count"], 2)
        self.assertEqual(self.manifest["edge_count"], 1)
        self.assertEqual(self.manifest["file_counts"], {"included": 12, "excluded": 3})
        self.assertEqual(self.manifest["graph_sha256"], reordered_manifest["graph_sha256"])

    def test_validate_build_accepts_matching_graph_and_report(self) -> None:
        result = validate_build(
            self.manifest,
            self.graph,
            "# Graph report\n\n" + provenance_stamp(self.manifest),
        )

        self.assertIsNone(result)

    def test_validate_build_rejects_node_count_mismatch(self) -> None:
        mismatched_graph = {"nodes": [{"id": "app"}], "edges": []}

        with self.assertRaisesRegex(ValueError, "node count"):
            validate_build(
                self.manifest,
                mismatched_graph,
                "Build: " + self.manifest["build_id"],
            )

    def test_validate_build_rejects_edge_count_mismatch(self) -> None:
        mismatched_graph = {
            "nodes": [{"id": "app"}, {"id": "keeper"}],
            "edges": [],
        }

        with self.assertRaisesRegex(ValueError, "edge count"):
            validate_build(
                self.manifest,
                mismatched_graph,
                "Build: " + self.manifest["build_id"],
            )

    def test_validate_build_rejects_changed_graph_content_with_matching_counts(self) -> None:
        changed_graph = {
            "nodes": [{"id": "app"}, {"id": "different-keeper"}],
            "edges": [{"source": "app", "target": "different-keeper"}],
        }

        with self.assertRaisesRegex(ValueError, "graph SHA-256"):
            validate_build(
                self.manifest,
                changed_graph,
                "Build: " + self.manifest["build_id"],
            )

    def test_validate_build_rejects_report_missing_build_id(self) -> None:
        with self.assertRaisesRegex(ValueError, "provenance stamp"):
            validate_build(self.manifest, self.graph, "# Graph report")

    def test_profile_health_rejects_empty_graph_missing_endpoints_and_self_loops(self) -> None:
        profile = {
            "max_generated_node_ratio": 0.05,
            "max_dangling_edge_ratio": 0.05,
        }
        cases = (
            ({"nodes": [], "edges": []}, "empty"),
            (
                {
                    "nodes": [{"id": "app"}],
                    "edges": [{"source": "missing-a", "target": "missing-b"}],
                },
                "local missing endpoint",
            ),
            (
                {
                    "nodes": [{"id": "app"}],
                    "edges": [{"source": "app", "target": "app"}],
                },
                "self-loop",
            ),
        )
        for graph, message in cases:
            with self.subTest(message=message):
                with self.assertRaisesRegex(ValueError, message):
                    validate_profile_health(graph, profile)

    def test_profile_health_excludes_external_imports_from_dangling_rate(self) -> None:
        profile = {
            "max_generated_node_ratio": 0.05,
            "max_dangling_edge_ratio": 0.05,
        }
        graph = {
            "nodes": [{"id": "app"}],
            "edges": [
                {
                    "source": "app",
                    "target": "github.com/cosmos/cosmos-sdk/types",
                    "relation": "imports",
                }
            ],
        }

        self.assertIsNone(validate_profile_health(graph, profile))

    def test_profile_health_rejects_one_missing_local_endpoint(self) -> None:
        profile = {
            "max_generated_node_ratio": 0.05,
            "max_dangling_edge_ratio": 1.0,
        }
        graph = {
            "nodes": [{"id": "app"}],
            "edges": [{"source": "app", "target": "missing", "relation": "calls"}],
        }

        with self.assertRaisesRegex(ValueError, "local missing endpoint"):
            validate_profile_health(graph, profile)

    def test_manifest_records_dangling_rate_without_external_imports(self) -> None:
        graph = {
            "nodes": [{"id": "app"}, {"id": "keeper"}],
            "edges": [
                {"source": "app", "target": "keeper", "relation": "calls"},
                {
                    "source": "app",
                    "target": "github.com/cosmos/cosmos-sdk/types",
                    "relation": "imports",
                },
            ],
        }

        manifest = make_manifest(
            profile="coding",
            profile_version=1,
            graphify_version="graphify test",
            git_sha="0123456789abcdef0123456789abcdef01234567",
            file_counts={"included": 2, "excluded": 0},
            graph=graph,
            timestamp="2026-07-29T00:00:00+00:00",
        )

        self.assertEqual(manifest["dangling_edge_rate"], 0.0)
        self.assertIn("dangling_edge_rate=0.000000", provenance_stamp(manifest))

    def test_validate_module_reports_build_id_for_valid_profile_directory(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            artifacts = Path(temporary_directory) / "coding"
            artifacts.mkdir()
            graph = {"nodes": [{"id": "app"}], "edges": []}
            manifest = make_manifest(
                profile="coding",
                profile_version=1,
                graphify_version="graphify test",
                git_sha="0123456789abcdef0123456789abcdef01234567",
                file_counts={"included": 1, "excluded": 0},
                graph=graph,
                timestamp="2026-07-29T00:00:00+00:00",
            )
            (artifacts / "graph.json").write_text(json.dumps(graph), encoding="utf-8")
            (artifacts / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
            (artifacts / "GRAPH_REPORT.md").write_text(
                "# Report\n" + provenance_stamp(manifest), encoding="utf-8"
            )
            (artifacts / "graph.html").write_text("<html></html>", encoding="utf-8")
            (artifacts / "wiki").mkdir()
            (artifacts / "wiki/index.md").write_text(
                "# Wiki\n" + provenance_stamp(manifest), encoding="utf-8"
            )

            result = subprocess.run(
                [
                    sys.executable,
                    "-m",
                    "tools.graphify.validate",
                    str(artifacts),
                    "--profile",
                    "coding",
                    "--profiles",
                    str(Path(__file__).with_name("profiles.json")),
                ],
                cwd=Path(__file__).resolve().parents[2],
                check=False,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("Graphify profile valid: " + manifest["build_id"], result.stdout)

    def test_validate_module_fails_for_invalid_profile_directory(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            artifacts = Path(temporary_directory) / "coding"
            artifacts.mkdir()

            result = subprocess.run(
                [
                    sys.executable,
                    "-m",
                    "tools.graphify.validate",
                    str(artifacts),
                    "--profile",
                    "coding",
                    "--profiles",
                    str(Path(__file__).with_name("profiles.json")),
                ],
                cwd=Path(__file__).resolve().parents[2],
                check=False,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("Graphify profile invalid:", result.stderr)


class FakeRunner:
    def __init__(self, invalid: bool = False) -> None:
        self.invalid = invalid
        self.sources = []

    def extract(self, source: Path, artifacts: Path) -> None:
        self.sources.append(source)
        artifacts.mkdir(parents=True)
        graph = {
            "nodes": [{"id": "app", "source_file": "app/app.go"}],
            "edges": (
                [{"source": "app", "target": "missing"}]
                if self.invalid
                else []
            ),
        }
        (artifacts / "graph.json").write_text(
            json.dumps(graph), encoding="utf-8"
        )
        (artifacts / "GRAPH_REPORT.md").write_text("# Graph report\n", encoding="utf-8")
        (artifacts / "graph.html").write_text("<html></html>\n", encoding="utf-8")
        wiki = artifacts / "wiki"
        wiki.mkdir()
        (wiki / "index.md").write_text("# Wiki\n", encoding="utf-8")

    def version(self) -> str:
        return "graphify test"


class AtomicRefreshTests(unittest.TestCase):
    def setUp(self) -> None:
        self.profile = load_profile(
            Path(__file__).resolve().parents[2] / "tools/graphify/profiles.json",
            "coding",
        )

    @staticmethod
    def write(path: Path, contents: str) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(contents, encoding="utf-8")

    def test_refresh_replaces_profile_only_after_validation(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            repository_root = Path(temporary_directory) / "repo"
            self.write(repository_root / "app/app.go", "package app\n")
            (repository_root / ".git").mkdir(parents=True)
            destination = repository_root / "graphify-out/coding"
            self.write(destination / "manifest.json", '{"build_id":"old"}\n')
            self.write(destination / "stale.txt", "old\n")
            runner = FakeRunner()

            refresh.refresh_profile(repository_root, destination, self.profile, runner)

            manifest = json.loads((destination / "manifest.json").read_text())
            self.assertTrue(manifest["build_id"].startswith("coding-"))
            self.assertEqual(manifest["profile_version"], 1)
            self.assertEqual(manifest["graphify_version"], "graphify test")
            self.assertIn("timestamp", manifest)
            self.assertFalse((destination / "stale.txt").exists())
            self.assertEqual(len(runner.sources), 1)
            self.assertTrue((destination / "graph.html").is_file())
            self.assertIn(manifest["build_id"], (destination / "GRAPH_REPORT.md").read_text())
            self.assertIn("node_count=1", (destination / "GRAPH_REPORT.md").read_text())
            self.assertIn("edge_count=0", (destination / "GRAPH_REPORT.md").read_text())
            self.assertIn(manifest["build_id"], (destination / "wiki/index.md").read_text())
            self.assertIn("node_count=1", (destination / "wiki/index.md").read_text())
            self.assertIn("edge_count=0", (destination / "wiki/index.md").read_text())

    def test_failed_validation_preserves_previous_profile(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            repository_root = Path(temporary_directory) / "repo"
            self.write(repository_root / "app/app.go", "package app\n")
            (repository_root / ".git").mkdir(parents=True)
            destination = repository_root / "graphify-out/coding"
            self.write(destination / "manifest.json", '{"build_id":"old"}\n')

            with self.assertRaisesRegex(ValueError, "local missing endpoint"):
                refresh.refresh_profile(
                    repository_root, destination, self.profile, FakeRunner(invalid=True)
                )

            self.assertEqual(
                (destination / "manifest.json").read_text(encoding="utf-8"),
                '{"build_id":"old"}\n',
            )

    def test_refresh_removes_staging_on_success_and_failure(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            repository_root = Path(temporary_directory) / "repo"
            self.write(repository_root / "app/app.go", "package app\n")
            (repository_root / ".git").mkdir(parents=True)
            parent = repository_root / "graphify-out"
            destination = parent / "coding"

            refresh.refresh_profile(repository_root, destination, self.profile, FakeRunner())
            with self.assertRaises(ValueError):
                refresh.refresh_profile(
                    repository_root, destination, self.profile, FakeRunner(invalid=True)
                )

            self.assertEqual(list(parent.glob(".coding-*")), [])

    def test_refresh_profile_api_is_available(self) -> None:
        self.assertTrue(callable(getattr(refresh, "refresh_profile", None)))

    def test_refresh_wrapper_uses_module_execution(self) -> None:
        wrapper = (
            Path(__file__).resolve().parents[2] / "scripts/graphify-refresh"
        ).read_text(encoding="utf-8")

        self.assertIn("python3 -m tools.graphify.refresh", wrapper)

    def test_refresh_wrapper_forwards_help_to_the_module(self) -> None:
        wrapper = (
            Path(__file__).resolve().parents[2] / "scripts/graphify-refresh"
        ).read_text(encoding="utf-8")

        self.assertIn('if [ "${1:-}" = "--help" ]', wrapper)

    def test_runner_does_not_disable_community_labels(self) -> None:
        runner_source = Path(refresh.__file__).read_text(encoding="utf-8")

        self.assertNotIn('"--no-label"', runner_source)

    def test_refresh_rejects_graphify_out_symlink_escape(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_root = Path(temporary_directory)
            repository_root = temporary_root / "repo"
            external = temporary_root / "external"
            self.write(repository_root / "app/app.go", "package app\n")
            (repository_root / ".git").mkdir(parents=True)
            external.mkdir()
            sentinel = external / "sentinel.txt"
            sentinel.write_text("must survive", encoding="utf-8")
            (repository_root / "graphify-out").symlink_to(
                external, target_is_directory=True
            )

            with self.assertRaisesRegex(ValueError, "graphify-out.*symlink"):
                refresh.refresh_profile(
                    repository_root,
                    repository_root / "graphify-out/coding",
                    self.profile,
                    FakeRunner(),
                )

            self.assertEqual(sentinel.read_text(encoding="utf-8"), "must survive")

    @unittest.skipUnless(shutil.which("graphify"), "graphify CLI is not installed")
    def test_graphify_runner_uses_real_cli_artifact_layout(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_root = Path(temporary_directory)
            source = temporary_root / "source"
            artifacts = temporary_root / "graphify-out"
            self.write(source / "demo.go", "package demo\nfunc Run() {}\n")

            GraphifyRunner().extract(source, artifacts)

            for relative_path in (
                "graph.json",
                "GRAPH_REPORT.md",
                "graph.html",
                "wiki/index.md",
            ):
                self.assertTrue((artifacts / relative_path).is_file(), relative_path)


class AgentGuidanceTests(unittest.TestCase):
    def test_guidance_requires_source_verification(self) -> None:
        guidance = (Path(__file__).resolve().parents[2] / "AGENTS.md").read_text(
            encoding="utf-8"
        )

        self.assertIn("scripts/graphify-refresh coding", guidance)
        self.assertIn("Source files are authoritative", guidance)
        self.assertIn("inferred", guidance)

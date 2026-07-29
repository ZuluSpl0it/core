"""Validate provenance shared by local Graphify build artifacts."""

import argparse
import hashlib
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List


def _canonical_graph(graph: Dict[str, Any]) -> bytes:
    """Serialize graph data deterministically for provenance hashing."""
    return json.dumps(
        graph, ensure_ascii=False, separators=(",", ":"), sort_keys=True
    ).encode("utf-8")


def graph_edges(graph: Dict[str, Any]) -> List[Dict[str, Any]]:
    """Return edges from Graphify's current or NetworkX export shape."""
    edges = graph.get("edges", graph.get("links", []))
    if not isinstance(edges, list):
        raise ValueError("graph edges must be a list")
    return edges


def make_manifest(
    profile: str,
    profile_version: int,
    graphify_version: str,
    git_sha: str,
    file_counts: Dict[str, int],
    graph: Dict[str, Any],
    timestamp: str = None,
) -> Dict[str, Any]:
    """Build reproducible provenance for one Graphify output snapshot."""
    graph_sha256 = hashlib.sha256(_canonical_graph(graph)).hexdigest()
    node_count = len(graph.get("nodes", []))
    edge_count = len(graph_edges(graph))
    health = graph_health_metrics(graph)
    return {
        "build_id": "{}-{}-{}".format(profile, git_sha[:12], graph_sha256[:12]),
        "profile": profile,
        "profile_version": profile_version,
        "graphify_version": graphify_version,
        "timestamp": timestamp or datetime.now(timezone.utc).isoformat(),
        "git_sha": git_sha,
        "graph_sha256": graph_sha256,
        "node_count": node_count,
        "edge_count": edge_count,
        "dangling_edge_rate": health["dangling_edge_rate"],
        "external_dependency_edge_count": health["external_dependency_edge_count"],
        "local_missing_endpoint_edge_count": health[
            "local_missing_endpoint_edge_count"
        ],
        "file_counts": file_counts,
    }


def provenance_stamp(manifest: Dict[str, Any]) -> str:
    """Return the exact provenance marker required in human-readable outputs."""
    return (
        "<!-- graphify-provenance: build_id={build_id} node_count={node_count} "
        "edge_count={edge_count} dangling_edge_rate={dangling_edge_rate:.6f} -->"
    ).format(**manifest)


def validate_build(manifest: Dict[str, Any], graph: Dict[str, Any], report: str) -> None:
    """Reject graph/report artifacts that do not match their manifest."""
    node_count = len(graph.get("nodes", []))
    if manifest.get("node_count") != node_count:
        raise ValueError("node count does not match manifest")

    edge_count = len(graph_edges(graph))
    if manifest.get("edge_count") != edge_count:
        raise ValueError("edge count does not match manifest")

    graph_sha256 = hashlib.sha256(_canonical_graph(graph)).hexdigest()
    if manifest.get("graph_sha256") != graph_sha256:
        raise ValueError("graph SHA-256 does not match manifest")

    if manifest.get("dangling_edge_rate") != graph_health_metrics(graph)[
        "dangling_edge_rate"
    ]:
        raise ValueError("dangling edge rate does not match manifest")

    if provenance_stamp(manifest) not in report:
        raise ValueError("report is missing provenance stamp")


def _generated_node(node: Dict[str, Any]) -> bool:
    source = str(node.get("source_file", "")).replace("\\", "/")
    return (
        source.endswith(".pb.go")
        or source.endswith(".pb.gw.go")
        or "/swagger-ui/" in source
        or source.startswith("client/docs/swagger-ui/")
        or bool(node.get("generated"))
    )


def _is_external_dependency(edge: Dict[str, Any], node_ids: set) -> bool:
    """Recognize unresolved imports as external dependencies, not bad local edges."""
    return (
        edge.get("source") in node_ids
        and edge.get("target") not in node_ids
        and (
            edge.get("external") is True
            or edge.get("relation") in ("import", "imports", "external_dep")
        )
    )


def graph_health_metrics(graph: Dict[str, Any]) -> Dict[str, Any]:
    """Measure local graph integrity without treating external imports as dangling."""
    nodes = graph.get("nodes", [])
    if not isinstance(nodes, list):
        raise ValueError("graph nodes must be a list")
    edges = graph_edges(graph)
    node_ids = {node.get("id") for node in nodes if isinstance(node, dict)}
    external_edges = [
        edge
        for edge in edges
        if isinstance(edge, dict) and _is_external_dependency(edge, node_ids)
    ]
    local_edges = [edge for edge in edges if edge not in external_edges]
    local_missing_edges = [
        edge
        for edge in local_edges
        if not isinstance(edge, dict)
        or edge.get("source") not in node_ids
        or edge.get("target") not in node_ids
    ]
    return {
        "external_dependency_edge_count": len(external_edges),
        "local_edge_count": len(local_edges),
        "local_missing_endpoint_edge_count": len(local_missing_edges),
        "dangling_edge_rate": (
            len(local_missing_edges) / len(local_edges) if local_edges else 0.0
        ),
    }


def validate_profile_health(graph: Dict[str, Any], profile: Dict[str, Any]) -> None:
    """Reject artifacts that exceed the profile's navigation-noise thresholds."""
    nodes = graph.get("nodes", [])
    if not isinstance(nodes, list):
        raise ValueError("graph nodes must be a list")
    if not nodes:
        raise ValueError("empty graph is not publishable")
    edges = graph_edges(graph)
    health = graph_health_metrics(graph)
    self_loops = [
        edge
        for edge in edges
        if isinstance(edge, dict)
        and edge.get("source") is not None
        and edge.get("source") == edge.get("target")
    ]
    if self_loops:
        raise ValueError("self-loop edges are not publishable")
    if health["local_missing_endpoint_edge_count"]:
        raise ValueError("local missing endpoint edges are not publishable")
    generated_count = sum(
        1 for node in nodes if isinstance(node, dict) and _generated_node(node)
    )
    generated_ratio = generated_count / len(nodes) if nodes else 0.0
    if generated_ratio > profile["max_generated_node_ratio"]:
        raise ValueError("generated node ratio exceeds profile threshold")
    if health["dangling_edge_rate"] > profile["max_dangling_edge_ratio"]:
        raise ValueError("dangling edge ratio exceeds profile threshold")


def validate_directory(
    artifacts: Path, manifest: Dict[str, Any], profile: Dict[str, Any]
) -> None:
    """Validate the complete local artifact set before it is published."""
    graph_path = artifacts / "graph.json"
    report_path = artifacts / "GRAPH_REPORT.md"
    html_path = artifacts / "graph.html"
    wiki_index = artifacts / "wiki/index.md"
    required = (graph_path, report_path, html_path, wiki_index, artifacts / "manifest.json")
    missing = [str(path.relative_to(artifacts)) for path in required if not path.is_file()]
    if missing:
        raise ValueError("missing graphify artifacts: {}".format(", ".join(missing)))
    graph = json.loads(graph_path.read_text(encoding="utf-8"))
    report = report_path.read_text(encoding="utf-8")
    wiki = wiki_index.read_text(encoding="utf-8")
    validate_build(manifest, graph, report)
    if provenance_stamp(manifest) not in wiki:
        raise ValueError("wiki index is missing provenance stamp")
    validate_profile_health(graph, profile)


def _load_profile(config_path: Path, name: str) -> Dict[str, Any]:
    """Load the profile needed by the standalone validation command."""
    config = json.loads(config_path.read_text(encoding="utf-8"))
    try:
        return {"name": name, "version": config["version"], **config["profiles"][name]}
    except KeyError as error:
        raise ValueError("unknown graph profile: {}".format(name)) from error


def main(argv: List[str] = None) -> int:
    """Validate a staged or published profile directory from the command line."""
    parser = argparse.ArgumentParser(description="Validate local Graphify artifacts")
    parser.add_argument("profile_dir", type=Path)
    parser.add_argument("--profile", default="coding")
    parser.add_argument("--profiles", type=Path, default=Path(__file__).with_name("profiles.json"))
    arguments = parser.parse_args(argv)
    try:
        artifacts = arguments.profile_dir
        manifest = json.loads((artifacts / "manifest.json").read_text(encoding="utf-8"))
        profile = _load_profile(arguments.profiles, arguments.profile)
        validate_directory(artifacts, manifest, profile)
    except (OSError, json.JSONDecodeError, ValueError) as error:
        print("Graphify profile invalid: {}".format(error), file=sys.stderr)
        return 1
    print("Graphify profile valid: {}".format(manifest["build_id"]))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

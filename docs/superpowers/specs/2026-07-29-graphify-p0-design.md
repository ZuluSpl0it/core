# Graphify P0 Design

## Goal

Make a local-only, source-focused knowledge graph trustworthy and useful for work on the `ustc_staking` branch, without changing `main` or publishing graph artifacts upstream.

## Scope

This P0 delivers graph isolation, graph profiles, atomic artifact generation, provenance, and integrity validation. It does not add the curated architecture documents or semantic code-to-document bridges identified in the review; those are P1.

## Artifact policy

`graphify-out/` is local-only. It is ignored by Git and is never committed. Reproducible scripts, graph scope configuration, validation tests, and agent workflow guidance are versioned on `ustc_staking`.

## Graph profiles

The refresh tooling builds two independent graphs under `graphify-out/`.

1. `coding/` is the default graph for feature work. It includes application wiring, commands, Terra modules, custom modules, Wasm bindings, server/types code, selected tests, human module specifications, and `.proto` contracts. It excludes generated protobuf Go implementations, generated gateway Go implementations, Swagger runtime/minified JavaScript, binaries, lockfiles, test fixtures, and historical changelog content.
2. `domain-api/` is an optional graph for API and domain work. It includes human specifications, Terra protobuf contracts, OpenAPI definitions, upgrade documentation, and test READMEs. It excludes generated Go bindings and Swagger runtime JavaScript.

Generated Go bindings remain available through the optional profile, but are not expanded into implementation-level symbols in the coding profile.

## Atomic refresh contract

A single refresh command creates a temporary build directory, extracts the selected profile, builds the graph, labels communities, emits `graph.json`, report, HTML, and wiki, validates them, then renames the completed directory into place. A failed command leaves the previous completed profile untouched.

Every output receives the same build manifest containing: profile name and version, Git SHA, Graphify version, build timestamp, included/excluded file counts, node count, edge count, and a SHA-256 digest of the serialized graph. The report and wiki index must contain the same build ID and counts as the manifest. The validator fails on disagreement.

## Integrity gates

The refresh fails when generated-source nodes exceed 5% of the coding graph, output counts disagree, the graph contains no nodes, or missing endpoints/self-loops are nonzero. External dependencies are represented or classified separately so they do not appear as local dangling-edge corruption. The validator records dangling-edge rate and rejects a coding graph above the configured threshold of 5%.

## Agent workflow

Versioned guidance tells coding agents to check graph freshness before feature work, use the coding graph first, inspect the cited source before changing code, and treat inferred relationships as leads rather than evidence. The graph remains optional: source files are the final authority.

## Testing

Tests use a small fixture corpus. They prove profile exclusions, provenance generation, atomic replacement behavior, and rejection of mismatched output metadata. The repository test suite remains the baseline check.

## Non-goals

- Commit or publish `graphify-out/`.
- Change `main`.
- Build a remote graph service or CI publication flow.
- Add architecture documentation beyond workflow guidance.

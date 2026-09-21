# Phase 2 community-pool funding preflight

| Gate | Evidence | Result |
|---|---|---|
| Production module activation | [`phase-1-release-readiness.md`](phase-1-release-readiness.md) states the module is not ready for public testnet or production and requires a live upgrade rehearsal before activation. The campaign evidence is local seven-validator testing. No production activation record appears in the project release evidence. | PASS based on current project records; reconfirm against the target chain before deployment |
| Base revision | Phase 1 hardening merge | `3bdf007e` |
| Contract branch exclusion | `git merge-base --is-ancestor phase2-treasury-manager HEAD` returns 1 | PASS |
| Existing store version | Phase 1 release readiness says the upgrade rehearsal is still a blocking gate; there is no project evidence of an activated production `x_ustcstaking` store. | PASS based on current project records; reconfirm against the target chain before deployment |

The production activation decision is based on repository release records and
the reported local-validator test campaign. It is not a direct query of a
production RPC endpoint. Before a chain upgrade, operators must independently
confirm the target chain has no active USTC staking store. If Phase 1 has
activated, stop this implementation path and prepare a separately approved
module-version migration.

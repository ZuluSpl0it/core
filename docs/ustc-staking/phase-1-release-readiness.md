# USTC Staking Phase 1 Release Readiness

## Current decision

Phase 1 functional behavior is ready for continued controlled testing, not
yet for a public-testnet or production upgrade. The local seven-validator
campaign passed the user lifecycle and restart paths. Code-level hardening
now adds owner indexing, imported-state checks, registered invariants, events,
the USTC simulation decoder, and real-app keeper coverage.

The remaining release blockers are an executed live-chain upgrade rehearsal,
the final repeated seven-validator campaign on the release revision, and
dependency/security disposition. Do not activate `ustc_staking` on a live
chain until every blocking gate below has recorded evidence.

## Blocking gates

1. **Build and tests**
   - `GOMAXPROCS=2 go test ./x/ustcstaking/... -count=1 -p 1`
   - real-app integration test passes;
   - export/import validation passes;
   - fuzz and race runs complete or have an explicit review disposition.

2. **Upgrade rehearsal**
   - start from a pre-USTC-staking app state;
   - apply plan `ustc_staking` at the intended height;
   - verify the `x_ustcstaking` store is added exactly once;
   - verify both module accounts exist with empty permissions;
   - export, restart, and re-import the upgraded state;
   - confirm existing LUNC validator state and all pre-existing balances are unchanged.

3. **Accounting and custody**
   - principal pool equals the sum of non-withdrawn principal;
   - active shares equal the reward-state total;
   - reward pool covers all claimable rewards;
   - no USTC module account has `Minter` or `Burner` permission;
   - supply conservation is demonstrated by real BankKeeper integration, not
     registered as a global runtime invariant.

4. **Seven-validator campaign**
   - rerun `phase-1-localnet-testing.md` from a clean generated network;
   - verify events, owner-scoped queries, pause/resume governance, claims,
     unbonding, withdrawal, unauthorized actions, node restart, and full
     network restart;
   - attach command logs, revision, genesis hash, and final supply comparison.

5. **Security and operations**
   - record dependency scanner output and dispositions in
     `phase-1-dependency-triage.md`;
   - review query resource bounds and event compatibility;
   - obtain independent validator review and governance approval;
   - document rollback authority and the no-retry rule for a failed upgrade.

## Evidence record

Record each gate with revision, date, operator, command, exit status, and
artifact path. A green unit test is not a substitute for upgrade or localnet
evidence. A clean localnet is not a substitute for independent review.

## Rollback and incident rule

If the upgrade handler, state validation, invariant check, or module-account
permission check fails, halt activation and preserve the failing state and
logs. Do not retry at a new height until the failure is understood and a new
release candidate is reviewed. This module has no safe in-place rollback
operation after state mutation.

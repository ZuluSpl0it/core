# Local Graphify Workflow

`graphify-out/` is local-only and ignored by Git. Never add it to a commit or publish it upstream.

Before feature work, run `scripts/graphify-refresh coding` when its manifest does not match `git rev-parse HEAD`.

Use graph queries and the local wiki to navigate. Source files are authoritative: inspect every cited source before editing. Treat inferred edges as leads to verify, not proof.

Use `domain-api` for contracts, specifications, and upgrades. Use `coding` for implementation impact, module lifecycle, keepers, transactions, and test paths.

## USTC Staking Baseline

Before USTC-staking implementation work, read these in order:

1. `docs/ustc-staking/architecture-foundation.md`
2. `docs/superpowers/specs/2026-07-29-ustc-staking-phase-1-design.md`
3. `docs/superpowers/plans/2026-07-29-ustc-staking-phase-1.md`
4. `docs/superpowers/specs/2026-09-19-ustc-staking-phase-1-release-hardening-design.md`
5. `docs/superpowers/plans/2026-09-19-ustc-staking-phase-1-release-hardening.md`
6. `docs/superpowers/specs/2026-09-19-ustc-staking-phase-2-community-pool-funding-design.md`
7. `docs/superpowers/plans/2026-09-19-ustc-staking-phase-2-community-pool-funding.md`
8. `docs/superpowers/specs/2026-07-29-ustc-staking-phase-3-validator-program-design.md`
9. `docs/superpowers/plans/2026-07-29-ustc-staking-phase-3-validator-program.md`

Keep the native `x/ustcstaking` ledger separate from `custom/staking` and LUNC
validator consensus. Phase 2 reward funding is governance-authorized and moves
existing USTC from the distribution community pool through the constrained
native funding API; do not add TreasuryManager, POL, Wasm, minting, or direct
reward-pool deposits. Phase 3 validator-program work is an optional incentive
overlay only: it must not change validator power, LUNC self-bond, commission,
slashing, jailing, or distribution rewards. Its performance pool is distinct
from Phase 1 staker rewards.

For stakeholder communication, use the audience-specific briefings rather than
condensing technical plans ad hoc:

- `docs/ustc-staking/briefings/technical-advisor-briefing.md`
- `docs/ustc-staking/briefings/c-suite-briefing.md`

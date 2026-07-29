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
4. `docs/superpowers/specs/2026-07-29-ustc-staking-phase-2-funding-design.md`
5. `docs/superpowers/plans/2026-07-29-ustc-staking-phase-2-funding.md`
6. `docs/superpowers/specs/2026-07-29-ustc-staking-phase-3-validator-program-design.md`
7. `docs/superpowers/plans/2026-07-29-ustc-staking-phase-3-validator-program.md`

Keep the native `x/ustcstaking` ledger separate from `custom/staking` and LUNC
validator consensus. Phase 2 TreasuryManager/POL work may fund rewards only
through the native module's constrained funding-authority interface.
Phase 3 validator-program work is an incentive overlay only: it must not
change validator power, LUNC self-bond, commission, slashing, jailing, or the
distribution module.

For stakeholder communication, use the audience-specific briefings rather than
condensing technical plans ad hoc:

- `docs/ustc-staking/briefings/technical-advisor-briefing.md`
- `docs/ustc-staking/briefings/c-suite-briefing.md`

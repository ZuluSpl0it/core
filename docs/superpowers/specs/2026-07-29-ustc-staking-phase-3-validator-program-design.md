# USTC Staking Phase 3 Validator Program and Activation Design

## Decision

Phase 3 is the final planned phase. It adds an opt-in USTC-staking validator
program, an initial governance-managed allow-list/cap, a USTC self-bond
eligibility check, and an optional performance pool. It also defines the
upgrade, audit, monitoring, governance, and launch gates for Phases 1–3.

The program remains an incentive overlay. It reads validator existence/status
from the existing staking keeper but never changes voting power, validator-set
selection, commission, slashing, jailing, redelegation, or LUNC self-bond.

## Why this is a separate final phase

Validator policy is consensus-adjacent operationally even when it has no
consensus effect. Keeping it after the native ledger and revenue adapter means
its allow-list, performance policy, and operational risks cannot block user
principal custody or revenue accounting. The existing `x/dyncomm` module is
the local example of reading validator state and using an end blocker; Phase 3
does not copy its commission-changing behavior.

## Program model

### Enrollment

- Governance sets an allow-list and a maximum number of program participants;
  initial defaults are 21 validators and a 1,000,000 USTC (`1_000_000_000000`
  `uusd`) minimum self-bond, matching the supplied paper's policy input.
- A validator operator enrolls by naming an active Phase 1 USTC position owned
  by its corresponding account address. The position must be active, USTC-only,
  and meet the configured minimum at enrollment and each epoch finalization.
- Enrolling does not create a new validator, alter LUNC staking, or reserve a
  validator-set slot. It only makes the operator eligible for program rewards.
- If the position begins unbonding or falls below the requirement, the
  enrollment becomes ineligible at the next epoch finalization. The user's
  Phase 1 withdrawal rights remain intact.

### Performance pool

- The pool is disabled by default and funded only by a distinct,
  governance-authorized USTC transfer. It is not paid from user principal,
  Phase 1 staker rewards, minted supply, or the validator distribution module.
- A configured `performance_authority` submits a finalized epoch with one
  non-negative score per eligible validator and a funding amount already held
  in a dedicated performance-pool module account.
- Governance publishes the scoring method and evidence off-chain before each
  activation. The chain verifies authorization, eligibility, score bounds,
  total allocation, and one-finalization-per-epoch; it does not pretend to
  derive a complete uptime score from incomplete rolling slashing counters.
- Validators claim their score-proportional USTC allocation. Dust remains in
  the performance pool for the next epoch.

This authority-submitted score differs from standard slashing: it is a
transparent incentive report, not a consensus penalty. No validator is jailed,
slashed, or removed because of a Phase 3 score.

## Deliberate differences from standard validator staking

| Area | Standard staking | Phase 3 program | Why |
|---|---|---|---|
| Validator eligibility | determines validator set and power | determines access to an optional USTC program | no effect on consensus |
| Self-bond | LUNC delegation securing the chain | independently locked USTC Phase 1 position | proposal requires USTC commitment, not consensus stake |
| Bad performance | slashing/jailing | no reward for an ineligible/zero-score epoch | avoid a second consensus penalty system |
| Metrics | consensus signing/slashing state | authorized, published epoch report | rolling signing counters alone are insufficient for a complete policy metric |
| Payout source | distribution/commission | pre-funded USTC performance-pool account | no minting and no claim on staker reward pool |
| Timing | validator/block lifecycle | explicit epoch finalization and claims | deterministic, auditable, and no mandatory end blocker |

## Native architecture

Phase 3 extends `x/ustcstaking`; it does not create a competing staking module.
It adds:

- `ValidatorEnrollment` records keyed by validator operator address.
- `PerformanceEpoch` and per-validator allocation/claim records.
- `ValidatorPerformancePoolName` module account with no Minter/Burner rights.
- Narrow read-only `StakingKeeper` interface for validator existence/status.
- Governance messages for program params, allow-list, authority rotation, and
  pool funding; a performance-authority message to finalize an epoch.
- Queries for enrollment, eligibility, epoch details, allocations, and claims.

No end blocker is required: enrollment eligibility is evaluated at an explicit
epoch finalization, and participants claim on demand. This avoids adding
per-block validator iteration merely for a periodic incentive program.

## Safety, governance, and activation

- Program params include allow-list, cap, minimum USTC self-bond, epoch duration,
  `performance_authority`, and paused flag. Governance changes are timelocked.
- An epoch finalization must have a unique epoch ID, a closed interval, a
  positive bounded score total, no duplicate validators, and allocations no
  greater than the funded performance-pool balance.
- Enrollment/epoch operations are pauseable. Pausing never blocks Phase 1
  principal withdrawal, Phase 1 reward claims, or already-created performance
  claims.
- The launch package requires the Phase 1 and 2 audits, Phase 3 audit,
  upgrade rehearsal, contract code checksum, governance timelock evidence,
  source/authority configuration, monitoring dashboards, and rollback/pause
  drills.
- The native upgrade identifier is `ustc_staking`. It is a stable technical
  identifier used by the app's upgrade handler; the governance proposal title
  may use different human-readable wording.

## Final-phase checklist

- [ ] Define eligibility, epoch, score, allocation, and claim state/messages.
- [ ] Implement enrollment and performance-pool accounting with invariants.
- [ ] Integrate read-only validator status and module account wiring.
- [ ] Add governance/timelock and performance-authority controls.
- [ ] Test adversarial enrollment, score, funding, claim, pause, and upgrade flows.
- [ ] Complete independent audits and launch rehearsal for all three phases.
- [ ] Publish scoring methodology, authority identities, allow-list, and operations runbook.

## Out of scope after Phase 3

- Consensus validator selection, LUNC self-bond enforcement, slashing/jailing,
  commission changes, or distribution-module modifications.
- Automatic on-chain uptime scoring beyond data that can be proven complete and
  policy-approved in a future separately specified upgrade.
- Additional DEX, POL, or revenue economics beyond Phase 2's constrained
  funding interface.

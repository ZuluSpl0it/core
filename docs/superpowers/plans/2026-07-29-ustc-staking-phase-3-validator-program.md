# USTC Staking Phase 3 Validator Program Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an opt-in USTC validator program and optional pre-funded performance pool without changing LUNC consensus staking behavior.

**Architecture:** Extend `x/ustcstaking` with validator enrollment and explicit performance epochs. It reads validator existence/status through a narrow `StakingKeeper`, validates a Phase 1 USTC position at enrollment/finalization, and pays only allocations already covered by `ustcstaking_validator_performance_pool`. No end blocker, staking hook, or distribution/slashing integration is added.

**Tech Stack:** Go, Cosmos SDK v0.50, protobuf + Buf, SDK math, gRPC-gateway, Go invariants, existing app upgrade and Wasm integration test harnesses.

---

## Persistent Phase 3 checklist

- [ ] Do not modify `custom/staking`, `x/staking`, distribution, mint, slashing, or validator power paths.
- [ ] Keep every performance payout USTC-only and pre-funded.
- [ ] Treat `performance_authority` as a constrained reporting authority, never governance authority.
- [ ] Verify every epoch score, allocation, and claim against module-account balances.
- [ ] Complete Phases 1–3 audit and launch gates before activation.

### Task 1: Define validator-program protobuf and state types

**Files:**
- Modify: `proto/terra/ustcstaking/v1/{ustcstaking,tx,query,genesis}.proto`
- Modify: `x/ustcstaking/types/*.pb.go`
- Create: `x/ustcstaking/types/{validator_program,validator_program_test}.go`

- [ ] **Step 1: Write failing validation tests.**

```go
func TestEnrollmentRejectsNonAllowlistedValidator(t *testing.T) {}
func TestEpochRejectsDuplicateValidatorAndNonPositiveScoreTotal(t *testing.T) {}
func TestProgramParamsRejectInvalidSelfBondAndEpochDuration(t *testing.T) {}
```

- [ ] **Step 2: Add durable protobuf state.**

```proto
message ValidatorEnrollment { string validator_address = 1; uint64 position_id = 2; bool active = 3; }
message PerformanceEpoch { uint64 id = 1; google.protobuf.Timestamp end_time = 2; string total_score = 3; cosmos.base.v1beta1.Coin funded_amount = 4; }
message PerformanceAllocation { uint64 epoch_id = 1; string validator_address = 2; string score = 3; cosmos.base.v1beta1.Coin amount = 4; bool claimed = 5; }
```

Add `MsgEnrollValidator`, `MsgUnenrollValidator`, `MsgFundPerformancePool`,
`MsgFinalizePerformanceEpoch`, `MsgClaimPerformanceRewards`, and governance
messages to update program params/allow-list/performance authority.

- [ ] **Step 3: Implement validation.** Require valid `valoper` and account
addresses, `uusd`, unique epoch/validator pairs, non-negative scores, at least
one positive score, and no allocation above funded pool balance.

- [ ] **Step 4: Generate and commit.**

Run: `make proto-gen && go test ./x/ustcstaking/types -run 'Test(Enrollment|Epoch|ProgramParams)' -count=1`

```bash
git add proto/terra/ustcstaking x/ustcstaking/types
git commit -m "feat: define USTC validator program API"
```

### Task 2: Implement enrollment and read-only validator checks

**Files:**
- Modify: `x/ustcstaking/types/expected_keepers.go`
- Create: `x/ustcstaking/keeper/{validator_program,enrollment}.go`
- Create: `x/ustcstaking/keeper/enrollment_test.go`

- [ ] **Step 1: Write enrollment lifecycle tests.**

```go
func TestEnrollRequiresActiveValidatorAndEligiblePhaseOnePosition(t *testing.T) {}
func TestEnrollmentDoesNotCallStakingMutationMethods(t *testing.T) {}
func TestUnenrollKeepsPhaseOnePositionWithdrawable(t *testing.T) {}
```

- [ ] **Step 2: Add the narrow interface.**

```go
type StakingKeeper interface {
  GetValidator(sdk.Context, sdk.ValAddress) (stakingtypes.Validator, error)
}
```

Use it only to verify the validator exists and is not unbonded. Convert the
operator address to its matching account form and require ownership of the
referenced active Phase 1 position. Never call delegation, power, slash, jail,
or hook APIs.

- [ ] **Step 3: Implement enrollment.** Require allow-list membership and cap
availability; persist one enrollment per operator. Revalidate the position at
epoch finalization rather than preventing a user from beginning Phase 1
unbonding. This differs from a consensus self-bond because it withdraws only
program eligibility, not validator security or user custody rights.

- [ ] **Step 4: Run and commit.**

Run: `go test ./x/ustcstaking/keeper -run 'Test(Enroll|Unenroll)' -count=1`

```bash
git add x/ustcstaking
git commit -m "feat: add USTC validator enrollment"
```

### Task 3: Add performance-pool custody and epoch allocation

**Files:**
- Modify: `app/modules.go`
- Modify: `x/ustcstaking/types/{keys,params,errors}.go`
- Create: `x/ustcstaking/keeper/{performance_pool,performance_epoch}.go`
- Create: `x/ustcstaking/keeper/performance_epoch_test.go`

- [ ] **Step 1: Write failing custody tests.**

```go
func TestFundPerformancePoolRequiresGovernanceAndUUSD(t *testing.T) {}
func TestFinalizeEpochPaysOnlyEligibleValidators(t *testing.T) {}
func TestEpochAllocationNeverExceedsPerformancePoolBalance(t *testing.T) {}
func TestEpochDustRemainsInPerformancePool(t *testing.T) {}
```

- [ ] **Step 2: Add module account.** Define
`ValidatorPerformancePoolName = "ustcstaking_validator_performance_pool"` and
add it to `maccPerms` with nil permissions. It differs from distribution:
funding is explicit USTC, never minted inflation or validator commission.

- [ ] **Step 3: Implement funding and allocation.** Governance funds the pool
from its account. `performance_authority` finalizes a unique, closed epoch;
the keeper filters ineligible entries, computes proportional integer shares,
retains dust, and writes one allocation per eligible validator. It cannot
transfer funds during finalization.

- [ ] **Step 4: Add performance-pool invariants and commit.** Check pool
balance covers all unclaimed allocations plus dust, with no duplicate
allocation or claim record.

Run: `go test ./x/ustcstaking/keeper -run 'Test(FundPerformance|FinalizeEpoch|PerformanceInvariant)' -count=1`

```bash
git add app/modules.go x/ustcstaking
git commit -m "feat: add validator performance pool"
```

### Task 4: Implement claims, queries, CLI, and no-end-block behavior

**Files:**
- Modify: `x/ustcstaking/keeper/{msg_server,query_server}.go`
- Create: `x/ustcstaking/client/cli/validator_program.go`
- Create: `x/ustcstaking/keeper/{query_server,performance_claim}_test.go`

- [ ] **Step 1: Write claim/query tests.**

```go
func TestValidatorCanClaimOnlyOwnUnclaimedAllocation(t *testing.T) {}
func TestClaimTransfersFromPerformancePoolOnlyOnce(t *testing.T) {}
func TestNoEndBlockerNeededToFinalizeOrClaimEpoch(t *testing.T) {}
```

- [ ] **Step 2: Implement claims and queries.** Claim verifies the caller
matches the validator's account address, marks the allocation claimed, then
transfers USTC from `ValidatorPerformancePoolName`. Query enrollment,
eligibility reason, epoch, allocation, and claim state without mutating state.

- [ ] **Step 3: Add CLI.**

```text
terrad tx ustcstaking enroll-validator <valoper> <position-id>
terrad tx ustcstaking unenroll-validator <valoper>
terrad tx ustcstaking claim-performance-rewards <epoch-id>
terrad query ustcstaking validator-enrollment <valoper>
terrad query ustcstaking performance-epoch <epoch-id>
```

Governance and performance-authority messages remain controlled transactions,
not user-default CLI actions.

- [ ] **Step 4: Keep blockers unchanged and commit.** Do not add Phase 3 to
`orderBeginBlockers` or `orderEndBlockers`; explicit finalization avoids
per-block validator scans.

Run: `go test ./x/ustcstaking/... -run 'Test(Claim|Query|NoEndBlocker)' -count=1`

```bash
git add x/ustcstaking
git commit -m "feat: add validator program claims"
```

### Task 5: Add governance, timelock, and authority-rotation controls

**Files:**
- Modify: `x/ustcstaking/types/{params,genesis}.go`
- Modify: `x/ustcstaking/keeper/{msg_server,genesis}.go`
- Create: `x/ustcstaking/keeper/governance_test.go`
- Create: `docs/ustc-staking/phase-3-release-readiness.md`

- [ ] **Step 1: Write governance tests.**

```go
func TestOnlyGovernanceCanChangeAllowlistCapOrPerformanceAuthority(t *testing.T) {}
func TestProgramPauseDoesNotBlockPhaseOneWithdrawalOrClaim(t *testing.T) {}
func TestTimelockedProgramParameterChangeCannotExecuteEarly(t *testing.T) {}
```

- [ ] **Step 2: Implement timelocked program changes.** Store pending program
params with an activation timestamp at least 30 days in the future. Governance
may schedule/execute changes; immediate pause and allow-list removal are safety
operations. Phase 1 params and Phase 2 TreasuryManager authorities are not
changed by Phase 3 messages.

- [ ] **Step 3: Add genesis and migration coverage.** Export/import
enrollments, program params, pool state, epochs, allocations, claims, and
pending changes in canonical order. Increment module consensus version only
for this real schema change.

- [ ] **Step 4: Run and commit.**

Run: `go test ./x/ustcstaking/... -run 'Test(Governance|ProgramPause|Genesis)' -count=1`

```bash
git add x/ustcstaking docs/ustc-staking/phase-3-release-readiness.md
git commit -m "feat: govern USTC validator program"
```

### Task 6: Rehearse the chain upgrade and full three-phase launch

**Files:**
- Create: `app/upgrades/ustc_staking/{constants,upgrades,ustcstaking}_test.go`
- Modify: `app/app.go`
- Modify: `docs/ustc-staking/phase-3-release-readiness.md`
- Modify: `AGENTS.md`

- [ ] **Step 1: Create the `ustc_staking` upgrade package.** Use the existing
`app/upgrades/v14_2` constants/handler layout and register its handler in
`app/app.go`. Initialize the Phase 1 store and Phase 3 program defaults; do
not activate TreasuryManager or a performance pool until governance has
completed the Phase 2/3 timelocks. `ustc_staking` is the fixed app upgrade
identifier; it is independent of a governance proposal's display title.

- [ ] **Step 2: Write upgrade rehearsal tests.** Import pre-upgrade app state,
run the approved handler, export state, then assert USTC store/module accounts,
empty enrollments, disabled performance pool, and unchanged LUNC validator
power/commission/slashing state.

- [ ] **Step 3: Complete launch readiness.** Require published allow-list,
self-bond/lock tiers, performance methodology, authority addresses, contract
checksum, audit reports, Graphify validation, monitoring, incident response,
and a pause/rollback drill. Activation is blocked until each item is checked.

- [ ] **Step 4: Run final verification and commit.**

Run: `make proto-gen && go test ./x/ustcstaking/... -count=1 && go test ./app/upgrades/... ./wasmbinding/test -count=1 && go test ./...`

```bash
git add app/upgrades docs/ustc-staking AGENTS.md proto/terra/ustcstaking x/ustcstaking
git commit -m "docs: verify USTC staking phase 3 launch"
```

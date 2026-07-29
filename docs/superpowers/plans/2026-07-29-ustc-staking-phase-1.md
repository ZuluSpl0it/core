# USTC Staking Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a native USTC-only locking module with fully funded rewards and no impact on LUNC validator staking or token issuance.

**Architecture:** `x/ustcstaking` is an independent Cosmos SDK module with a principal escrow account and a reward-pool account. Positions contribute lock-tier shares to a module-local cumulative reward index; funding transfers real USTC into the reward pool before increasing that index. The module has no `StakingKeeper`, no distribution keeper, no mint keeper, and no block-processing requirement in Phase 1.

**Tech Stack:** Go, Cosmos SDK v0.50 module/keeper APIs, protobuf + Buf, SDK integer/decimal arithmetic, gRPC-gateway, `testify` suites, Go invariants.

---

## Locked Phase 1 policy defaults

- Denom: `uusd` only (the chain's USTC micro-denom); all other denoms fail.
- Authorities: `authority` governs parameters and can rotate
  `funding_authority`; `funding_authority` alone may sign `MsgFundRewards`.
  Phase 1 initializes both to governance so bootstrap funding remains
  governance-controlled; Phase 2 can rotate only the funding authority to
  TreasuryManager's contract address.
- Funding: `MsgFundRewards.sender` moves its own USTC into
  `ustcstaking_reward_pool` before reward accounting changes.
- Zero active shares: reject funding. This prevents the first staker from capturing funds deposited before any eligible position exists.
- Position lifecycle: `active → unbonding → withdrawn`; unbonding shares cease earning immediately. Rewards accrued through the start of unbonding remain claimable.
- Rewards: no minting, no distribution-module hooks, no validator commission, and no automatic APR. Claims are capped by funds held in the reward pool.
- Locks: parameters define lock tiers and share multipliers; the first implementation stores the chosen tier ID and its snapshot multiplier on every position so later parameter changes do not rewrite an existing agreement.
- Maturity: `Withdraw` checks the recorded completion timestamp. Do not add an end blocker merely to mature positions; this differs from validator staking because no consensus state must be updated at a particular block.

Changing any of these defaults requires a spec revision, test changes, and governance review before implementation.

## Repository map

| Concern | Existing reference | Phase 1 action |
|---|---|---|
| App store/keeper setup | `app/keepers/keepers.go:114` | add store key and `UstcStakingKeeper` after bank/account keepers |
| Basic/module registration | `app/modules.go:82`, `:142` | add `ustcstaking.AppModuleBasic{}` and `ustcstaking.NewAppModule(...)` |
| Native module conventions | `x/taxexemption/module.go`, `x/dyncomm/module.go` | follow codec, gateway, genesis, service, and migration shape |
| Proto conventions | `proto/terra/taxexemption/v1/*.proto` | add `proto/terra/ustcstaking/v1/{ustcstaking,tx,query,genesis}.proto` |
| Module account permissions | `app/modules.go:111` | add principal and reward module accounts; neither gets Minter or Burner permissions |
| CLI wiring | `cmd/terrad/root.go:149` | inherited through `ModuleBasics`; add module root tx/query commands |
| App tests | `x/*/keeper/test_utils.go`, `tests/e2e/initialization/config.go` | add isolated keeper setup and one end-to-end genesis/action flow |

## Persistent implementation checklist

- [ ] Complete Tasks 1–10 in order; do not start later tasks with failing earlier tests.
- [ ] For every standard-staking-looking API, document whether it is intentionally absent and why.
- [ ] Review every balance transfer for source account, destination account, denom, authority, and accounting update order.
- [ ] Run unit, invariant, integration, proto-generation, full Go, and upgrade-rehearsal checks before a release proposal.
- [ ] Keep Phase 2 integration limited to `MsgFundRewards` until its contract spec and audit are approved.

### Task 1: Define protobuf API and generate code

**Files:**
- Create: `proto/terra/ustcstaking/v1/ustcstaking.proto`
- Create: `proto/terra/ustcstaking/v1/tx.proto`
- Create: `proto/terra/ustcstaking/v1/query.proto`
- Create: `proto/terra/ustcstaking/v1/genesis.proto`
- Generated: `x/ustcstaking/types/*.pb.go`, `*.pb.gw.go`
- Test: `x/ustcstaking/types/msgs_test.go`

- [ ] **Step 1: Define the durable state vocabulary.**

```proto
message Position {
  uint64 id = 1;
  string owner = 2;
  cosmos.base.v1beta1.Coin principal = 3;
  uint32 lock_tier_id = 4;
  string share_multiplier = 5;
  string reward_debt = 6;
  google.protobuf.Timestamp unbonding_end_time = 7;
  PositionStatus status = 8;
}
message RewardState { string reward_index = 1; string total_shares = 2; }
message Params { string bond_denom = 1; repeated LockTier lock_tiers = 2; string authority = 3; string funding_authority = 4; bool paused = 5; }
```

- [ ] **Step 2: Define messages and queries.**

```proto
service Msg {
  rpc Stake(MsgStake) returns (MsgStakeResponse);
  rpc BeginUnbonding(MsgBeginUnbonding) returns (MsgBeginUnbondingResponse);
  rpc Withdraw(MsgWithdraw) returns (MsgWithdrawResponse);
  rpc ClaimRewards(MsgClaimRewards) returns (MsgClaimRewardsResponse);
  rpc FundRewards(MsgFundRewards) returns (MsgFundRewardsResponse);
  rpc UpdateParams(MsgUpdateParams) returns (MsgUpdateParamsResponse);
  rpc UpdateFundingAuthority(MsgUpdateFundingAuthority) returns (MsgUpdateFundingAuthorityResponse);
}
service Query {
  rpc Position(QueryPositionRequest) returns (QueryPositionResponse);
  rpc PositionsByOwner(QueryPositionsByOwnerRequest) returns (QueryPositionsByOwnerResponse);
  rpc RewardState(QueryRewardStateRequest) returns (QueryRewardStateResponse);
  rpc Params(QueryParamsRequest) returns (QueryParamsResponse);
}
```

- [ ] **Step 3: Write validation tests before message helpers.**

```go
func TestMsgStakeValidateBasicRejectsNonUSTC(t *testing.T) { /* `uluna` returns ErrInvalidDenom */ }
func TestMsgFundRewardsSignerIsFundingAuthority(t *testing.T) { /* one funding-authority signer */ }
func TestMsgUpdateFundingAuthoritySignerIsGovernanceAuthority(t *testing.T) {}
func TestPositionRejectsZeroPrincipalAndUnknownTier(t *testing.T) { /* deterministic failures */ }
```

- [ ] **Step 4: Generate protobuf and gateway code.**

Run: `make proto-gen`

Expected: generated USTC-staking Go and gateway files update with no manual edits.

- [ ] **Step 5: Commit.**

```bash
git add proto/terra/ustcstaking x/ustcstaking/types
git commit -m "feat: define USTC staking API"
```

### Task 2: Add types, parameter validation, codecs, and genesis validation

**Files:**
- Create: `x/ustcstaking/types/{keys,params,errors,codec,genesis,expected_keepers}.go`
- Create: `x/ustcstaking/types/{params,genesis,codec}_test.go`

- [ ] **Step 1: Write failing parameter and genesis tests.**

```go
func TestParamsValidateRejectsNonUSTCDenom(t *testing.T) {}
func TestParamsValidateRejectsDuplicateTierIDsAndNonPositiveMultipliers(t *testing.T) {}
func TestGenesisValidateRejectsPositionTotalDifferentFromStoredPrincipal(t *testing.T) {}
```

- [ ] **Step 2: Implement constants and narrow keeper interfaces.**

```go
const (
  ModuleName = "ustcstaking"
  StoreKey = "x_" + ModuleName
  PrincipalPoolName = ModuleName
  RewardPoolName = ModuleName + "_reward_pool"
)
type BankKeeper interface {
  SendCoinsFromAccountToModule(sdk.Context, sdk.AccAddress, string, sdk.Coins) error
  SendCoinsFromModuleToAccount(sdk.Context, string, sdk.AccAddress, sdk.Coins) error
  GetBalance(sdk.Context, sdk.AccAddress, string) sdk.Coin
}
```

- [ ] **Step 3: Implement key prefixes and validation.** Use distinct prefixes for
positions by ID, owner-to-ID index, unbonding completion index, params, and
reward state. Validate `uusd`, non-empty authority, unique tier IDs, positive
durations/multipliers, unique positions, and exact stored totals. Require valid
bech32 `authority` and `funding_authority` addresses; their equality is valid
for Phase 1 bootstrap.

The separate pools differ from standard staking's bonded/not-bonded pools:
they separate user principal from voluntarily funded rewards, not bonded
validator tokens from unbonding validator tokens.

- [ ] **Step 4: Register interfaces and Amino compatibility.** Follow
`x/taxexemption/types/codec.go`; register every `sdk.Msg` and protobuf type.

- [ ] **Step 5: Run focused tests and commit.**

Run: `go test ./x/ustcstaking/types -run 'Test(Params|Genesis|Msg)' -count=1`

```bash
git add x/ustcstaking/types
git commit -m "feat: add USTC staking types"
```

### Task 3: Implement keeper state and reward-index accounting

**Files:**
- Create: `x/ustcstaking/keeper/{keeper,position,rewards}.go`
- Create: `x/ustcstaking/keeper/{position,rewards}_test.go`

- [ ] **Step 1: Write failing accounting tests.**

```go
func TestFundThenClaimAllocatesBySnapshottedShares(t *testing.T) {}
func TestFundingWithNoActiveSharesFailsWithoutTransfer(t *testing.T) {}
func TestClaimNeverExceedsRewardPoolBalance(t *testing.T) {}
func TestTierChangeDoesNotChangeExistingPositionShares(t *testing.T) {}
```

- [ ] **Step 2: Implement deterministic state accessors.**

```go
func (k Keeper) GetPosition(ctx sdk.Context, id uint64) (types.Position, bool)
func (k Keeper) SetPosition(ctx sdk.Context, position types.Position)
func (k Keeper) GetRewardState(ctx sdk.Context) types.RewardState
func (k Keeper) SetRewardState(ctx sdk.Context, state types.RewardState)
func (k Keeper) AccruedRewards(position types.Position, state types.RewardState) math.Int
```

- [ ] **Step 3: Implement funding index update after transfer.** Transfer USTC
to `RewardPoolName`; only then add `amount / totalShares` to `rewardIndex`.
Retain truncation dust in the reward pool and record no claimable debt for it.
This differs from `x/distribution`: no validator commission, historical
rewards, or mint/distribution event is consulted.

- [ ] **Step 4: Run keeper tests and commit.**

Run: `go test ./x/ustcstaking/keeper -run 'Test(Fund|Claim|Tier)' -count=1`

```bash
git add x/ustcstaking/keeper
git commit -m "feat: add USTC reward accounting"
```

### Task 4: Implement position lifecycle messages

**Files:**
- Create: `x/ustcstaking/keeper/msg_server.go`
- Create: `x/ustcstaking/keeper/msg_server_test.go`

- [ ] **Step 1: Write lifecycle failure tests.**

```go
func TestStakeTransfersOnlyUSTCToPrincipalPool(t *testing.T) {}
func TestBeginUnbondingSettlesRewardsAndStopsFutureAccrual(t *testing.T) {}
func TestWithdrawBeforeCompletionFails(t *testing.T) {}
func TestWithdrawTransfersOnlyRecordedPrincipalToOwner(t *testing.T) {}
func TestOnlyAuthorityCanFundOrUpdateParams(t *testing.T) {}
```

- [ ] **Step 2: Implement `Stake`.** Validate pause, denom, amount, and tier;
transfer owner funds to `PrincipalPoolName`; snapshot the tier multiplier and
current reward index; persist the active position and increase total shares.

- [ ] **Step 3: Implement `BeginUnbonding`, `Withdraw`, and `ClaimRewards`.**
Settle reward debt before removing active shares, set the completion timestamp,
and index it for queries. `Withdraw` verifies owner/status/time before moving
the principal pool balance. `ClaimRewards` pays only accrued, rounded-down USTC
from `RewardPoolName`.

- [ ] **Step 4: Implement `FundRewards`, `UpdateParams`, and
`UpdateFundingAuthority`.** `FundRewards` requires `funding_authority`, active
shares, and USTC; `UpdateParams` and `UpdateFundingAuthority` require
`authority`. The funding transfer originates from the funding authority's own
account, so Phase 2 can use TreasuryManager's contract address through a
Stargate message. `UpdateParams` cannot alter a position's snapshotted
multiplier.

- [ ] **Step 5: Run lifecycle tests and commit.**

Run: `go test ./x/ustcstaking/keeper -run 'Test(Stake|BeginUnbonding|Withdraw|Fund)' -count=1`

```bash
git add x/ustcstaking/keeper
git commit -m "feat: add USTC position lifecycle"
```

### Task 5: Add queries and user CLI

**Files:**
- Create: `x/ustcstaking/keeper/query_server.go`
- Create: `x/ustcstaking/keeper/query_server_test.go`
- Create: `x/ustcstaking/client/cli/{query,tx}.go`
- Create: `x/ustcstaking/client/cli/{query,tx}_test.go`

- [ ] **Step 1: Write query tests for ID lookup, owner pagination, reward state,
and params.** Unknown IDs return `NotFound`; owner pagination is deterministic.

- [ ] **Step 2: Implement `QueryServer` using generated service interfaces.**
Return position principal, status, completion time, accumulated claimable
rewards, reward index, total shares, and params. Querying must not mutate
reward debt or state.

- [ ] **Step 3: Add CLI commands.**

```text
terrad tx ustcstaking stake <amount> <tier-id>
terrad tx ustcstaking begin-unbonding <position-id>
terrad tx ustcstaking withdraw <position-id>
terrad tx ustcstaking claim-rewards <position-id>
terrad query ustcstaking position <id>
terrad query ustcstaking positions <owner>
```

Authority-only commands are intentionally not ordinary user CLI defaults; they
remain governance/governance-authority transactions.

- [ ] **Step 4: Run query/CLI tests and commit.**

Run: `go test ./x/ustcstaking/... -run 'Test(Query|GetQueryCmd|GetTxCmd)' -count=1`

```bash
git add x/ustcstaking
git commit -m "feat: add USTC staking queries"
```

### Task 6: Add genesis, migration, invariants, and no-op end-block policy

**Files:**
- Create: `x/ustcstaking/{genesis,module}.go`
- Create: `x/ustcstaking/{genesis,module}_test.go`
- Create: `x/ustcstaking/keeper/invariants.go`
- Create: `x/ustcstaking/keeper/invariants_test.go`

- [ ] **Step 1: Write failing round-trip and invariant tests.**

```go
func TestGenesisExportImportPreservesPositionsQueuesAndRewardIndex(t *testing.T) {}
func TestPrincipalInvariantMatchesPrincipalPoolBalance(t *testing.T) {}
func TestRewardInvariantDoesNotExceedRewardPoolBalance(t *testing.T) {}
func TestNoEndBlockerRequiredForMatureWithdrawal(t *testing.T) {}
```

- [ ] **Step 2: Implement genesis and consensus version 1.** Import params,
reward state, positions, owner indexes, and unbonding indexes. Export the same
canonical order. Register a 1→2 migration only when a later schema revision is
actually introduced; do not add a fake migration in the initial release.

- [ ] **Step 3: Register invariants.** Check stored principal total equals the
principal module-account USTC balance; reward-pool balance covers claimable
rewards plus dust; total active shares equals active positions' snapshotted
shares.

- [ ] **Step 4: Keep end block absent.** Maturity is checked at withdrawal.
This is intentionally unlike the existing `x/treasury` and `x/dyncomm` end
blockers because no periodic recalculation or consensus-side effect exists.

- [ ] **Step 5: Commit.**

```bash
git add x/ustcstaking
git commit -m "feat: add USTC staking genesis and invariants"
```

### Task 7: Integrate the module into the application

**Files:**
- Modify: `app/keepers/keepers.go`
- Modify: `app/modules.go`
- Modify: `app/app.go` only if wiring requires it
- Modify: `tests/e2e/initialization/config.go`
- Test: `app/modules_test.go` (create if absent) and `x/ustcstaking/keeper/test_utils.go`

- [ ] **Step 1: Write integration tests before wiring.**

```go
func TestTerraAppRegistersUstcStakingModuleAndStores(t *testing.T) {}
func TestModuleAccountsHaveNoMinterOrBurnerPermission(t *testing.T) {}
func TestGenesisIncludesUstcStakingDefaults(t *testing.T) {}
func TestFundingAuthorityCanBeRotatedWithoutChangingGovernanceAuthority(t *testing.T) {}
```

- [ ] **Step 2: Add one KV store and keeper.** Add `ustcstakingtypes.StoreKey`
to `keys`, add `UstcStakingKeeper` to `AppKeepers`, and construct it after
account and bank keepers using the governance module address as both initial
`authority` and `funding_authority`.
Do not inject `StakingKeeper`, distribution, mint, or slashing dependencies.

- [ ] **Step 3: Add module accounts.** Add `PrincipalPoolName` and
`RewardPoolName` to `maccPerms` with `nil` permissions. This differs from
staking's bonded pools: neither account burns, mints, delegates, or influences
validator state.

- [ ] **Step 4: Register the module.** Add the basic module and app module;
include it in `orderInitGenesis` after bank and before dependent consumers.
Do not add it to begin/end blocker orders. Verify `BasicModuleManager` exposes
CLI and gRPC-gateway routes.

- [ ] **Step 5: Extend the E2E genesis helper and commit.**

Run: `go test ./app ./x/ustcstaking/... ./tests/e2e/initialization -count=1`

```bash
git add app tests/e2e/initialization x/ustcstaking
git commit -m "feat: wire USTC staking module"
```

### Task 8: Add adversarial, property, and upgrade-rehearsal tests

**Files:**
- Create: `x/ustcstaking/keeper/fuzz_test.go`
- Create: `x/ustcstaking/keeper/upgrade_rehearsal_test.go`
- Modify: `docs/ustc-staking/architecture-foundation.md`

- [ ] **Step 1: Add fuzz/property tests.** Generate random stake, funding,
claim, unbonding, withdrawal, and invalid-denom sequences. After every action,
assert the three invariants and that no account receives more USTC than was
funded plus its returned principal.

- [ ] **Step 2: Add failure-path tests.** Cover authority substitution,
funding with zero shares, pause, duplicate withdrawal, stale position owner,
malformed pagination, decimal truncation, and reward-pool insolvency.

- [ ] **Step 3: Rehearse initial-state import.** Construct an application test
state without the USTC module, apply the module's default genesis through the
same module-manager initialization path, and export it. Assert that both
module accounts, default params, an empty position index, and zero reward
state are present. The module starts at consensus version 1; a later chain
upgrade handler is responsible only for invoking this already-tested genesis
initialization path, not for a fictional 1→2 migration.

- [ ] **Step 4: Commit.**

```bash
git add x/ustcstaking app/upgrades docs/ustc-staking
git commit -m "test: harden USTC staking upgrade paths"
```

### Task 9: Generate artifacts and run the full verification matrix

**Files:**
- Modify only generated protobuf artifacts and documentation as produced above.

- [ ] **Step 1: Format and regenerate.**

Run: `make proto-gen && gofmt -w $(rg --files -g '*.go' x/ustcstaking) app/keepers/keepers.go app/modules.go`

- [ ] **Step 2: Run focused module tests.**

Run: `go test ./x/ustcstaking/... -count=1`

Expected: all USTC module, keeper, query, CLI, genesis, invariant, and fuzz
tests pass.

- [ ] **Step 3: Run integration and repository tests.**

Run: `go test ./app ./tests/e2e/... -count=1 && go test ./...`

Expected: all packages pass with no changes to `custom/staking` behavior.

- [ ] **Step 4: Verify local knowledge artifacts only.**

Run: `scripts/graphify-refresh coding && python3 -m tools.graphify.validate graphify-out/coding --profile coding && git check-ignore graphify-out/coding/graph.json`

Expected: refreshed, valid, ignored local graph; no Graphify artifact staged.

- [ ] **Step 5: Commit only source/docs/generated code.**

```bash
git status --short
git add proto/terra/ustcstaking x/ustcstaking app tests docs
git commit -m "docs: verify USTC staking phase 1"
```

### Task 10: Complete the release-readiness checklist before activation

**Files:**
- Create: `docs/ustc-staking/phase-1-release-readiness.md`

- [ ] **Step 1: Record exact deployed parameters.** Include USTC denom,
authority, lock tiers, multipliers, unbonding durations, pause procedure, and
initial reward funding transaction.

- [ ] **Step 2: Record non-standard behavior.** State plainly that this is not
validator staking: it gives no voting power, has no slashing, makes no yield
promise, and uses only pre-funded rewards.

- [ ] **Step 3: Obtain independent review.** Require accounting, authorization,
genesis/upgrade, and DoS review; record findings and disposition before the
governance proposal is submitted.

- [ ] **Step 4: Gate Phase 2.** Do not deploy a POL/TreasuryManager adapter
until its dedicated spec, contract audit, allocation/timelock policy, and
failure-mode tests are complete.

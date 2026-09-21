# USTC Staking Phase 1 Release-Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden the validated Phase 1 `x/ustcstaking` module for public-testnet and production-upgrade review without changing its economics or LUNC validator behavior.

**Architecture:** Add a derived owner-position index, strict imported-state validation, registered module-accounting invariants, stable events, and a module-specific simulation decoder. Add the `ustc_staking` application upgrade with `x_ustcstaking` store creation and default genesis initialization, then prove the result with real AccountKeeper/BankKeeper integration tests and a seven-validator rerun.

**Tech Stack:** Go, Cosmos SDK v0.53 APIs used by this repository, CometBFT ABCI, protobuf-generated USTC types, `testify`, Go fuzz/race tests, the existing Terra upgrade framework, and localnet Docker tooling.

---

## Non-negotiable boundaries

- [ ] Do not modify `custom/staking`, `x/staking`, distribution, mint, slashing, jailing, validator power, commission, or LUNC delegation behavior.
- [ ] Keep `uusd` as the only principal and reward denom.
- [ ] Keep principal and reward module accounts permissionless: no Minter or Burner permissions.
- [ ] Do not add TreasuryManager/POL or Phase 3 validator-program code.
- [ ] Do not register global USTC supply conservation as a runtime invariant; prove it in real application tests and permission checks.
- [ ] Keep `graphify-out/` local-only and unstaged.

## File map

| Area | Files | Responsibility |
|---|---|---|
| Owner index | `x/ustcstaking/types/keys.go`, `x/ustcstaking/keeper/position.go`, `x/ustcstaking/keeper/query_server.go` | Owner-to-position secondary records |
| Imported state | `x/ustcstaking/types/params.go`, `x/ustcstaking/types/genesis.go`, `x/ustcstaking/types/errors.go`, `x/ustcstaking/keeper/genesis_validation.go` | Reject malformed tiers, positions, reward state, and exhausted IDs |
| Invariants | `x/ustcstaking/keeper/invariants.go`, tests, `x/ustcstaking/module.go` | Principal, share, and reward-solvency routes |
| Upgrade | `app/upgrades/ustc_staking/`, `app/app.go` | Add store, run new-module genesis, materialize module accounts |
| Events | `x/ustcstaking/types/events.go`, `x/ustcstaking/keeper/msg_server.go` | Stable success-only event contracts |
| Simulation | `x/ustcstaking/simulation/`, `x/ustcstaking/module.go` | Decode USTC records instead of Market records |
| App tests | `app/ustc_staking_upgrade_test.go`, `x/ustcstaking/keeper/integration_test.go` | Real keepers, upgrade, invariants, and validator isolation |
| Operations | `docs/ustc-staking/` | Correct runbook and record release gates |

## Task 1: Build the owner-position secondary index

**Files:**

- Modify: `x/ustcstaking/types/keys.go`
- Modify: `x/ustcstaking/keeper/position.go`
- Modify: `x/ustcstaking/keeper/query_server.go`
- Modify: `x/ustcstaking/genesis.go`
- Test: `x/ustcstaking/keeper/position_test.go`
- Test: `x/ustcstaking/keeper/query_server_test.go`
- Test: `x/ustcstaking/genesis_test.go`

- [ ] **Step 1: Define canonical owner-index keys.**

Add helpers with fixed-width IDs:

```go
func OwnerPositionKey(owner sdk.AccAddress, id uint64) []byte {
	key := make([]byte, 0, len(owner.Bytes())+8)
	key = append(key, owner.Bytes()...)
	key = append(key, sdk.Uint64ToBigEndian(id)...)
	return key
}

func OwnerFromPositionKey(key []byte) (sdk.AccAddress, uint64, error) {
	if len(key) != sdk.AddrLen+8 {
		return nil, 0, fmt.Errorf("invalid owner position key length %d", len(key))
	}
	return sdk.AccAddress(key[:sdk.AddrLen]), sdk.BigEndianToUint64(key[sdk.AddrLen:]), nil
}
```

Test exact 20-byte owner plus 8-byte ID encoding and malformed lengths.

- [ ] **Step 2: Write failing index lifecycle tests.**

Add these tests:

```go
func TestSetPositionCreatesOwnerIndex(t *testing.T) {}
func TestSetPositionUpdateDoesNotDuplicateOwnerIndex(t *testing.T) {}
func TestSetPositionRejectsOwnerChange(t *testing.T) {}
func TestDeletePositionRemovesOwnerIndex(t *testing.T) {}
func TestInitGenesisRebuildsOwnerIndex(t *testing.T) {}
```

Inspect both primary and owner-prefix stores. Use two valid Terra account addresses.

- [ ] **Step 3: Make keeper writes maintain the index.**

Change `SetPosition` to return an error. On insert, write the primary record and owner index. On update, reject owner changes, remove any stale old index, and write exactly one current index entry. Change `DeletePosition` to remove both records. Update every caller and assert the returned error.

- [ ] **Step 4: Rebuild the index during genesis import.**

Clear only the owner-index prefix, then import validated positions through `SetPosition`. Do not serialize the derived index in protobuf genesis.

- [ ] **Step 5: Query only the requested owner prefix.**

Replace the global position scan with:

```go
owner, err := sdk.AccAddressFromBech32(req.Owner)
if err != nil {
	return nil, sdkerrors.ErrInvalidAddress.Wrapf("owner: %s", err)
}
index := prefix.NewStore(ctx.KVStore(q.storeKey), types.OwnerPositionKeyPrefix)
ownerStore := prefix.NewStore(index, owner.Bytes())
```

Paginate `ownerStore`, decode each 8-byte ID, load the primary record, and return corruption errors for missing or mismatched records. Preserve numeric ordering and SDK pagination semantics.

- [ ] **Step 6: Run focused tests and commit.**

```bash
go test ./x/ustcstaking/keeper ./x/ustcstaking -run 'Test(SetPosition|DeletePosition|InitGenesis|QueryPositionsByOwner)' -count=1
git add x/ustcstaking
git commit -m "feat: index USTC positions by owner"
```

Expected: PASS with deterministic owner-only pagination.

## Task 2: Harden params, genesis, and position-ID validation

**Files:**

- Modify: `x/ustcstaking/types/params.go`
- Modify: `x/ustcstaking/types/genesis.go`
- Modify: `x/ustcstaking/types/errors.go`
- Modify: `x/ustcstaking/keeper/msg_server.go`
- Create: `x/ustcstaking/keeper/genesis_validation.go`
- Test: `x/ustcstaking/types/params_test.go`
- Test: `x/ustcstaking/types/genesis_test.go`
- Test: `x/ustcstaking/keeper/msg_server_test.go`

- [ ] **Step 1: Add dedicated errors.**

Append stable error codes for position-ID exhaustion, malformed lock snapshots, and corrupt owner indexes. Do not renumber existing errors.

- [ ] **Step 2: Write failing validation tests.**

```go
func TestParamsValidateRejectsZeroLockTierID(t *testing.T) {}
func TestGenesisValidateRejectsNilOrZeroLockSnapshot(t *testing.T) {}
func TestGenesisValidateRejectsActiveMaturityTimestamp(t *testing.T) {}
func TestGenesisValidateRejectsUnbondingWithoutMaturity(t *testing.T) {}
func TestGenesisValidateRejectsWrongClaimableDenom(t *testing.T) {}
func TestGenesisValidateRejectsMaxPositionID(t *testing.T) {}
func TestStakeRejectsPositionIDExhaustionBeforeTransfer(t *testing.T) {}
```

Use valid owners and `uusd` in positive fixtures. Assert failed stake leaves the BankKeeper call count and store unchanged.

- [ ] **Step 3: Enforce lock-tier and snapshot rules.**

Reject parameter tier ID zero. Require every position to carry positive `LockDuration` and `ShareMultiplier`; do not require historical tier IDs to remain in current params. Require explicit non-negative `uusd` claimable rewards instead of accepting an empty denom.

- [ ] **Step 4: Enforce status-specific state.**

Implement these exact checks in `ValidateGenesis`:

```text
active: positive principal, positive shares, reward_debt >= 0, maturity == nil
unbonding: positive principal, shares == 0, reward_debt == 0, maturity != nil
withdrawn: principal == 0, shares == 0, reward_debt == 0
all: valid owner, uusd principal, uusd claimable rewards, non-negative values
```

Reject unknown status and nil arithmetic values that cannot normalize to valid zero. Preserve the active-share total check.

- [ ] **Step 5: Guard next-ID arithmetic.**

Require `next_position_id != 0`, every stored ID below `^uint64(0)`, and `next_position_id > maxID`. In `Stake`, reject the current ID `^uint64(0)` before `SendCoinsFromAccountToModule`.

- [ ] **Step 6: Add bank-backed imported-state validation.**

Create `Keeper.ValidateState(ctx)` for checks requiring balances. It runs principal-custody, active-share, reward-solvency, and unexpected-denom checks against real BankKeeper after genesis or upgrade initialization. Keep pure protobuf checks in `types.ValidateGenesis`.

- [ ] **Step 7: Run tests and commit.**

```bash
go test ./x/ustcstaking/types ./x/ustcstaking/keeper -run 'Test(Params|Genesis|Stake.*Exhaust|ValidateState)' -count=1
git add x/ustcstaking
git commit -m "fix: harden USTC imported state validation"
```

Expected: malformed fixtures fail before store or bank mutation.

## Task 3: Expose production accounting checks

**Files:**

- Modify: `x/ustcstaking/types/expected_keepers.go`
- Create: `x/ustcstaking/keeper/invariants.go`
- Create: `x/ustcstaking/keeper/invariants_test.go`
- Modify: `x/ustcstaking/module.go`
- Modify: `proto/terra/ustcstaking/v1/query.proto`
- Modify: `x/ustcstaking/keeper/query_server.go`
- Modify: `x/ustcstaking/client/cli/query.go`
- Modify: `x/ustcstaking/keeper/test_utils_test.go`

- [ ] **Step 1: Extend the narrow BankKeeper interface.**

Add `GetAllBalances(ctx, addr) sdk.Coins`. Update the recording keeper to expose principal and reward balances. Do not add AccountKeeper, StakingKeeper, DistributionKeeper, or MintKeeper to the USTC keeper interface.

- [ ] **Step 2: Write invariant tests first.**

```go
func TestPrincipalInvariantMatchesPool(t *testing.T) {}
func TestPrincipalInvariantReportsMismatch(t *testing.T) {}
func TestActiveSharesInvariantMatchesRewardState(t *testing.T) {}
func TestRewardSolvencyInvariantIncludesActiveAndClaimable(t *testing.T) {}
func TestRewardSolvencyInvariantAllowsRoundingDust(t *testing.T) {}
func TestInvariantRejectsUnexpectedPoolDenom(t *testing.T) {}
func TestRegisterInvariantsAddsAllRoutes(t *testing.T) {}
```

Assert route names exactly: `principal-custody`, `active-shares`, and `reward-solvency`.

- [ ] **Step 3: Implement invariant calculation.**

Add pure keeper methods returning the SDK invariant form `(string, bool)). Use `authtypes.NewModuleAddress` for both pools. Diagnostics include expected and actual values plus the first offending position ID.

- [ ] **Step 4: Expose the operator check.**

The app uses Cosmos SDK v0.53, where `sdk.InvariantRegistry` is deprecated,
`module.Manager.RegisterInvariants` is a no-op, and `x/crisis` is absent.
Do not restore crisis or introduce a per-block scan. Keep `RegisterInvariants`
for SDK-compatible tooling and expose the same accounting check through a
read-only `Query/ValidateState` and `terrad query ustcstaking validate-state`.
Integration tests execute the query with real application keepers against
both valid and deliberately corrupted state.

Do not add global supply conservation.

- [ ] **Step 5: Run and commit.**

```bash
go test ./x/ustcstaking/... -run 'Test(Principal|ActiveShares|RewardSolvency|RegisterInvariants|ValidateState)' -count=1
git add x/ustcstaking
git commit -m "feat: register USTC staking accounting invariants"
```

Expected: deliberate corrupt states produce deterministic diagnostics.

## Task 4: Add the live-chain `ustc_staking` upgrade

**Files:**

- Create: `app/upgrades/ustc_staking/constants.go`
- Create: `app/upgrades/ustc_staking/upgrades.go`
- Modify: `app/app.go`
- Create: `app/upgrades/ustc_staking/upgrade_test.go`
- Test: `x/ustcstaking/genesis_test.go`

- [ ] **Step 1: Write registration tests.**

Assert `app.Upgrades` contains one `ustc_staking` entry whose `StoreUpgrades.Added` contains `ustcstakingtypes.StoreKey`, with no deleted or renamed stores.

- [ ] **Step 2: Add the upgrade package.**

Follow `app/upgrades/v6` and `app/upgrades/v14_2`:

```go
const UpgradeName = "ustc_staking"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
	CreateUpgradeHandler: CreateUSTCStakingUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{ustcstakingtypes.StoreKey},
	},
}
```

The handler calls `mm.RunMigrations(ctx, cfg, fromVM)`. Before returning, call `keepers.AccountKeeper.GetModuleAccount` for both USTC module accounts and verify empty permission sets.

- [ ] **Step 3: Register imports and upgrade list.**

Import the package in `app/app.go` and append `ustc_staking.Upgrade` after `v14_2.Upgrade`. Preserve existing upgrade ordering and handler setup.

- [ ] **Step 4: Rehearse a new-module upgrade.**

Use a pre-upgrade version map without `ustcstaking`, run the actual handler with a mounted `x_ustcstaking` store, and assert:

```go
require.Equal(t, uint64(1), versionMap[ustcstakingtypes.ModuleName])
require.Equal(t, types.DefaultGenesisState(), exportedState)
require.Empty(t, principalPoolBalance)
require.Empty(t, rewardPoolBalance)
require.Empty(t, moduleAccount.GetPermissions())
```

Snapshot unrelated account balances, total `uusd` supply, and validator state before the handler and compare after it.

- [ ] **Step 5: Test wrong-plan and repeat behavior.**

Assert the upgrade keeper refuses an unregistered plan and a second application does not recreate or overwrite state.

- [ ] **Step 6: Run and commit.**

```bash
go test ./app ./app/upgrades/ustc_staking ./x/ustcstaking -run 'Test(USTCStaking|Upgrade|Genesis)' -count=1
git add app/app.go app/upgrades/ustc_staking x/ustcstaking
git commit -m "feat: add USTC staking chain upgrade"
```

Expected: PASS, including live-chain store addition and default state.

## Task 5: Add stable module events

**Files:**

- Create: `x/ustcstaking/types/events.go`
- Modify: `x/ustcstaking/keeper/msg_server.go`
- Test: `x/ustcstaking/keeper/msg_server_test.go`
- Test: `x/ustcstaking/keeper/rewards_test.go`
- Modify: `x/ustcstaking/spec/05_events.md`

- [ ] **Step 1: Define event constants.**

```go
const (
	EventTypeStake                  = "ustcstaking_stake"
	EventTypeBeginUnbonding         = "ustcstaking_begin_unbonding"
	EventTypeWithdraw               = "ustcstaking_withdraw"
	EventTypeClaimRewards           = "ustcstaking_claim_rewards"
	EventTypeFundRewards            = "ustcstaking_fund_rewards"
	EventTypeUpdateParams           = "ustcstaking_update_params"
	EventTypeUpdateFundingAuthority = "ustcstaking_update_funding_authority"
)
```

- [ ] **Step 2: Write event assertions before emission.**

Wrap unit contexts with `sdk.NewEventManager()`. For each successful message, assert exactly one matching event, required keys, canonical coin strings, decimal index strings, and UTC RFC3339Nano timestamps. Failed messages must produce no module-specific event.

- [ ] **Step 3: Emit only after successful state transitions.**

Emit at the end of each successful handler. Capture funding index before/after, old/new funding authorities, and final reward-pool balance. Do not emit from validation branches or before bank transfers.

- [ ] **Step 4: Update specification and commit.**

Replace the current “not yet emitted” wording in `x/ustcstaking/spec/05_events.md` with the final event table and rollback behavior.

```bash
go test ./x/ustcstaking/... -run 'Test.*Event' -count=1
git add x/ustcstaking
git commit -m "feat: emit USTC staking module events"
```

Expected: PASS with no events on rejected transactions.

## Task 6: Replace the Market simulation decoder

**Files:**

- Create: `x/ustcstaking/simulation/decoder.go`
- Create: `x/ustcstaking/simulation/decoder_test.go`
- Modify: `x/ustcstaking/module.go`

- [ ] **Step 1: Write decoder tests.**

Use `keeper.MakeTestCodec` and KV pairs for params, reward state, next ID, primary positions, and owner-index records. Assert output contains USTC type names and values. Unknown prefixes must return a deterministic diagnostic rather than Market decoding.

- [ ] **Step 2: Implement the decoder.**

Use `bytes.HasPrefix` against `PositionKeyPrefix`, `OwnerPositionKeyPrefix`, `RewardStateKey`, `ParamsKey`, and `NextPositionIDKey`. Unmarshal with the USTC codec, decode IDs big-endian, and return deterministic hex-key output for unknown records without mutating state.

- [ ] **Step 3: Register the correct decoder.**

Remove the `x/market/simulation` import from `x/ustcstaking/module.go` and register `ustcstaking/simulation.NewDecodeStore(am.cdc)`.

- [ ] **Step 4: Run and commit.**

```bash
go test ./x/ustcstaking/simulation ./x/ustcstaking -count=1
git add x/ustcstaking
git commit -m "fix: use USTC staking simulation decoder"
```

Expected: PASS with no Market simulation dependency from the USTC module.

## Task 7: Add real AccountKeeper/BankKeeper integration coverage

**Files:**

- Create: `x/ustcstaking/keeper/integration_test.go`
- Create: `app/ustc_staking_integration_test.go`
- Modify: `x/ustcstaking/keeper/test_utils_test.go`
- Modify: `tests/e2e/initialization/config.go` only if the genesis helper needs USTC defaults

- [ ] **Step 1: Build an app fixture with real keepers.**

Use the Terra app constructor and in-memory DB. Fund eight users with `uusd` and `uluna` fees. Retrieve module addresses from real AccountKeeper and assert both USTC accounts exist with empty permission sets.

- [ ] **Step 2: Exercise the reference lifecycle.**

Execute:

```text
configure two temporary tiers
stake users 1–8 with mixed tier amounts
fund rewards from the configured authority
claim from active positions
begin unbonding and claim from unbonding positions
advance block time and withdraw principals
repeat a zero-value claim
export and import state
```

After every action, invoke all three invariant routes and record principal pool, reward pool, active shares, claimable rewards, and total `uusd` supply.

- [ ] **Step 3: Add failure and rollback checks.**

Attempt wrong denom, wrong authority, paused stake/funding, third-party withdrawal, early withdrawal, duplicate withdrawal, insolvent claim, malformed owner, and no-active-share funding. Compare KV snapshots, module balances, and total supply before and after every rejected transaction.

- [ ] **Step 4: Prove validator isolation.**

Snapshot validator power, delegation, commission, slashing, and jailing state before the lifecycle. Assert byte-equivalent exported staking state afterward.

- [ ] **Step 5: Run and commit.**

```bash
go test ./x/ustcstaking/keeper ./app -run 'Test(USTCStaking|RealBank|ReferenceLifecycle|ValidatorIsolation)' -count=1
git add x/ustcstaking app tests/e2e/initialization
git commit -m "test: verify USTC staking with real app keepers"
```

Expected: PASS using real AccountKeeper/BankKeeper, not the recording mock.

## Task 8: Correct runbook and create release gates

**Files:**

- Modify: `docs/ustc-staking/phase-2-community-pool-localnet-testing.md`
- Create: `docs/ustc-staking/phase-1-release-readiness.md`
- Create: `docs/ustc-staking/phase-1-dependency-triage.md`

- [ ] **Step 1: Re-run documented CLI commands.**

Build `terrad` and execute the runbook on seven validators. Correct flags, governance JSON, query paths, expected events, pause/resume behavior, module-account checks, and cleanup steps. Record command exit status and expected JSON for each scenario.

- [ ] **Step 2: Document upgrade rehearsal and rollback gates.**

Add exact instructions for preparing an existing database, writing the `ustc_staking` plan, verifying store creation, checking module-account permissions, exporting state, and confirming unchanged LUNC validator state. A failed rehearsal blocks activation.

- [ ] **Step 3: Record dependency scan results.**

Run the repository's available vulnerability tools and record each advisory in `phase-1-dependency-triage.md` with:

```text
advisory ID | module/package | reachable path | severity | fixed version
| mitigation | release disposition | verification date
```

Classify an advisory as Phase 1-blocking only when affected code is reachable in the release binary and can affect custody, authorization, consensus, or upgrade safety. Do not change dependency versions in this plan.

- [ ] **Step 4: Create the readiness checklist.**

Require focused tests, real-keeper tests, upgrade rehearsal, repeated seven-validator campaign, fuzz/race/static checks, export validation, event review, query-resource review, dependency dispositions, independent review, and governance approval before activation.

- [ ] **Step 5: Commit documentation.**

```bash
git diff --check
git add docs/ustc-staking
git commit -m "docs: add USTC staking phase 1 release gates"
```

Expected: no whitespace errors.

## Task 9: Run complete verification and refresh local knowledge

**Files:**

- Modify only generated protobuf or documentation files produced by verification commands.

- [ ] **Step 1: Format and run focused suites.**

```bash
gofmt -w x/ustcstaking app/upgrades/ustc_staking app/ustc_staking_upgrade_test.go app/ustc_staking_integration_test.go
go test ./x/ustcstaking/... ./app/upgrades/ustc_staking ./app -count=1
go test -race ./x/ustcstaking/... -count=1
go vet ./x/ustcstaking/... ./app/upgrades/ustc_staking ./app
```

Expected: all tests and vet pass. A Go cache cleanup warning caused by a read-only sandbox is recorded separately if it occurs.

- [ ] **Step 2: Run repository verification.**

```bash
go test ./tests/e2e/... -count=1
go test ./...
```

Expected: all packages pass and no files under `custom/staking` change.

- [ ] **Step 3: Refresh and validate Graphify locally.**

```bash
scripts/graphify-refresh coding
python3 -m tools.graphify.validate graphify-out/coding --profile coding
git check-ignore graphify-out/coding/graph.json
```

Expected: current-commit graph, validation success, and ignored output.

- [ ] **Step 4: Inspect final diff boundaries.**

```bash
git status --short
git diff --stat origin/ustc_staking...HEAD
git diff --name-only origin/ustc_staking...HEAD | rg '^(x/ustcstaking|app/upgrades/ustc_staking|app/app.go|app/ustc_staking|docs/ustc-staking|tests/e2e)'
```

Expected: only approved paths; no `graphify-out/`, binaries, dependency lockfiles, or unrelated validator modules.

- [ ] **Step 5: Repeat validator campaign.**

Run the corrected seven-validator campaign against the release candidate. Require the previous functional PASS results plus successful upgrade rehearsal, invariant checks, event assertions, exported-state validation, and unchanged total `uusd` supply. Record revision, binary checksum, date, node count, and failed or skipped cases.

- [ ] **Step 6: Final commit.**

```bash
git add x/ustcstaking app app/upgrades/ustc_staking tests/e2e docs/ustc-staking
git commit -m "test: complete USTC staking phase 1 hardening"
```

## Completion criteria

The plan is complete only when the approved design's ten acceptance criteria are evidenced by code, automated output, upgrade rehearsal artifacts, and the repeated validator report. Passing unit tests alone is insufficient for public testnet or production readiness.

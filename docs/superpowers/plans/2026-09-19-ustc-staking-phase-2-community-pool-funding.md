# USTC Staking Phase 2 Community-Pool Funding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make native USTC staking rewards fundable only by a governance-approved debit from the distribution community pool while preserving direct, contract-free user staking.

**Architecture:** `MsgStake` remains a user-signed native message. `MsgFundRewards` becomes governance-authorized and calls a narrow application adapter that subtracts from distribution FeePool accounting and uses the bank keeper's module-to-module transfer into `ustcstaking_reward_pool`; only after that succeeds does the module update its reward index. Remove the separate funding authority and all Phase 2 Wasm assumptions while keeping the reward module account blocked from direct sends.

**Tech Stack:** Go, Cosmos SDK v0.53.6, protobuf/Buf generation, native `x/distribution`, native `x/ustcstaking`, testify, application keeper integration tests.

---

## Preconditions and non-negotiable boundaries

- [ ] Confirm `x/ustcstaking` has not been activated on production. If it has,
  stop this plan and write a versioned state-migration plan.
- [ ] Start from the tested Phase 1 `ustc_staking` revision, not from the
  superseded `phase2-treasury-manager` branch.
- [ ] Do not add `contracts/`, Wasm bindings, Stargate messages, POL, DEX,
  buyback, burn, or automated funding.
- [ ] Do not modify `custom/staking`, SDK validator staking, distribution
  rewards, mint, slashing, validator power, commission, or governance voting
  weight.
- [ ] Preserve direct user `MsgStake`, `MsgClaimRewards`,
  `MsgBeginUnbonding`, and `MsgWithdraw` behavior.
- [ ] Community-pool funding must be atomic with reward-index accounting and
  must never mint USTC.

## File map

| File/group | Responsibility |
|---|---|
| `proto/terra/ustcstaking/v1/{tx,ustcstaking}.proto` | governance funding API and removal of funding-authority state |
| `x/ustcstaking/types/{msgs,params,codec,events,expected_keepers}.go` | validation, registration, event names, and keeper interfaces |
| `x/ustcstaking/keeper/{keeper,rewards,msg_server}.go` | community-pool debit, reward-pool credit, authorization, and index update |
| `app/keepers/{community_pool_adapter,keepers}.go` | atomically debit FeePool accounting, move module funds, and inject the adapter |
| `app/ustc_staking_integration_test.go` | real BankKeeper and distribution FeePool accounting test |
| `x/ustcstaking/{spec,simulation}/` | protocol documentation and decoder compatibility |
| `docs/ustc-staking/` | governance proposal, reconciliation, and localnet runbook |

Tasks 2 through 5 form one cross-package API migration. Their focused tests
run at intermediate checkpoints, but they are committed together at the end of
Task 5 so no commit leaves generated APIs, keeper calls, and application
wiring inconsistent.

### Task 1: Establish the production-activation gate

**Files:**
- Create: `docs/ustc-staking/phase-2-community-pool-preflight.md`

- [ ] **Step 1: Record the deployment check**

Create the document with this required evidence table:

```markdown
# Phase 2 community-pool funding preflight

| Gate | Required evidence | Result |
|---|---|---|
| Production module activation | `terrad query ustcstaking params` is unavailable on production, or governance/upgrade records prove the module has not activated | Required before implementation |
| Base revision | tested Phase 1 commit hash | `3bdf007e` |
| Contract branch exclusion | `git merge-base --is-ancestor phase2-treasury-manager HEAD` exits non-zero | Required before implementation |
| Existing store version | no production `x/ustcstaking` store exists | Required before implementation |

Implementation may proceed only when every gate is PASS. A production store
requires a separately approved module-version migration.
```

- [ ] **Step 2: Verify the branch excludes contract work**

Run:

```bash
git merge-base --is-ancestor phase2-treasury-manager HEAD
```

Expected: non-zero exit status. Record `PASS`; do not merge or cherry-pick the
contract branch.

- [ ] **Step 3: Commit the preflight record**

```bash
git add docs/ustc-staking/phase-2-community-pool-preflight.md
git commit -m "docs: record native phase 2 preflight"
```

### Task 2: Replace funding-authority protobuf and parameter APIs

**Files:**
- Modify: `proto/terra/ustcstaking/v1/tx.proto`
- Modify: `proto/terra/ustcstaking/v1/ustcstaking.proto`
- Modify: `x/ustcstaking/types/{msgs,params,codec}.go`
- Create: `x/ustcstaking/types/msgs_test.go`
- Modify: `x/ustcstaking/types/params_test.go`
- Regenerate: `x/ustcstaking/types/{tx,ustcstaking}.pb.go`

- [ ] **Step 1: Write failing message and parameter tests**

Add tests proving the governance authority is the funding signer and that
params contain no independently rotatable funding authority:

```go
func TestMsgFundRewardsUsesGovernanceAuthoritySigner(t *testing.T) {
    authority := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
    msg := MsgFundRewards{
        Authority: authority.String(),
        Amount:    sdk.NewInt64Coin(BondDenom, 42),
    }
    require.NoError(t, msg.ValidateBasic())
    require.Equal(t, []sdk.AccAddress{authority}, msg.GetSigners())
}

func TestDefaultParamsUseOnlyGovernanceAuthority(t *testing.T) {
    params := DefaultParams()
    require.Equal(t, authtypes.NewModuleAddress(govtypes.ModuleName).String(), params.Authority)
    require.NoError(t, params.Validate())
}
```

- [ ] **Step 2: Run the focused tests and confirm failure**

Run:

```bash
go test ./x/ustcstaking/types -run 'TestMsgFundRewardsUsesGovernanceAuthoritySigner|TestDefaultParamsUseOnlyGovernanceAuthority' -count=1
```

Expected: compile failure because `MsgFundRewards.Authority` does not exist.

- [ ] **Step 3: Change the protobuf source**

Use this funding message and remove the `UpdateFundingAuthority` RPC and its
request/response messages:

```protobuf
message MsgFundRewards {
  option (cosmos.msg.v1.signer) = "authority";
  option (gogoproto.equal) = true;
  option (gogoproto.goproto_getters) = false;
  string authority = 1 [(cosmos_proto.scalar) = "cosmos.AddressString", (gogoproto.moretags) = "yaml:\"authority\""];
  cosmos.base.v1beta1.Coin amount = 2 [(gogoproto.nullable) = false, (gogoproto.moretags) = "yaml:\"amount\""];
}
```

Change `Params` to reserve the removed field permanently:

```protobuf
message Params {
  option (gogoproto.equal) = true;

  reserved 4;
  reserved "funding_authority";

  string bond_denom = 1 [(gogoproto.moretags) = "yaml:\"bond_denom\""];
  repeated LockTier lock_tiers = 2 [(gogoproto.nullable) = false, (gogoproto.moretags) = "yaml:\"lock_tiers\""];
  string authority = 3 [(cosmos_proto.scalar) = "cosmos.AddressString", (gogoproto.moretags) = "yaml:\"authority\""];
  bool paused = 5 [(gogoproto.moretags) = "yaml:\"paused\""];
}
```

- [ ] **Step 4: Regenerate protobuf output**

Run:

```bash
make proto-gen
```

Expected: generated `tx.pb.go` and `ustcstaking.pb.go` expose
`MsgFundRewards.Authority`; no generated `MsgUpdateFundingAuthority` remains.

- [ ] **Step 5: Update handwritten message and parameter code**

In `msgs.go`, retain `MsgFundRewards` and remove every
`MsgUpdateFundingAuthority` assertion, route, type, signer, sign-bytes, and
validation method. Implement funding validation as:

```go
func (msg MsgFundRewards) GetSigners() []sdk.AccAddress { return signer(msg.Authority) }

func (msg MsgFundRewards) ValidateBasic() error {
    if err := validateSigner(msg.Authority); err != nil {
        return err
    }
    return validateAmount(msg.Amount)
}
```

In `params.go`, remove `FundingAuthority` initialization and validation:

```go
func DefaultParams() Params {
    return Params{
        BondDenom: BondDenom,
        LockTiers: []LockTier{},
        Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
        Paused: false,
    }
}
```

Remove the obsolete Amino message registration from `codec.go`.

- [ ] **Step 6: Run the type checkpoint**

```bash
go test ./x/ustcstaking/types -count=1
```

Expected: all `x/ustcstaking/types` tests pass. Do not commit yet; keeper and
application call sites are completed in Tasks 3 through 5.

### Task 3: Add the narrow distribution keeper dependency

**Files:**
- Modify: `x/ustcstaking/types/expected_keepers.go`
- Modify: `x/ustcstaking/keeper/keeper.go`
- Modify: `x/ustcstaking/keeper/test_utils_test.go`
- Modify: `x/ustcstaking/keeper/rewards.go`
- Modify: `x/ustcstaking/keeper/rewards_test.go`

- [ ] **Step 1: Write failing community-pool funding tests**

Add a fake that records calls and can return an error:

```go
type recordingCommunityPoolKeeper struct {
    calls     int
    amount    sdk.Coins
    recipientModule string
    err       error
    bank      *recordingBankKeeper
}

func (d *recordingCommunityPoolKeeper) DistributeFromCommunityPoolToModule(
    _ context.Context,
    amount sdk.Coins,
    recipientModule string,
) error {
    d.calls++
    d.amount = amount
    d.recipientModule = recipientModule
    if d.err != nil {
        return d.err
    }
    if d.bank != nil {
        d.bank.rewardPoolBalance = sdk.NewCoin(types.BondDenom, amount.AmountOf(types.BondDenom))
    }
    return nil
}
```

Add these tests:

```go
func TestFundRewardsDebitsCommunityPoolToRewardPool(t *testing.T) {
    ctx, k := newKeeperTest(t)
    k.SetRewardState(ctx, types.RewardState{
        RewardIndex: math.LegacyZeroDec(),
        TotalShares: math.NewInt(100),
    })

    err := k.FundRewards(ctx, sdk.NewInt64Coin(types.BondDenom, 40))

    require.NoError(t, err)
    communityPool := k.communityPoolKeeper.(*recordingCommunityPoolKeeper)
    require.Equal(t, 1, communityPool.calls)
    require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 40)), communityPool.amount)
    require.Equal(t, types.RewardPoolName, communityPool.recipientModule)
    require.Equal(t, math.LegacyMustNewDecFromStr("0.4"), k.GetRewardState(ctx).RewardIndex)
}

func TestFundRewardsLeavesIndexUnchangedWhenCommunityPoolFails(t *testing.T) {
    ctx, k := newKeeperTest(t)
    initial := types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.NewInt(100)}
    k.SetRewardState(ctx, initial)
    k.communityPoolKeeper.(*recordingCommunityPoolKeeper).err = errors.New("insufficient community pool")

    err := k.FundRewards(ctx, sdk.NewInt64Coin(types.BondDenom, 40))

    require.Error(t, err)
    require.Equal(t, initial, k.GetRewardState(ctx))
}
```

- [ ] **Step 2: Run tests and confirm failure**

```bash
go test ./x/ustcstaking/keeper -run 'TestFundRewardsDebitsCommunityPoolToRewardPool|TestFundRewardsLeavesIndexUnchangedWhenCommunityPoolFails' -count=1
```

Expected: compile failure because the keeper lacks `communityPoolKeeper` and
`FundRewards` still takes a sender.

- [ ] **Step 3: Define and inject the interface**

Add to `expected_keepers.go`:

```go
type CommunityPoolKeeper interface {
    DistributeFromCommunityPoolToModule(ctx context.Context, amount sdk.Coins, recipientModule string) error
}
```

Change the keeper constructor:

```go
type Keeper struct {
    cdc                 codec.BinaryCodec
    storeKey            storetypes.StoreKey
    bankKeeper          types.BankKeeper
    communityPoolKeeper types.CommunityPoolKeeper
}

func NewKeeper(
    cdc codec.BinaryCodec,
    storeKey storetypes.StoreKey,
    bankKeeper types.BankKeeper,
    communityPoolKeeper types.CommunityPoolKeeper,
) Keeper {
    return Keeper{
        cdc: cdc,
        storeKey: storeKey,
        bankKeeper: bankKeeper,
        communityPoolKeeper: communityPoolKeeper,
    }
}
```

Update every test constructor to pass `&recordingCommunityPoolKeeper{}`. Give
that fake the same `recordingBankKeeper` pointer so successful fake funding
updates the reward-pool balance observed by event tests.

- [ ] **Step 4: Replace account-funded reward movement**

Implement:

```go
func (k Keeper) FundRewards(ctx sdk.Context, amount sdk.Coin) error {
    if amount.Denom != types.BondDenom || amount.Amount.IsNil() || !amount.Amount.IsPositive() {
        return types.ErrInvalidDenom.Wrapf("expected positive %s funding amount", types.BondDenom)
    }

    state := k.GetRewardState(ctx)
    if state.TotalShares.IsNil() || !state.TotalShares.IsPositive() {
        return types.ErrNoActiveShares
    }

    if err := k.communityPoolKeeper.DistributeFromCommunityPoolToModule(
        ctx,
        sdk.NewCoins(amount),
        types.RewardPoolName,
    ); err != nil {
        return err
    }

    index := state.RewardIndex
    if index.IsNil() {
        index = math.LegacyZeroDec()
    }
    state.RewardIndex = index.Add(amount.Amount.ToLegacyDec().QuoInt(state.TotalShares))
    k.SetRewardState(ctx, state)
    return nil
}
```

Delete the account-to-module transfer from this method.

- [ ] **Step 5: Format and inspect the keeper checkpoint**

```bash
gofmt -w x/ustcstaking/types/expected_keepers.go x/ustcstaking/keeper/keeper.go x/ustcstaking/keeper/rewards.go x/ustcstaking/keeper/rewards_test.go x/ustcstaking/keeper/test_utils_test.go
git diff --check
```

Expected: formatting and diff checks pass. The keeper package is expected to
remain uncompilable until Task 4 updates the message server to the new API; do
not commit this intermediate state.

### Task 4: Enforce governance authorization and reconciliation events

**Files:**
- Modify: `x/ustcstaking/keeper/msg_server.go`
- Modify: `x/ustcstaking/keeper/msg_server_test.go`
- Modify: `x/ustcstaking/types/events.go`

- [ ] **Step 1: Write failing authorization and event tests**

Add:

```go
func TestFundRewardsRequiresGovernanceAuthority(t *testing.T) {
    ctx, k := newKeeperTest(t)
    authority := testAddress()
    configureKeeper(t, ctx, k, authority)
    k.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.NewInt(100)})

    _, err := NewMsgServerImpl(k).FundRewards(sdk.WrapSDKContext(ctx), &types.MsgFundRewards{
        Authority: testAddress(),
        Amount: sdk.NewInt64Coin(types.BondDenom, 40),
    })

    require.ErrorIs(t, err, types.ErrUnauthorized)
    require.Zero(t, k.communityPoolKeeper.(*recordingCommunityPoolKeeper).calls)
}
```

Also add `TestFundRewardsEventIdentifiesCommunityPool`. Configure 100 active
shares, fund 40 `uusd`, and assert `authority`, `source=community_pool`,
`amount=40uusd`, indexes `0` and `0.4`, and reward-pool balances before and
after. Use the existing `findModuleEvent` and `eventAttribute` helpers.

- [ ] **Step 2: Run tests and confirm failure**

```bash
go test ./x/ustcstaking/keeper -run 'TestFundRewardsRequiresGovernanceAuthority|TestFundRewardsEventIdentifiesCommunityPool' -count=1
```

Expected: compile or assertion failure because the server still reads
`FundingAuthority` and `Sender`.

- [ ] **Step 3: Replace the message handler**

Use this control flow:

```go
func (k msgServer) FundRewards(goCtx context.Context, msg *types.MsgFundRewards) (*types.MsgFundRewardsResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)
    params := k.GetParams(ctx)
    if params.Paused {
        return nil, types.ErrModulePaused
    }
    if msg.Authority != params.Authority {
        return nil, types.ErrUnauthorized
    }

    beforeIndex := k.GetRewardState(ctx).RewardIndex
    rewardPool := authtypes.NewModuleAddress(types.RewardPoolName)
    beforeBalance := k.bankKeeper.GetBalance(ctx, rewardPool, types.BondDenom)
    if err := k.Keeper.FundRewards(ctx, msg.Amount); err != nil {
        return nil, err
    }
    afterIndex := k.GetRewardState(ctx).RewardIndex
    afterBalance := k.bankKeeper.GetBalance(ctx, rewardPool, types.BondDenom)

    ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeFundRewards,
        sdk.NewAttribute("authority", msg.Authority),
        sdk.NewAttribute("source", "community_pool"),
        sdk.NewAttribute("amount", msg.Amount.String()),
        sdk.NewAttribute("reward_index_before", beforeIndex.String()),
        sdk.NewAttribute("reward_index_after", afterIndex.String()),
        sdk.NewAttribute("reward_pool_balance_before", beforeBalance.String()),
        sdk.NewAttribute("reward_pool_balance_after", afterBalance.String()),
    ))
    return &types.MsgFundRewardsResponse{Amount: msg.Amount}, nil
}
```

Delete `UpdateFundingAuthority` and remove
`EventTypeUpdateFundingAuthority`.

Update `configureKeeper` and every keeper test fixture to omit
`FundingAuthority`. Replace all `MsgFundRewards{Sender: ...}` literals with
`MsgFundRewards{Authority: ...}` and remove update-funding-authority tests.
Verify the cleanup with:

```bash
rg -n 'FundingAuthority|MsgUpdateFundingAuthority|\.Sender' x/ustcstaking/keeper x/ustcstaking/types
```

Expected: no obsolete funding-authority or funding-sender references.

- [ ] **Step 4: Run the keeper checkpoint**

```bash
go test ./x/ustcstaking/keeper -count=1
```

Expected: all keeper tests pass and failed funding emits no success event. Do
not commit until application wiring is complete in Task 5.

### Task 5: Wire the real distribution keeper and prove supply conservation

**Files:**
- Create: `app/keepers/community_pool_adapter.go`
- Modify: `app/keepers/keepers.go`
- Modify: `app/ustc_staking_integration_test.go`
- Modify: `x/ustcstaking/genesis_test.go`

- [ ] **Step 1: Rewrite the real-keeper integration test first**

In `TestUSTCStakingUsesRealAppKeepers`:

1. Leave direct owner staking unchanged.
2. Fund the distribution module account with `40uusd` using
   `banktestutil.FundModuleAccount`.
3. Set `DistrKeeper.FeePool.CommunityPool` to `40uusd` as decimal coins.
4. Execute `MsgFundRewards` with the governance module address as authority.
5. Assert the community pool and distribution module bank balance decrease by
   40, reward pool increases by 40, the owner claims 40, principal remains
   100, and total USTC supply is unchanged.
6. Assert `BankKeeper.BlockedAddr(rewardModule.GetAddress())` is still true so
   ordinary accounts cannot bypass governance and reward-index accounting.
7. Snapshot total USTC supply after test-only community-pool setup and before
   staking/funding actions; compare against that snapshot at the end.

The critical setup and execution are:

```go
authority := authtypes.NewModuleAddress(govtypes.ModuleName)
funding := types.NewInt64Coin(ustcstakingtypes.BondDenom, 40)
require.NoError(t, banktestutil.FundModuleAccount(
    suite.Ctx,
    suite.App.BankKeeper,
    distrtypes.ModuleName,
    types.NewCoins(funding),
))
feePool, err := suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
require.NoError(t, err)
feePool.CommunityPool = types.NewDecCoinsFromCoins(funding)
require.NoError(t, suite.App.DistrKeeper.FeePool.Set(suite.Ctx, feePool))
initialSupply := suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom)

_, err = msgServer.FundRewards(suite.Ctx, &ustcstakingtypes.MsgFundRewards{
    Authority: authority.String(),
    Amount: funding,
})
require.NoError(t, err)
```

- [ ] **Step 2: Run the integration test and confirm failure**

```bash
go test ./app -run TestUSTCStakingUsesRealAppKeepers -count=1
```

Expected: compile failure because the application community-pool adapter does
not exist yet.

- [ ] **Step 3: Implement the module-to-module community-pool adapter**

Create `app/keepers/community_pool_adapter.go`:

```go
package keepers

import (
    "context"

    sdk "github.com/cosmos/cosmos-sdk/types"
    distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
    distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

type moduleBankKeeper interface {
    SendCoinsFromModuleToModule(
        ctx context.Context,
        senderModule string,
        recipientModule string,
        amount sdk.Coins,
    ) error
}

type CommunityPoolAdapter struct {
    distributionKeeper distrkeeper.Keeper
    bankKeeper         moduleBankKeeper
}

func NewCommunityPoolAdapter(
    distributionKeeper distrkeeper.Keeper,
    bankKeeper moduleBankKeeper,
) CommunityPoolAdapter {
    return CommunityPoolAdapter{
        distributionKeeper: distributionKeeper,
        bankKeeper: bankKeeper,
    }
}

func (a CommunityPoolAdapter) DistributeFromCommunityPoolToModule(
    ctx context.Context,
    amount sdk.Coins,
    recipientModule string,
) error {
    feePool, err := a.distributionKeeper.FeePool.Get(ctx)
    if err != nil {
        return err
    }
    updated, negative := feePool.CommunityPool.SafeSub(sdk.NewDecCoinsFromCoins(amount...))
    if negative {
        return distrtypes.ErrBadDistribution
    }
    if err := a.bankKeeper.SendCoinsFromModuleToModule(
        ctx,
        distrtypes.ModuleName,
        recipientModule,
        amount,
    ); err != nil {
        return err
    }
    feePool.CommunityPool = updated
    return a.distributionKeeper.FeePool.Set(ctx, feePool)
}
```

Do not add `ustcstaking_reward_pool` to `allowedReceivingModAcc`. The adapter
uses module-to-module transfer specifically to preserve the direct-send
blocklist.

- [ ] **Step 4: Reorder keeper construction**

Remove the current USTC staking keeper construction immediately after
BankKeeper. Recreate it immediately after `DistrKeeper`:

```go
appKeepers.UstcStakingKeeper = ustcstakingkeeper.NewKeeper(
    appCodec,
    appKeepers.keys[ustcstakingtypes.StoreKey],
    appKeepers.BankKeeper,
    NewCommunityPoolAdapter(appKeepers.DistrKeeper, appKeepers.BankKeeper),
)
```

Do not pass `StakingKeeper` into `x/ustcstaking`.

- [ ] **Step 5: Update isolated constructors**

Update `x/ustcstaking/genesis_test.go` and all compile errors from
`NewKeeper(...)` to pass a fake `CommunityPoolKeeper`. The fake must panic if
called in tests unrelated to funding so accidental coupling is visible.

- [ ] **Step 6: Run integration and app tests, then commit the atomic migration**

```bash
go test ./app -run 'TestUSTCStakingUsesRealAppKeepers|TestUSTCStakingUpgrade' -count=1 -p 1
go test ./x/ustcstaking/... -count=1 -p 1
git add proto/terra/ustcstaking/v1 x/ustcstaking app/keepers/community_pool_adapter.go app/keepers/keepers.go app/ustc_staking_integration_test.go
git commit -m "feat: fund USTC staking from the community pool"
```

Expected: tests pass and the integration test proves unchanged total supply.

### Task 6: Remove obsolete authority state from genesis and simulation surfaces

**Files:**
- Modify: `x/ustcstaking/types/{genesis,params}_test.go`
- Modify: `x/ustcstaking/genesis_test.go`
- Modify: `x/ustcstaking/simulation/decoder_test.go`
- Modify: `x/ustcstaking/spec/{01_concepts,02_state,04_messages,05_events,06_params}.md`

- [ ] **Step 1: Add serialization assertions**

Add a genesis JSON round-trip test that marshals default genesis and asserts:

```go
require.NotContains(t, string(encoded), "funding_authority")
require.Contains(t, string(encoded), "authority")
```

Update simulation decoder fixtures so they contain `authority` and never
contain `sender` for `MsgFundRewards`. Remove fixtures for
`MsgUpdateFundingAuthority`.

- [ ] **Step 2: Run tests and confirm stale expectations fail**

```bash
go test ./x/ustcstaking/types ./x/ustcstaking/simulation -count=1
```

Expected before cleanup: failures referencing removed funding-authority fields
or message types.

- [ ] **Step 3: Update protocol specifications**

Document these exact rules:

- direct user staking uses `MsgStake` and the principal pool;
- governance funding uses `MsgFundRewards` and the community pool;
- the application adapter is the only community-pool debit path and uses
  `SendCoinsFromModuleToModule` while preserving the reward-pool blocklist;
- events identify `source=community_pool`;
- no funding authority, funding account, contract, mint, or validator effect.

- [ ] **Step 4: Run tests and commit**

```bash
go test ./x/ustcstaking/types ./x/ustcstaking/simulation -count=1
git add x/ustcstaking/types x/ustcstaking/simulation x/ustcstaking/spec
git commit -m "docs: align USTC protocol with governance funding"
```

### Task 7: Add governance and localnet operational coverage

**Files:**
- Create: `docs/ustc-staking/phase-2-community-pool-funding.md`
- Modify: `docs/ustc-staking/phase-2-community-pool-localnet-testing.md`
- Modify: `docs/ustc-staking/phase-1-release-readiness.md`
- Modify: `docs/ustc-staking/briefings/{technical-advisor-briefing,c-suite-briefing}.md`

- [ ] **Step 1: Write the governance message template**

Document this proposal message body:

```json
{
  "@type": "/terra.ustcstaking.v1.MsgFundRewards",
  "authority": "$GOVERNANCE_MODULE_ADDRESS",
  "amount": {
    "denom": "uusd",
    "amount": "$APPROVED_MICRO_USTC"
  }
}
```

The runbook must require operators to derive the governance module address from
the running binary rather than copying an address from another network.

- [ ] **Step 2: Define reconciliation evidence**

Require one evidence row per proposal with:

```text
proposal_id, execution_height, approved_uusd,
community_pool_before, community_pool_after,
reward_pool_before, reward_pool_after,
reward_index_before, reward_index_after, tx_or_proposal_event
```

The pass condition is:

```text
community_pool_before - community_pool_after = approved_uusd
reward_pool_after - reward_pool_before = approved_uusd
event.amount = approved_uusd
total_uusd_supply_after = total_uusd_supply_before
```

- [ ] **Step 3: Replace obsolete localnet funding instructions**

Remove instructions that sign `MsgFundRewards` with a funding account. Add a
governance proposal test followed by direct user staking, claim, unbond, and
withdraw. Include negative proposals for wrong denom, zero amount, no active
shares, insufficient community pool, and paused state.

- [ ] **Step 4: Update stakeholder briefings**

State that Phase 2 is native governance funding, TreasuryManager/POL is not in
scope, and users never pass principal through a contract.

- [ ] **Step 5: Commit operational documentation**

```bash
git add docs/ustc-staking
git commit -m "docs: add community-pool funding runbook"
```

### Task 8: Verify upgrade safety and protected boundaries

**Files:**
- Test: `app/upgrades/ustc_staking/upgrade_test.go`
- Test: `app/ustc_staking_upgrade_test.go`

- [ ] **Step 1: Prove this remains a pre-activation version-1 module**

Run:

```bash
go test ./app/upgrades/ustc_staking ./app -run 'USTCStakingUpgrade|TestUSTCStakingUpgrade' -count=1 -p 1
```

Expected: version 1 upgrade tests pass. If production activation evidence from
Task 1 is not PASS, stop instead of changing these tests.

- [ ] **Step 2: Prove protected modules are untouched**

Run:

```bash
git diff --name-only ustc_staking...HEAD -- custom/staking x/staking x/distribution x/mint x/slashing
```

Expected: no output. Using `DistrKeeper` through its public interface does not
require modifying `x/distribution`.

- [ ] **Step 3: Run formatting and focused verification**

```bash
gofmt -w x/ustcstaking app/keepers/keepers.go app/ustc_staking_integration_test.go
git diff --check
go test ./x/ustcstaking/... ./app/upgrades/ustc_staking ./app -count=1 -p 1
```

Expected: all commands pass.

- [ ] **Step 4: Run repository-wide verification**

```bash
GOMAXPROCS=2 go test ./... -count=1 -p 1
make lint
```

Expected: both commands pass. If an infrastructure limit interrupts either
command, preserve the output and do not mark the release gate complete.

- [ ] **Step 5: Commit any test-only assertion updates**

```bash
git add app/upgrades/ustc_staking/upgrade_test.go app/ustc_staking_upgrade_test.go
git diff --cached --quiet || git commit -m "test: verify native USTC funding upgrade boundaries"
```

### Task 9: Independent review and release gate

**Files:**
- Create: `docs/ustc-staking/phase-2-community-pool-release-readiness.md`

- [ ] **Step 1: Record technical review evidence**

Require sign-off for:

- governance authority validation;
- exact `FeePool.SafeSub` accounting and distribution-to-reward-pool
  module-to-module bank movement;
- continued blocking of direct account sends to the reward pool;
- community-pool decimal accounting;
- atomic rollback on every failure path;
- principal/reward module-account separation;
- reward-index arithmetic and dust;
- unchanged direct staking and withdrawal behavior;
- absence of mint and validator-state changes.

- [ ] **Step 2: Record localnet campaign evidence**

Require a multi-validator campaign covering successful governance funding,
all negative funding cases, direct staking, claim, pause/resume, unbonding,
withdrawal, node restart, full-network restart, invariant checks, and unchanged
total USTC supply.

- [ ] **Step 3: Commit release gates**

```bash
git add docs/ustc-staking/phase-2-community-pool-release-readiness.md
git commit -m "docs: define native phase 2 release gates"
```

## Completion criteria

Phase 2 is implementation-complete only when:

1. Users stake directly through native `MsgStake` with no contract path.
2. Governance is the sole funding authority.
3. Funding atomically decreases the community pool and increases the native
   reward pool by the same `uusd` amount.
4. Failed funding changes no balances or reward state.
5. Total USTC supply remains unchanged.
6. No funding-authority parameter or rotation message remains.
7. Protected validator and monetary modules are untouched.
8. Focused and repository-wide tests, lint, localnet rehearsal, and independent
   review all pass.

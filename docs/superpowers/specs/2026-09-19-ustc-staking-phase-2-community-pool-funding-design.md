# USTC Staking Phase 2 Community-Pool Funding Design

## Status and supersession

This design replaces the contract-based Phase 2 funding design dated
2026-07-29. Phase 2 no longer includes TreasuryManager, POL, DEX revenue
routing, Wasm dispatch, or a contract funding authority.

This design assumes `x/ustcstaking` has not yet been activated on a production
chain. If Phase 1 is activated first, implementation must stop until a separate
module-version and live-state migration design is approved.

## Decision

USTC staking remains entirely native. Users submit `MsgStake` directly from
their own accounts to `x/ustcstaking`, just as users submit ordinary chain
transactions for LUNC staking. No contract receives user principal or acts on
behalf of a staker.

Rewards are funded only by governance-approved transfers of existing `uusd`
from the distribution community pool. Governance executes native
`MsgFundRewards`; `x/ustcstaking` authorizes the governance module account,
asks the distribution keeper to debit the community pool and transfer the
approved amount to `ustcstaking_reward_pool`, then updates the reward index in
the same cached SDK message execution.

```text
user wallet
  -> MsgStake(owner, uusd, lock tier)
  -> ustcstaking principal pool

governance proposal
  -> MsgFundRewards(authority, uusd)
  -> distribution community pool debit
  -> ustcstaking reward pool credit
  -> reward-index update
```

## User staking model

The user path already established by Phase 1 remains unchanged:

- The owner signs `MsgStake` directly.
- BankKeeper transfers the owner's `uusd` to the native principal module
  account.
- The module creates a position with an immutable lock-tier snapshot.
- The owner later submits native messages to claim, begin unbonding, and
  withdraw.
- Wallets can expose these messages without deploying or calling CosmWasm.

"Like LUNC staking" describes the transaction experience, custody, and native
query integration. It does not make USTC a second consensus bond denomination.
USTC positions do not select validators, alter validator power, earn validator
commission, participate in slashing, or add governance voting weight.

## Governance funding API

`MsgFundRewards` remains the single funding API, but its authority and source
semantics change:

```protobuf
message MsgFundRewards {
  option (cosmos.msg.v1.signer) = "authority";
  string authority = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  cosmos.base.v1beta1.Coin amount = 2 [(gogoproto.nullable) = false];
}
```

The existing field number 1 is retained while the JSON/protobuf field name
changes from `sender` to `authority`. `authority` must equal the module's
governance authority, normally the governance module account. The amount must
be positive `uusd`.

The `funding_authority` parameter and `MsgUpdateFundingAuthority` are removed.
Protobuf field number 4 and the name `funding_authority` are reserved in
`Params` to prevent accidental reuse.

There is no ordinary-user CLI command for funding. Governance tooling composes
`MsgFundRewards` inside a proposal. Direct user staking CLI commands remain
available.

## Atomic keeper operation

The USTC staking keeper receives a narrow distribution interface:

```go
type DistributionKeeper interface {
    DistributeFromFeePool(
        ctx context.Context,
        amount sdk.Coins,
        recipient sdk.AccAddress,
    ) error
}
```

Funding executes in this order:

1. Reject when the module is paused.
2. Verify `msg.authority == params.authority`.
3. Require exactly one positive `uusd` coin.
4. Require positive active `total_shares`.
5. Snapshot the reward index and reward-pool balance for the event.
6. Call `DistributeFromFeePool` with the reward module-account address.
7. Increase the cumulative reward index by `amount / total_shares`.
8. Persist reward state and emit the funding event.

Cosmos SDK message execution uses a cached multistore. An error from community
pool validation, bank transfer, or message handling discards all state changes.
The reward index is not changed before a successful community-pool transfer.

`DistributeFromFeePool` is used instead of directly moving distribution module
coins because the community pool is accounting state inside the distribution
module, not a standalone bank account. The SDK method updates both its decimal
community-pool ledger and the distribution module's bank balance.

## Funding event

Successful funding emits `ustcstaking_fund_rewards` with:

- `authority`
- `source=community_pool`
- `amount`
- `reward_index_before`
- `reward_index_after`
- `reward_pool_balance_before`
- `reward_pool_balance_after`

The governance proposal, distribution community-pool delta, event amount, and
reward-pool delta must reconcile exactly. No event is emitted on failure.

## Failure behavior

- Wrong authority: reject before touching distribution or USTC staking state.
- Non-USTC, zero, negative, or multi-coin input: reject.
- No active shares: reject; community-pool funds remain untouched.
- Insufficient community pool: propagate the distribution error; reward pool
  and reward index remain unchanged.
- Paused module: reject new stakes and reward funding while preserving claims,
  matured withdrawals, and existing custody guarantees.
- Reward-pool receive failure: the entire message rolls back.

## Keeper construction

`x/ustcstaking` currently constructs before the distribution keeper and only
depends on BankKeeper. Phase 2 moves USTC staking keeper construction to after
the distribution keeper and injects the narrow interface above. No dependency
on the SDK staking keeper is added.

The two USTC module accounts remain unchanged:

- `ustcstaking` holds user principal.
- `ustcstaking_reward_pool` holds only funded rewards.

Neither module account receives mint or burner permissions.

## Compatibility and rollout

This is a pre-activation API correction:

- Phase 1 position, reward, and queue state formats remain unchanged.
- `Params.funding_authority` is removed before production activation.
- Generated protobuf, Amino registration, simulation decoding, genesis
  fixtures, CLI documentation, and tests are regenerated or updated together.
- The Phase 1 upgrade handler continues adding the module at version 1 because
  there is no live version-1 state to migrate under this assumption.

If a production chain activates the existing Phase 1 schema first, this plan
must not be applied as written. A new consensus version, migration handler,
backward-compatibility analysis, and upgrade rehearsal become mandatory.

## Security boundaries

- No contract, Stargate dispatch, or arbitrary funding account participates.
- Only governance can authorize a community-pool debit.
- Funding cannot mint USTC or draw from user principal.
- Community-pool accounting and bank movement use the SDK distribution keeper.
- Claims remain limited by the native reward-pool balance.
- USTC staking never changes LUNC validator state, distribution rewards,
  slashing, jailing, commission, or voting power.

## Required tests

1. Direct user staking transfers `uusd` from the signer to the principal pool
   without a contract.
2. Governance funding reduces community-pool accounting and its backing bank
   balance by the exact amount.
3. The reward pool increases by the same amount and the reward index changes
   deterministically.
4. A staker can claim the funded reward and total USTC supply is unchanged.
5. Wrong authority, wrong denom, zero amount, no active shares, paused state,
   and insufficient community pool all leave every involved balance and the
   reward index unchanged.
6. Genesis import/export contains no funding authority.
7. Simulation decoding and interface registration contain no removed message.
8. Existing stake, unbond, withdraw, claim, invariant, restart, and upgrade
   tests continue to pass.

## Operational governance flow

1. Query active shares, reward state, reward-pool balance, and community-pool
   `uusd` balance.
2. Submit a governance proposal containing one `MsgFundRewards` with the
   governance module authority and approved amount.
3. After proposal execution, reconcile the proposal amount, community-pool
   decrease, reward-pool increase, and funding event.
4. Halt further proposals and investigate if any value differs.

No recurring automatic distribution or promised APR is introduced. Each
funding tranche requires an explicit governance decision.

## Out of scope

- TreasuryManager, POL, DEX fees, buyback, burn, or revenue splitting.
- CosmWasm artifacts, migration, checksums, or Wasm message routing.
- Permissionless donations to the reward pool.
- Validator selection, validator incentives, USTC self-bond, consensus power,
  slashing, or commission.
- Automatic reward schedules or yield guarantees.

<!--
order: 4
-->

# Messages

## User messages

### `MsgStake`

```go
type MsgStake struct {
    Owner      string
    Amount     sdk.Coin // positive uusd
    LockTierId uint32
}
```

Validates the owner, denomination, amount, pause state, and configured tier. Transfers the amount to the principal pool and returns a new position ID.

### `MsgBeginUnbonding`

```go
type MsgBeginUnbonding struct {
    Owner      string
    PositionId uint64
}
```

Requires ownership and `ACTIVE` status. Settles accrued rewards, removes active shares, records the maturity timestamp using the position's duration snapshot, and changes status to `UNBONDING`.

### `MsgWithdraw`

```go
type MsgWithdraw struct {
    Owner      string
    PositionId uint64
}
```

Requires ownership, `UNBONDING` status, and a maturity timestamp no later than the current block time. Transfers the original principal from the principal pool and marks the position `WITHDRAWN`.

### `MsgClaimRewards`

```go
type MsgClaimRewards struct {
    Owner      string
    PositionId uint64
}
```

Settles rewards for the position and transfers them from the reward pool. Claims fail if the reward pool lacks the required balance. The position remains usable according to its lifecycle status.

### `MsgFundRewards`

```go
type MsgFundRewards struct {
    Authority string // governance module address
    Amount sdk.Coin // positive uusd
}
```

Requires the configured governance authority. Debits the distribution community pool, transfers the existing funds into the reward pool, and increases the reward index across active shares. Users stake directly through `MsgStake`; no contract handles principal.

## Authority messages

### `MsgUpdateParams`

Requires the configured governance authority and replaces the complete validated parameter set. This is the control used to configure tiers or pause/resume operations.

## Queries

The module exposes:

* `Query/Position(position_id)`;
* `Query/PositionsByOwner(owner, pagination)`;
* `Query/RewardState()`; and
* `Query/Params()`.

REST paths are rooted at `/terra/ustcstaking/v1/` as defined in the protobuf query service.

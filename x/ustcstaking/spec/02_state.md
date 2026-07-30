<!--
order: 2
-->

# State

## Positions

Positions are keyed by numeric ID. The position stores:

| Field | Meaning |
|---|---|
| `id` | Unique position ID. |
| `owner` | Bech32 account address. |
| `principal` | Original positive `uusd` deposit, returned on withdrawal. |
| `shares` | Active reward shares; zero after unbonding begins. |
| `lock_tier_id` | Tier selected when the position was created. |
| `share_multiplier` | Multiplier snapshot used to derive shares. |
| `reward_debt` | Global reward-index snapshot for settled accounting. |
| `unbonding_end_time` | Maturity time after unbonding begins. |
| `status` | `ACTIVE`, `UNBONDING`, or `WITHDRAWN`. |
| `claimable_rewards` | Rewards settled for later claim. |
| `lock_duration` | Immutable duration snapshot used for maturity. |

Storage prefixes:

```text
0x10<position_id>           -> Position
0x11<owner><position_id>    -> owner lookup namespace
0x12<unbonding...>           -> unbonding lookup namespace
0x13                        -> next position ID
```

The owner and unbonding namespaces are reserved for indexed access. Current owner queries scan position state and filter by owner; the position key is authoritative.

## Reward state

```text
0x20 -> RewardState
```

`RewardState` contains:

* `reward_index`: cumulative fixed-point rewards per active share;
* `total_shares`: sum of shares for active positions.

The reward index is unchanged when there are no active shares; funding without active shares is rejected rather than creating unallocated rewards.

## Parameters

```text
0x30 -> Params
```

`Params` contains the fixed bond denomination, lock tiers, governance authority, funding authority, and pause flag. Genesis validation requires non-negative accounting values, valid owners, valid coin denominations, unique position IDs, and equality between active position shares and `RewardState.total_shares`.

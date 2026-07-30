<!--
order: 6
-->

# Parameters

| Key | Type | Current default/constraint |
|---|---|---|
| `bond_denom` | string | Must be `uusd`. |
| `lock_tiers` | `[]LockTier` | Empty by default; governance must configure a tier before staking is enabled. |
| `authority` | address | Governance module address by default; controls parameter updates. |
| `funding_authority` | address | Governance module address by default; controls reward funding. |
| `paused` | bool | `false` by default. |

Each `LockTier` has:

| Key | Type | Constraint |
|---|---|---|
| `id` | uint32 | Unique within the parameter set. |
| `duration` | duration | Positive. |
| `multiplier` | decimal | Positive fixed-point value. |

There is intentionally no inflation rate, mint allowance, reward emission schedule, or automatic funding parameter. Rewards enter the system only through `MsgFundRewards` and are limited by the reward-pool balance.

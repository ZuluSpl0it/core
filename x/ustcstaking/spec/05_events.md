<!--
order: 5
-->

# Events

The current implementation does not yet emit module-specific `sdk.Event`s from message handlers. Transaction results still expose the SDK's standard message metadata, but indexers should not assume dedicated `stake`, `unbond`, `withdraw`, `claim`, or `fund` events exist until an event schema is added.

Before production release, the recommended event contract is:

| Type | Attribute keys |
|---|---|
| `ustcstaking_stake` | `position_id`, `owner`, `amount`, `lock_tier_id`, `shares` |
| `ustcstaking_begin_unbonding` | `position_id`, `owner`, `unbonding_end_time`, `claimable_rewards` |
| `ustcstaking_withdraw` | `position_id`, `owner`, `principal` |
| `ustcstaking_claim_rewards` | `position_id`, `owner`, `amount` |
| `ustcstaking_fund_rewards` | `sender`, `amount`, `reward_index` |
| `ustcstaking_update_params` | `authority`, `paused`, `lock_tier_count` |
| `ustcstaking_update_funding_authority` | `authority`, `funding_authority` |

The event names and attributes above are a specification recommendation, not current behavior. They should be finalized before clients or indexers depend on them.

<!--
order: 5
-->

# Events

Message handlers emit module-specific `sdk.Event`s only after successful bank
transfers and state writes. Failed messages emit no module-specific event.
Indexers can use these events for notification, but must rebuild state from
queries or exported genesis when correcting historical data.

The production event contract is:

| Type | Attribute keys |
|---|---|
| `ustcstaking_stake` | `position_id`, `owner`, `amount`, `lock_tier_id`, `shares` |
| `ustcstaking_begin_unbonding` | `position_id`, `owner`, `unbonding_end_time`, `claimable_rewards` |
| `ustcstaking_withdraw` | `position_id`, `owner`, `principal` |
| `ustcstaking_claim_rewards` | `position_id`, `owner`, `amount` |
| `ustcstaking_fund_rewards` | `sender`, `amount`, `reward_index_before`, `reward_index_after`, `reward_pool_balance` |
| `ustcstaking_update_params` | `authority`, `paused`, `lock_tier_count` |
| `ustcstaking_update_funding_authority` | `authority`, `old_funding_authority`, `new_funding_authority` |

Coin attributes use canonical SDK strings. Decimal indexes use canonical
decimal strings. Timestamps use UTC RFC3339Nano. Event types and attribute
keys are stable API surface and require a specification revision before they
are renamed.

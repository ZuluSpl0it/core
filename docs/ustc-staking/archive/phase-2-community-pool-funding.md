# USTC Staking Phase 2: Community-Pool Funding Runbook

Status: implementation and controlled-test runbook. This is not authorization
to activate the module on a public network.

## Funding model

Users stake and manage positions directly through native `x/ustcstaking`
messages. Governance funds rewards by executing `MsgFundRewards` from the
distribution community pool. No funding account, contract, POL, DEX route,
automated revenue source, or mint path participates.

The message body is:

```json
{
  "@type": "/terra.ustcstaking.v1.MsgFundRewards",
  "authority": "$GOVERNANCE_MODULE_ADDRESS",
  "amount": {"denom": "uusd", "amount": "$APPROVED_MICRO_USTC"}
}
```

Derive the authority from the running binary and chain configuration; do not
copy an address from another network. The governance proposal should state the
amount, purpose, current reward-pool balance, active shares, resulting reward
index estimate, and expected post-funding community-pool balance.

Before submission, confirm the module is not paused, active shares are
positive, the amount is positive `uusd`, and the community pool can cover it.
At execution, the application adapter subtracts from distribution FeePool
accounting and transfers the same coins from the distribution module account
to `ustcstaking_reward_pool`. Failed execution must not alter reward state or
balances. Do not add the reward pool to bank `allowedReceivingModAcc`.

## Reconciliation record

Keep one evidence row per executed proposal:

```text
proposal_id, execution_height, approved_uusd,
community_pool_before, community_pool_after,
distribution_module_before, distribution_module_after,
reward_pool_before, reward_pool_after,
reward_index_before, reward_index_after, tx_or_proposal_event
```

Pass conditions:

```text
community_pool_before - community_pool_after = approved_uusd
distribution_module_before - distribution_module_after = approved_uusd
reward_pool_after - reward_pool_before = approved_uusd
event.source = community_pool
event.amount = approved_uusd
total_uusd_supply_after = total_uusd_supply_before
```

Capture the proposal result, execution height, transaction/event, pre- and
post-state queries, and total-supply query. Verify the reward index and reward
pool balance against active shares and the module's documented truncation
policy.

## Controlled-network test checklist

- successful proposal funding; verify all reconciliation conditions;
- direct account submission with an incorrect authority is rejected;
- wrong denomination and zero amount are rejected;
- no active shares and insufficient community-pool balance are rejected;
- paused module rejects funding;
- failed cases leave FeePool accounting, module balances, reward pool, reward
  index, and total supply unchanged;
- users directly stake, claim, pause/resume through governance, unbond, and
  withdraw; verify principal and reward module separation;
- restart a validator and the full network; re-query balances and state;
- confirm direct account transfers to the reward module remain blocked.

Do not treat Phase 1 seven-validator functional testing as evidence of Phase 2
funding safety. Public testnet or production activation remains gated on
upgrade rehearsal, invariant registration and tests, independent review, and
the release checklist in `phase-2-community-pool-release-readiness.md`.

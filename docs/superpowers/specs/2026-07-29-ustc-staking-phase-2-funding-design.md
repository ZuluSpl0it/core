# USTC Staking Phase 2 Funding Design

> **Superseded:** This contract-based design was replaced on 2026-09-19 by
> `2026-09-19-ustc-staking-phase-2-community-pool-funding-design.md`. Do not
> implement this document.

## Decision

Phase 2 uses a separate CosmWasm `TreasuryManager`/POL adapter to convert
approved USTC revenue into Phase 1 reward funding. The native
`x/ustcstaking` ledger remains the sole owner of positions, reward indexes,
claims, and principal custody.

The adapter reaches the native module through a standard protobuf Stargate
`MsgFundRewards` dispatch signed by the contract address. Governance configures
one `funding_authority` in `x/ustcstaking`; this is distinct from the module's
governance `authority`, which controls parameters and can replace or revoke the
funding authority.

## Why this choice

1. **Fault isolation.** POL, DEX routing, oracle assumptions, and contract
   upgrades are higher-risk and change more often than position accounting. A
   contract failure can pause revenue without corrupting positions or user
   principal.
2. **Auditable money flow.** Each reward allocation is one atomic transaction:
   contract USTC balance → `MsgFundRewards` → native reward-pool account →
   reward-index update. The native module rejects the allocation if the money
   transfer or authorization fails.
3. **Reuse of the existing chain boundary.** `custom/wasm/keeper/handler_plugin.go`
   already creates an SDK message handler backed by the application message
   router. `wasmbinding/message_plugin.go` reserves Terra custom messages for
   market swaps. Using protobuf Stargate dispatch avoids adding a broad new
   custom-message format just for funding.
4. **Governance containment.** A single native `funding_authority` can be
   revoked without migrating positions. Contract admin, code migration, fee
   sources, allocation policy, and timelocks remain contract/governance
   concerns rather than permanent native staking behavior.
5. **Staged delivery.** Phase 1 can launch with governance bootstrap funding;
   Phase 2 adds a revenue source without changing claim, lock, or custody
   semantics.

## Deliberate differences from standard Cosmos patterns

| Area | Common pattern | Chosen design | Why |
|---|---|---|---|
| Funding | native mint/distribution or direct module coupling | contract emits an authorized native funding message | rewards remain fully pre-funded and contract risk stays outside the ledger |
| Contract integration | chain-specific custom message for every action | protobuf Stargate call only for `MsgFundRewards` | uses the existing SDK message router and keeps custom bindings narrow |
| Authority | one module authority for every privileged action | governance authority plus revocable funding authority | contract needs limited funding rights, never parameter or custody rights |
| Revenue accounting | off-chain accounting or periodic manual transfer | immutable receipt and allocation events plus native funding event | provides cross-layer reconciliation |
| DEX policy | hard-code a venue into the staking module | allow-list revenue sources in TreasuryManager | DEX routes can change without upgrading position accounting |

## Components and responsibility boundaries

### `x/ustcstaking` (native)

- Stores `authority` and `funding_authority` separately.
- Accepts `MsgFundRewards` only when `sender == funding_authority`.
- Transfers `uusd` from the signer account to `ustcstaking_reward_pool` before
  changing the reward index.
- Does not know DEX routes, LP tokens, prices, burn logic, or contract state.
- Emits `ustcstaking.reward_funded` with sender, amount, pre/post index, and
  reward-pool balance.

### `TreasuryManager` (CosmWasm)

- Holds only funds that are deliberately routed to it by approved revenue
  sources.
- Maintains an allow-list of source contracts/accounts, an emergency pause,
  a code-migration authority, and a timelocked allocation policy.
- Records each receipt by unique receipt ID and rejects duplicate processing.
- Allocates a configured part of realized USTC revenue to one of three
  destinations: reward funding, reserve/treasury, or buyback/burn executor.
- Emits receipt and allocation events; its USTC balance must reconcile with
  recorded unallocated/reserve allocations.
- Never holds user staking principal and cannot call `Stake`, `Withdraw`,
  `ClaimRewards`, or parameter updates on behalf of users.

### POL / DEX revenue source

- Is a separately allow-listed contract or account that transfers realized
  `uusd` to `TreasuryManager`.
- Supplies no pricing, yield, or entitlement data to `x/ustcstaking`.
- Is disabled by removing it from TreasuryManager's source allow-list; this
  must not require a native module upgrade.

## Atomic funding flow

```text
approved source
  → transfers realized uusd to TreasuryManager
  → TreasuryManager records receipt_id
  → after timelock/policy validation, TreasuryManager dispatches Stargate MsgFundRewards
  → x/ustcstaking verifies contract == funding_authority and denom == uusd
  → bank transfer contract → ustcstaking_reward_pool
  → reward index increases and native event is emitted
```

The entire allocation transaction reverts if the native message fails. The
contract must mark a receipt allocated only after the native response succeeds.
This ordering prevents a receipt being counted as paid when reward funding did
not occur.

## Policy defaults and bounds

- Default allocation: 50% reserve/treasury and 50% buyback/burn, as proposed
  in the v1.1 paper. **No reward funding percentage is enabled by default.**
  Governance must explicitly create a reward-funding allocation before it can
  fund stakers.
- Treasury/reserve allocation bounds: 5%–50% of a policy update, preserving
  the paper's stated safety range.
- Allocation-policy changes, source allow-list changes, code migration, and
  funding-authority replacement require governance authorization and a
  30-day timelock before execution.
- Emergency pause stops new receipts, swaps, allocations, and Stargate funding;
  it never blocks native withdrawals or reward claims already funded.
- All contract-facing USTC amounts are `uusd`; other denoms are rejected.

These are policy defaults rather than a yield guarantee. The staking module
never calculates or promises APR.

## Required Phase 1 alignment

Before Phase 1 coding begins, amend the Phase 1 spec and plan as follows:

1. Replace the single `authority` funding check with two fields:
   `authority` for `MsgUpdateParams` and `funding_authority` for
   `MsgFundRewards`.
2. Add `MsgUpdateFundingAuthority` signed by `authority`, with a non-empty
   bech32 address validation and an event containing old/new addresses.
3. Make `MsgFundRewards.sender` the contract signer; it transfers its own
   `uusd` balance to the reward pool atomically.
4. Add integration tests that execute the generated Stargate message from a
   contract account through the existing Wasm SDK message handler, reject a
   non-authorized contract, and prove the transaction rolls back on native
   validation failure.

## Security and operational requirements

- Independent Rust/CosmWasm audit for TreasuryManager and a separate Go audit
  for the native authorization/accounting changes.
- Contract unit, multi-test integration, and chain-level Wasm dispatch tests.
- Replay-resistant receipt IDs; no allocation may be processed twice.
- Reconciliation tooling compares source receipts, TreasuryManager balances and
  events, native funding events, and `ustcstaking_reward_pool` balances.
- Contract migration test proves state preservation and unchanged authorization.
- Operational runbook documents pause, authority rotation, source removal,
  timelock execution, failed Stargate dispatch, and emergency withdrawal
  behavior.

## Out of scope

- Validator whitelist, validator self-bond requirements, performance pool, or
  any validator rewards.
- Changes to LUNC consensus staking, slashing, distribution, minting, or
  governance voting power.
- A hard-coded DEX, oracle, or liquidity provider.
- Any guarantee of reward amount, APR, buyback result, or price outcome.

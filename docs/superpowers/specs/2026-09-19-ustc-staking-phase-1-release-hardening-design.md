# USTC Staking Phase 1 Release-Hardening Design

## Status and decision

Phase 1 functional behavior is accepted as validated at revision
`5c5d7b55807ae6a604337a7e2789c69eab401879`. Validator testing covered a
seven-validator local network, staking, funded reward distribution, claims,
governance pause and resume, unbonding, withdrawal, unauthorized operations,
node and network restarts, supply conservation, real BankKeeper accounting,
offline export, and 586,799 fuzz-generated cases.

This work does not change Phase 1 economics. It closes release-hardening gaps
before a public testnet or production upgrade:

1. add the live-chain upgrade and new-store initialization;
2. register production accounting invariants;
3. replace the global owner-query scan with a secondary index;
4. strengthen imported-state validation and position-ID handling;
5. add stable module events;
6. replace the incorrect Market simulation decoder;
7. correct the localnet runbook and repeat the release verification campaign.

Repository-wide dependency advisories are a release gate, but dependency
updates are a separate security-maintenance project. Mixing broad dependency
changes into accounting hardening would enlarge the regression surface and
make validator retesting harder to interpret.

## Architectural boundaries

The original Phase 1 boundaries remain unchanged:

- `x/ustcstaking` remains separate from `custom/staking` and LUNC validator
  consensus.
- Only `uusd` is accepted as principal or reward funding.
- Rewards remain pre-funded. No mint, distribution, or promised-APR path is
  added.
- The principal and reward module accounts retain no Minter or Burner
  permissions.
- No begin blocker, end blocker, staking hook, validator-power change,
  commission change, slashing, or jailing behavior is added.
- Phase 2 TreasuryManager/POL and Phase 3 validator incentives remain out of
  scope.

The hardening changes may add indexes, validation, diagnostics, events, tests,
and upgrade wiring. They must not alter valid position balances, accrued reward
results, authority roles, lock semantics, or withdrawal availability.

## Approaches considered

Three delivery shapes were considered:

1. One plan including hardening and all dependency upgrades. This gives one
   nominal release branch but mixes unrelated consensus and dependency risk.
2. One plan per finding. This maximizes isolation but creates unnecessary
   planning and validation overhead for tightly related module changes.
3. One cohesive Phase 1 hardening plan, with dependency remediation tracked as
   a separate release gate. This is the selected approach.

The selected approach keeps one coherent accounting and state-migration review
surface while preventing repository-wide dependency churn from obscuring
Phase 1 results.

## Live-chain upgrade

### Upgrade identity and registration

Create `app/upgrades/ustc_staking` with fixed technical identifier
`ustc_staking`. Register it in `app/app.go` alongside existing upgrades. Its
`StoreUpgrades.Added` list contains only `ustcstakingtypes.StoreKey`
(`x_ustcstaking`). No existing store is deleted or renamed.

The Phase 3 design currently reserves the same identifier for its future final
launch. This Phase 1 hardening design supersedes that detail: `ustc_staking`
names the initial native-module upgrade. Any later Phase 2 or Phase 3 chain
upgrade must use a distinct identifier and revise its design before coding.

### Initialization behavior

The upgrade handler calls `module.Manager.RunMigrations`. Cosmos SDK v0.53
detects that `ustcstaking` is absent from the prior version map and invokes the
module's `DefaultGenesis` and `InitGenesis` path. The resulting state is:

- governance module address as both `authority` and `funding_authority`;
- `uusd` bond denom;
- no lock tiers;
- module unpaused;
- zero reward index and total shares;
- next position ID equal to 1;
- no positions or owner-index entries.

The handler also materializes the `ustcstaking` and
`ustcstaking_reward_pool` module accounts through `AccountKeeper` and verifies
that both have no permissions. Explicit creation makes the upgrade result
observable and avoids deferring account creation until the first transfer.

### Upgrade safety proof

An application-level rehearsal starts from a pre-upgrade version map and state,
applies the real store loader and upgrade handler, then checks:

- the new store opens and contains valid default state;
- both module accounts exist with no permissions and zero balances;
- export and validation succeed immediately after upgrade;
- all pre-existing account balances and total `uusd` supply are unchanged;
- LUNC validator set, voting power, delegations, commission, slashing, and
  jailing state are unchanged;
- applying the handler at the wrong plan name or attempting duplicate store
  creation is rejected by the normal upgrade machinery.

## Accounting invariants

Create pure keeper invariant routes and expose them through
`AppModule.RegisterInvariants` for SDK-compatible tooling. Because the app's
Cosmos SDK v0.53 manager makes that registry a no-op and crisis is absent, the
production operator path is the on-demand `Query/ValidateState` documented
below. Each check returns deterministic diagnostics without mutating state.

### Principal custody

Sum `principal.amount` for every active and unbonding position. Withdrawn
positions must contribute zero. The sum must equal the `uusd` balance of the
`ustcstaking` principal module account exactly. Any non-`uusd` balance in that
account is also a violation.

### Active shares

Sum snapshotted `shares` for active positions. Unbonding and withdrawn
positions must have zero shares. The sum must equal
`RewardState.total_shares` exactly. Active shares and the stored total must be
non-negative.

### Reward solvency

For each active position, compute currently payable rewards with the same
keeper calculation used by claims. For each unbonding or withdrawn position,
use stored `claimable_rewards`. The sum of all payable rewards must not exceed
the `uusd` balance of `ustcstaking_reward_pool`. A larger pool balance is valid
because integer division leaves dust. Any non-`uusd` pool balance is a
violation.

### Supply conservation boundary

Global USTC supply conservation is not registered as a runtime module
invariant. Other authorized chain modules may legitimately mint or burn USTC,
so `x/ustcstaking` cannot claim ownership of the global supply baseline.
Instead:

- application integration tests snapshot total `uusd` supply before and after
  each complete staking lifecycle and require equality;
- tests verify both USTC staking module accounts lack Minter and Burner
  permissions;
- tests prove all successful staking operations are balance transfers and all
  rejected operations leave balances and supply unchanged.

This provides a valid module-scoped guarantee without creating an invariant
that can halt for unrelated protocol activity.

## Owner-position index

`OwnerPositionKeyPrefix` becomes an active secondary index. Entries are keyed
by canonical owner address bytes followed by big-endian position ID. The value
is empty; the primary position record remains authoritative.

`SetPosition` maintains the index and rejects owner changes for an existing
position. `DeletePosition`, if used, removes both records. `InitGenesis`
rebuilds the index from validated positions; the index is derived state and is
not added to the genesis protobuf.

`PositionsByOwner` paginates only the selected owner's index, loads each
primary position by ID, and returns IDs in numeric order. Missing primary
records, owner mismatches, and malformed index keys are treated as state
corruption and return errors rather than silently omitting data.

Tests cover deterministic ordering, key and offset pagination, multiple
owners, genesis rebuild, update idempotence, corruption handling, and a
100,000-position benchmark demonstrating work proportional to one owner's
results rather than the global position count.

## Genesis and imported-state validation

Validation becomes status-specific and rejects any state that could panic,
wrap an identifier, create an unpayable obligation, or encode an impossible
position.

### Lock tiers and snapshots

- Lock tier ID zero is reserved and rejected in params and positions.
- Param tier IDs remain unique.
- Param durations and multipliers must be positive.
- Every position must have a positive snapshotted lock duration and share
  multiplier, including positions whose original tier was later removed from
  params.
- A position's tier ID need not still exist in current params; snapshots make
  historical positions independent of later governance changes.

### Position status rules

- Active: positive principal and shares, non-negative reward debt, explicit
  zero-or-positive `uusd` claimable rewards, and no unbonding completion time.
- Unbonding: positive principal, zero shares and reward debt, explicit
  zero-or-positive `uusd` claimable rewards, and a non-zero completion time.
- Withdrawn: zero principal, shares, and reward debt; explicit
  zero-or-positive `uusd` claimable rewards remain allowed until claimed; the
  recorded completion time may remain for audit history.
- Unknown status, malformed owner, nil arithmetic values, wrong denoms, and
  negative amounts are rejected.

### Reward-state and obligation checks

Reward index and total shares must be initialized and non-negative. Active
shares must equal stored total shares. For every active position,
`shares * reward_index - reward_debt` must be non-negative. Genesis validation
also sums payable rewards and rejects obligations greater than the supplied
reward-pool balance when validation runs with application bank state.

Pure protobuf `ValidateGenesis` has no bank access. Balance-backed checks
therefore run during application import and upgrade rehearsal through a keeper
validation method after bank genesis is loaded.

### Position-ID exhaustion

`next_position_id` must be non-zero and greater than every stored position ID.
`math.MaxUint64` is rejected as a stored position ID because no next ID can
represent its successor. `Stake` checks exhaustion before transferring funds;
it returns a dedicated error and performs no state or balance change.

## Module events

Add constants and tests for these stable event contracts:

| Event type | Required attributes |
|---|---|
| `ustcstaking_stake` | `position_id`, `owner`, `amount`, `lock_tier_id`, `shares` |
| `ustcstaking_begin_unbonding` | `position_id`, `owner`, `unbonding_end_time`, `claimable_rewards` |
| `ustcstaking_withdraw` | `position_id`, `owner`, `principal` |
| `ustcstaking_claim_rewards` | `position_id`, `owner`, `amount` |
| `ustcstaking_fund_rewards` | `sender`, `amount`, `reward_index_before`, `reward_index_after`, `reward_pool_balance` |
| `ustcstaking_update_params` | `authority`, `paused`, `lock_tier_count` |
| `ustcstaking_update_funding_authority` | `authority`, `old_funding_authority`, `new_funding_authority` |

Events emit only after all validation, transfers, and state writes succeed.
Failed messages emit no module event. Amounts use canonical SDK coin strings;
indexes use canonical decimal strings; timestamps use UTC RFC3339Nano.

Events are observability records, not state. Indexers must be able to rebuild
their view from queries and exported state.

## Simulation decoder

Create `x/ustcstaking/simulation/decoder.go` and its unit tests. Register this
decoder from `x/ustcstaking/module.go` instead of importing
`x/market/simulation`.

The decoder handles params, reward state, next position ID, primary position
records, and owner-index records. Unknown keys produce deterministic hex-key
output rather than being interpreted as Market state. Decoder code remains
diagnostic only and cannot mutate consensus state.

## Real-keeper integration tests

Retain fast keeper unit tests, but add application tests using the real Terra
AccountKeeper and BankKeeper. The reference scenario creates eight users and
asserts:

- exact principal-pool custody after stakes, unbondings, and withdrawals;
- reward-pool solvency across funding, claims, rounding dust, and zero-value
  repeated claims;
- exact active-share totals after each transition;
- unchanged total `uusd` supply across successful and failed operations;
- rollback of transfers and state when authorization, denom, balance, pause,
  maturity, or owner checks fail;
- no changes to validator staking state.

Cosmos SDK v0.53 marks `sdk.InvariantRegistry` deprecated, makes
`module.Manager.RegisterInvariants` a no-op, and this application no longer
includes `x/crisis`. Do not restore crisis or add a per-block scan. Keep the
three pure invariant routes on the module for compatible tooling, and expose
an explicit read-only `Query/ValidateState` plus `terrad query ustcstaking
validate-state` that runs the same checks on demand. Genesis import, upgrade
initialization, real-app integration tests, and validator campaigns exercise
that checker. A future SDK runtime invariant facility can register the same
checks without changing their accounting rules.

## Documentation and verification

Update the localnet runbook to match actual CLI behavior, governance messages,
pause semantics, event output, upgrade steps, and test prerequisites. Preserve
the existing seven-validator campaign and add a pre-upgrade-to-post-upgrade
rehearsal.

Release verification requires:

- focused types, keeper, CLI, genesis, invariant, event, decoder, and upgrade
  tests;
- real-keeper application integration tests;
- race tests and fuzz regression seeds;
- static checks and formatting;
- full repository tests;
- a repeated seven-validator campaign using the release candidate;
- recorded total-supply snapshots and exported-state validation;
- Graphify refresh and validation, kept local and ignored by Git;
- independent review of accounting, authorization, imported state, upgrade,
  and query resource use.

Dependency scan results must be recorded with advisory, affected package,
reachable code path, severity, mitigation, and release disposition. Fixes that
change repository dependencies require their own plan and regression cycle.

## Acceptance criteria

Phase 1 hardening is complete only when:

1. an existing pre-USTC chain state upgrades through `ustc_staking` without
   changing existing balances, supply, or validator state;
2. default USTC staking state and both permissionless module accounts exist
   after upgrade;
3. principal, shares, and reward-solvency invariant routes are available to
   compatible tooling, and bounded `Query/ValidateState` pages plus the CLI's
   complete pinned-height aggregation pass against real keeper state;
4. owner queries use the secondary index and remain deterministic;
5. malformed genesis/import states and exhausted IDs fail before mutation;
6. every successful message emits its specified event and failed messages do
   not;
7. simulation uses the USTC decoder;
8. all automated verification and the repeated validator campaign pass;
9. dependency advisories have explicit release dispositions;
10. no Phase 2, Phase 3, mint, distribution, or validator-consensus behavior
    enters the change set.

Passing these criteria establishes a Phase 1 release candidate. It does not
replace an independent security audit or governance approval for activation.

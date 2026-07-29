# USTC Staking Phase 1 Design

## Scope

Build a native `x/ustcstaking` Cosmos SDK module that locks USTC, accounts for
only already-funded USTC rewards, and permits deterministic claims. It is a
separate financial-locking system, not a replacement or extension of LUNC
validator staking.

The Phase 1 module accepts a controlled USTC funding message. Protocol-owned
liquidity (POL), the CosmWasm `TreasuryManager`, DEX-fee routing, and validator
participation policy are deferred to Phase 2. This boundary permits a safe,
auditable launch with bootstrap funding and prevents contract or DEX policy
from being coupled to the core staking ledger.

## Architecture

`x/ustcstaking` owns:

- USTC staking positions and lock-tier parameters.
- Unbonding queue and release state.
- Separate principal and reward-pool module-account balances.
- A cumulative reward-per-share index, per-position reward debt, and a defined
  rounding-dust policy.
- `FundRewards`, `Stake`, `BeginUnbonding`, `Withdraw`, and `ClaimRewards`
  messages, with query, genesis, migration, invariant, and test support.

The module depends only on narrow account and bank keeper interfaces. It stores
two explicit authorities: `authority` controls parameters and
`funding_authority` alone may call `FundRewards`. The funding authority is
restricted to USTC and moves funds into the reward module account before
accounting is updated. Phase 1 sets both to governance; Phase 2 may replace
only `funding_authority` with TreasuryManager's contract address.

## Repository integration

Graphify identifies the application seams in `app/keepers/keepers.go`,
`app/modules.go`, `cmd/terrad/root.go`, and the existing `x/treasury`,
`x/dyncomm`, and `x/market` modules.

The implementation adds a separate `x/ustcstaking` package and keeper. It is
registered in `ModuleBasics`, keeper construction, `appModules`, genesis order,
and—only if needed after measurement—the end-block order. Its protobuf,
codec, gRPC, REST gateway, CLI, genesis, migration, and test patterns follow
the existing native modules.

Do not modify `custom/staking`: that package is a wrapper for the SDK
validator-staking module, including LUNC bond-denom genesis and legacy query
compatibility. USTC locking must not inherit validator-facing behavior.

## Deliberate differences from standard staking

| Area | Standard `x/staking` | `x/ustcstaking` | Why |
|---|---|---|---|
| Purpose | secure consensus through validator delegation | lock USTC for a funded-reward program | USTC positions must not affect LUNC consensus |
| Asset | LUNC bond denom | USTC only | proposal is USTC-specific |
| Rewards | distribution module and validator commission | module-local, pre-funded reward index | no minting and no implicit subsidy |
| Penalties | slashing and validator jailing | no slashing in Phase 1 | users are not securing the chain |
| Position routing | delegation and redelegation between validators | no validator choice or redelegation | eliminates validator-power coupling |
| Unbonding | validator-delegation unbonding | position-specific release queue | lock policy is independent of validator safety |
| Funding | inflation/fee distribution mechanisms | explicit governed `FundRewards` deposit | every payable reward must have a real USTC source |

## Safety requirements

- No mint-capable keeper, mint message, or implicit issuance path.
- Principal and reward funds cannot be commingled in accounting or transfer
  paths.
- A claim cannot exceed the reward pool actually funded and allocated.
- All amount arithmetic uses SDK integer/decimal primitives with tested
  rounding and dust handling.
- Parameters are bounded and governance-controlled; funding authority changes,
  pause behavior, and upgrades are explicit.
- Genesis/export and migration preserve all position, queue, and reward-index
  state.
- Invariants cover principal custody, reward custody, position totals, and
  non-negative claimable rewards.

## Persistent delivery checklist

- [ ] Define state, messages, params, error cases, and invariants.
- [ ] Add protobuf, codecs, generated interfaces, and CLI/gateway services.
- [ ] Implement keeper accounting and module-account custody.
- [ ] Implement messages with authorization and denom checks.
- [ ] Implement queries, genesis, migration, and deterministic queue handling.
- [ ] Register keeper, module, module account, application lifecycle, and tests.
- [ ] Add unit, invariant, simulation/property, integration, and upgrade tests.
- [ ] Run security/economic review and upgrade rehearsal.
- [ ] Publish operator, validator, and user documentation.

## Deferred Phase 2 checklist

- [ ] Specify TreasuryManager contract authority and failure handling.
- [ ] Specify POL fee collection and DEX integration.
- [ ] Specify treasury/refill and buyback/burn allocation governance bounds.
- [ ] Specify timelock, emergency controls, audits, and contract migration.
- [ ] Integrate Phase 2 only through the Phase 1 `FundRewards` boundary.

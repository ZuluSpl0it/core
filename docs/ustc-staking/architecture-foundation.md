# USTC Staking: Architecture Foundation

Status: discovery baseline. This is not an approval to implement economics, minting, or a contract.

## Intent

The proposed feature is a separate, native `x/ustcstaking` module: users lock
USTC and earn only revenue that has first been received by the protocol. It
must not alter LUNC validator consensus, the existing `x/staking` bond denom,
or introduce reward minting.

The two Agora proposals establish the native-module direction. The v1.1
"Self-Sustaining" paper adds the missing funding layer: a CosmWasm
`TreasuryManager` and protocol-owned-liquidity (POL) revenue route, with a
governed treasury-refill / USTC-buyback-and-burn split. Treat that layer as an
optional, separately-approved dependency—not as a prerequisite for the core
staking ledger.

## Recommended boundaries

| Layer | Responsibility | Must not own |
|---|---|---|
| `x/ustcstaking` | positions, lock tiers, unbonding queue, reward index, claims, queries, invariants | validator power, LUNC delegation, token issuance |
| Revenue adapter | accepts only authorized, already-realized USTC revenue and credits the reward pool | DEX routing policy, arbitrary transfers, price decisions |
| Treasury/POL contracts | collect approved DEX fees, maintain POL, execute governed allocation | staking position state |
| Governance/upgrade | parameter changes, allow-lists, activation and migrations | routine per-user accounting |

The core module should work with a bootstrap-funded reward pool before any POL
contract is connected. This makes the economic source auditable and keeps a
CosmWasm incident from compromising the staking ledger.

## Repository integration map

Graphify's refreshed `coding` profile identifies the existing application and
module lifecycle seams:

1. Add `x/ustcstaking` as a new module; do not extend `custom/staking`.
   `custom/staking` wraps the SDK's validator staking module and changes its
   legacy queries and LUNC bond-denom genesis behavior.
2. Register `ustcstaking.AppModuleBasic{}` in `app/modules.go` `ModuleBasics`.
   This provides codecs, interfaces, CLI, REST/gRPC gateway, and genesis
   validation.
3. Construct the keeper in the app, add the module account permissions, and
   add `ustcstaking.NewAppModule(...)` to `appModules` in `app/modules.go`.
4. Put protobuf messages and query services under `proto/`; generate Go and
   gateway code by the repository's normal proto workflow. Message server,
   query server, genesis import/export, migrations, and invariants belong in
   `x/ustcstaking`.
5. If queues or reward-index updates need block processing, register a narrow
   end blocker and add its module name to the application blocker order. The
   existing `x/treasury`, `x/dyncomm`, and `x/market` modules are the closest
   local examples for lifecycle, genesis, services, and end-block patterns.
6. Add keepers only for the capabilities actually needed: bank transfers,
   account/module addresses, and optionally a tightly constrained Wasm query
   or execute interface. Avoid depending on the SDK `StakingKeeper`.

## Economic and safety rules

- Rewards are funded by balances actually credited to the USTC reward pool;
  no mint path and no promised APR.
- Claims cannot exceed funded rewards. Use a cumulative reward-per-share index
  plus deterministic rounding and an explicit dust policy.
- Principal and rewards use separate module-account accounting. Neither may be
  mixed with treasury or POL funds.
- Revenue adapters must be allow-listed, denom-restricted to USTC, and capped
  by governance-set limits. No arbitrary external contract can credit rewards.
- Parameter changes that affect payout allocation require governance,
  validation bounds, and a timelock. The paper's suggested 5–50% treasury
  allocation bound and 30-day notice are policy inputs, not code defaults
  until adopted.
- Validator participation requirements in the paper (initial 21 and USTC
  self-bond) are separate policy. They need a precise enforcement and slashing
  model before they can be implemented.

## Unresolved decisions before implementation

1. Exact source of USTC revenue at launch and its custody path.
2. Whether POL/TreasuryManager is in scope for the first upgrade; recommended:
   no, ship only the native ledger plus a governed funding interface.
3. Lock tiers, early exit behavior, unbonding period, reward eligibility, and
   the treatment of inactive or unclaimed rewards.
4. Governance authority, timelock mechanism, emergency pause, and migration
   authority.
5. Audit scope: module accounting, contract interaction, economic simulations,
   invariant and adversarial tests, and upgrade rehearsal.

## Sources

- Signal proposal: https://agora.terra-classic.io/t/signal-proposal-ustc-staking-system-no-minting-framework/452
- Technical implementation proposal: https://agora.terra-classic.io/t/terra-classic-ustc-staking-system-technical-implementation-proposal/492
- `USTC_Staking_System_v1.1_SelfSustaining_LunaClassicDAO.pdf` supplied by the requester.

## Knowledge-base status

Local-only Graphify artifacts are refreshed and validated in `graphify-out/`:
`coding` provides implementation navigation; `domain-api` provides the smaller
API/domain slice. These artifacts are ignored by Git and must never be pushed.

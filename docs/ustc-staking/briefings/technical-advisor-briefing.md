# USTC Staking: Technical Advisor Briefing

**Audience:** Technical advisor supporting C-suite discussion  
**Purpose:** Explain the approved native USTC staking design and current
community-pool funding phase without making unsupported financial claims.

## Bottom line

USTC staking is a proposed USTC utility and rewards program—not a new form of
LUNC validator staking. Users lock USTC in a new native chain module and can
claim only USTC that has already been placed in a dedicated reward pool. The
design does not mint tokens, promise an APR, change LUNC validator power, or
put user principal inside a treasury/POL contract.

The approved implementation sequence is deliberately staged:

| Phase | What it delivers | Why it is separate |
|---|---|---|
| 1. Native USTC ledger | locks, withdrawals, claims, funded rewards | principal custody and accounting must be correct before any revenue integration |
| 2. Native community-pool funding | governance moves existing community-pool USTC into the reward pool | no contract or revenue automation; user principal remains native |
| 3. Validator program and launch | optional validator eligibility/performance incentives plus activation controls | validator policy is operationally sensitive but must not affect consensus |

Phase 2 funding is a governance action. Users stake USTC directly through the
native module; no contract receives principal or acts as a reward funder.
TreasuryManager, POL, DEX routing, and automated revenue remain out of scope.

## How the system works

### Phase 1 — native USTC locking and funded rewards

1. A user locks USTC in a position with a chosen lock tier.
2. USTC principal is held in a dedicated on-chain escrow account.
3. Rewards are held in a different on-chain reward account.
4. A governance-approved `MsgFundRewards` debits the distribution community
   pool and transfers the same USTC into the reward account.
5. The module allocates that deposit across eligible positions using a
   cumulative reward-per-share index.
6. A user claims the accrued amount or starts unbonding; when its release date
   arrives, the user withdraws principal.

The principal account and reward account are intentionally separate. A reward
claim cannot exceed the funded reward balance, and a funding shortfall cannot
consume principal.

### Phase 2 — native community-pool reward funding

Governance executes `MsgFundRewards` to move an approved amount of existing
USTC from distribution's community pool to the native reward pool. The app
adapter subtracts FeePool accounting and transfers the same amount between
module accounts before the reward index changes. This phase has no
TreasuryManager, revenue receipt, POL, DEX, reserve-split, buyback, or automated
funding behavior. Future revenue work would require a separate design and
approval.

### Phase 3 — optional validator program

Phase 3 can allow a capped, governance-approved validator group to demonstrate
a minimum USTC position and qualify for a separate performance pool. The paper
suggests an initial 21-validator cap and a 1,000,000 USTC minimum; these are
policy defaults subject to governance approval.

The program only controls eligibility for optional USTC incentives. It does
not create validators, alter their LUNC stake, change voting power, change
commission, or impose slashing/jailing. Performance scores are submitted by a
constrained authority with published methodology and evidence because the
chain's rolling signing counters alone do not provide a complete, policy-ready
uptime measurement.

## Why the design choices were made

| Choice | Reason | Business consequence |
|---|---|---|
| Native module for positions | deterministic accounting, queries, upgrades, and invariant testing in the chain | more initial Go/upgrade work, lower ambiguity around custody |
| Native governance funding | reward source, amount, and resulting balances are explicitly authorized and reconcilable | funding depends on governance decisions and available community-pool balance |
| Pre-funded rewards only | every payable reward has a visible USTC source | no artificial yield or issuance; payouts depend on actual funding |
| Single governance authority | governance controls both module rules and community-pool funding | clear authorization; proposal review and reconciliation remain essential |
| No direct coupling to standard staking | USTC utility must not change LUNC consensus | validator-policy benefits remain optional and non-consensus |
| Explicit epoch reports for performance | avoids claiming that incomplete on-chain counters are a full performance oracle | requires transparent scorer governance and monitoring |
| No mandatory end blocker | maturity and claims are checked on demand | lower per-block complexity; explicit finalization is needed for performance epochs |

## Standard C-suite questions and answer frame

### What business problem does this solve?

It creates a structured use for USTC: users can choose to lock it in exchange
for a share of USTC that has already been earned or allocated by the protocol.
It may support retention, liquidity strategy, and a clearer treasury policy.
It does **not** itself create revenue, restore a peg, or guarantee demand.

### Where would rewards come from?

For this phase, only from USTC already in the distribution community pool and
explicitly approved by governance. No POL/DEX revenue is routed automatically.
No reward-rate or profit estimate should be presented without an independent
economic model and a separate future funding design.

### What are the main cost categories?

1. Native module engineering, protobuf/API work, test infrastructure, and an
   upgrade rehearsal.
2. Native governance proposal, reconciliation, and operations tooling.
3. Independent Go accounting/security review.
4. Economic simulation, monitoring/reconciliation tooling, governance process,
   incident drills, and operational ownership.
5. Treasury/POL capital, DEX routing costs, liquidity-management costs, and
   any legal, tax, or compliance review required by the organization.

Actual cost estimates are outside this technical plan and require vendor
quotes, staffing assumptions, legal jurisdiction, and revenue-source design.

### How could this be financially sustainable?

It is sustainable only if realized, net USTC inflows exceed the combination of
reward commitments, reserve requirements, contract/DEX operating costs, audit
and maintenance costs, and any buyback/burn allocation. The design protects
the chain from paying more than it has funded; it does not prove that inflows
will be sufficient.

### What is the organization accepting if it approves this?

- Delivery and audit spend before adoption/revenue is proven.
- Smart-contract, DEX/liquidity, governance, economic-model, and reputational
  risk in Phase 2.
- A transparent but authority-based performance-score process in Phase 3.
- Ongoing responsibility to monitor balances, receipts, allocation events,
  authority changes, and policy timelocks.
- The possibility that program uptake or revenue is below expectations, in
  which case rewards are lower or absent rather than minted.

### Why not use existing staking or simply issue rewards?

Existing staking secures LUNC consensus; using it would risk mixing USTC
economics with validator power and slashing rules. Minted rewards would create
a new issuance path and weaken the proposed “rewards come from real funding”
discipline. The chosen design is more work, but it keeps these systems
separate and auditable.

## Technical assurance package

Before activation, the plan requires:

- unit, integration, invariant, fuzz/property, genesis/export, and upgrade
  tests for the native module;
- contract unit, multi-test, migration, and real chain-handler Stargate tests;
- receipt-to-pool reconciliation and monitoring;
- separate Go and Rust/CosmWasm audits;
- governance timelock, authority-rotation, emergency-pause, and rollback
  drills;
- published parameters, scoring methodology, source allow-list, and contract
  checksum.

The local Graphify code graph is available for implementation navigation, but
source files and these approved documents remain authoritative.

## Decisions C-suite must make before activation

1. Whether to fund a Phase 1 pilot and the maximum pilot allocation.
2. Which USTC revenue sources are permitted, and which costs are netted before
   allocation.
3. Reserve/reward/buyback allocation policy, bounds, timelock, and emergency
   authority.
4. Audit budget, accountable owners, monitoring expectations, and pause
   authority.
5. Whether the Phase 3 validator program is enabled at launch, including its
   allow-list, cap, minimum USTC position, score methodology, and reporting
   authority.

## Source baseline

- Approved Phase 1, 2, and 3 designs and implementation plans under
  `docs/superpowers/`.
- [Architecture foundation](../architecture-foundation.md).
- User-provided `USTC_Staking_System_v1.1_SelfSustaining_LunaClassicDAO.pdf`.
- Agora signal and technical proposals cited in the architecture foundation.

This is an internal technical/business briefing, not an investment forecast,
legal opinion, or promise of yield.

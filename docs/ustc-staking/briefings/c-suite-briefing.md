# USTC Staking: C-Suite Decision Briefing

**Purpose:** Explain what is being proposed, why it can be built, what it may
cost, and the risks leadership would accept. This is a decision document—not a
promise of profit, price support, or investment return.

## The decision in one page

The approved USTC program is being delivered in stages:

1. Let people lock USTC for a period of time.
2. Pay rewards only from USTC the program already holds.
3. Governance can fund rewards from the distribution community pool; a
   separate validator-incentive overlay remains later work.

The important safeguard is simple: the system cannot create rewards from
nothing. It can only distribute USTC that has first been funded.

## What this is—and is not

| This is | This is not |
|---|---|
| a controlled USTC utility and reward program | a promise of yield or profit |
| governance-controlled funding from the existing community pool | an automated revenue or yield guarantee |
| a separate system for USTC locks | a change to LUNC validator power or voting |
| a staged plan with stop/go gates | an irreversible all-at-once launch |
| a design that keeps customer funds separate from operating funds | a replacement for audit, legal review, or treasury discipline |

## How money is protected

When a participant locks USTC, it is kept separately from rewards. Users stake
directly in the native module; governance-approved rewards come from the
distribution community pool, not from a separate funding account or contract.

In plain terms:

```text
customer locked USTC        → protected native principal account
governance-approved rewards → native reward account (community pool debit)
```

A reward cannot be paid unless it is already in the reward account. If
governance does not approve further community-pool funding, no new rewards are
added; locked principal remains separately held in the native module.

## The three delivery stages

### Stage 1: Build the safe core

Build the on-chain recordkeeping for locks, release dates, and claims. Start
with a leadership/governance-funded reward pool if a pilot is approved.

**Why first:** it proves the customer-funds and rewards accounting before
governance approves community-pool funding.

### Stage 2: Govern community-pool reward funding

Governance executes a native message to move an approved amount of existing
USTC from the distribution community pool into the reward account. The chain
updates community-pool accounting and the reward balance together. No funding
contract, DEX, POL, buyback, or automated revenue route is part of this phase.

### Stage 3: Optional validator program and full launch controls

Optionally allow a limited group of validators to qualify for a separate
incentive pool by holding a required USTC position and meeting a published
performance standard. The initial policy concept is up to 21 validators and a
1,000,000 USTC minimum position; leadership/governance must explicitly approve
or change those values.

**Important:** this does not change who runs the chain, validator voting power,
or penalties. It is an optional incentive program only.

## Why leadership can have confidence it is buildable

- The plan uses the chain's established way to add native modules, accounts,
  public queries, upgrades, and tests.
- It keeps user principal in a native module and routes rewards only through
  governance-controlled community-pool accounting.
- It defines tests for fund movements, permissions, claims, upgrades,
  insufficient funding, and emergency pauses.
- It requires independent review of chain accounting and upgrade behavior
  before activation.
- It can be piloted with controlled funding before relying on external revenue.

Buildable does not mean risk-free. It means there is a defined implementation,
test, audit, and launch path rather than an open-ended concept.

## Business case: what must be true

The current phase does not create revenue. It can distribute only the amount
that governance approves from the existing community pool. Any future POL,
trading, reserve, or buyback allocation requires separate design and approval.

Before approving a launch budget, request a financial model that shows:

- the community-pool balance and amount proposed for rewards;
- transaction costs and the remaining community-pool balance;
- the maximum amount governance is willing to allocate;
- one-time build/audit costs and ongoing operating costs;
- maximum exposure if funding stops or participation is
  materially higher or lower than expected;
- the rule for reducing or stopping future rewards if funding is insufficient.

This plan intentionally contains no revenue, profit, or return forecast because
those values have not yet been independently established.

## Costs leadership should expect to authorize

- Core chain engineering and upgrade preparation.
- Native chain accounting, governance proposal, reconciliation, and operations
  engineering.
- Independent security and accounting review of the native module.
- Testing, monitoring, reconciliation, and incident drills.
- Ongoing ownership of governance controls, authority changes, reporting, and
  software maintenance.

## Risks leadership would be accepting

| Risk | What it means | Planned control |
|---|---|---|
| Funding risk | governance may approve little or no funding | rewards are limited to funded USTC; disclose the available community-pool balance |
| Custody/accounting risk | a funding or accounting defect could misstate available rewards | native module separation, bank restrictions, tested reconciliation, independent review |
| Governance risk | a poor funding decision can damage trust | publish amount, purpose, and before/after balances; require proposal review |
| Security risk | code or integration defects can affect funds or availability | independent audits, test gates, upgrade rehearsal, monitoring |
| Adoption risk | users or validators may not participate | pilot before scale; do not assume returns or demand |
| Reputation risk | users may misunderstand rewards as guaranteed | clear disclosures: pre-funded only, no promised APR, transparent reporting |
| Operational risk | monitoring or response may fail during an incident | named owners, reconciliation, pause drills, and rollback plans |
| Regulatory/tax risk | treatment may vary by jurisdiction | obtain appropriate professional review before activation |

## Decisions required from leadership

1. Approve or decline a limited Phase 1 pilot and set a maximum funding amount.
2. Approve the budget and ownership model for engineering, audit, monitoring,
   and professional review.
3. Approve the risk limits for reserve, rewards, buyback/burn, and emergency
   pause authority.
4. Set governance and reporting limits for community-pool allocations; any
   future revenue funding needs a separate proposal and design.
5. Decide whether Phase 3 validator incentives are enabled at launch or held
   back until the core and funding stages prove stable.
6. Require all release gates: audits, testing, governance notice period,
   authority setup, monitoring, and incident rehearsal.

## Recommended leadership position

Approve planning and controlled testing only after a budget, ownership, and
risk limit are agreed. Treat community-pool funding and Phase 3 validator
incentives as separate activation decisions—not automatic consequences of
building the core.

That approach preserves optionality: leadership can stop after a successful
pilot or consider a future revenue-funding design only after its business case
and controls are separately reviewed.

## Questions leadership should ask the technical advisor

- Can any locked USTC be used to pay rewards, operating expenses, or buybacks?
  **No.** The design keeps those funds separate.
- Can the system pay more than it has funded? **No.** Claims are capped by the
  reward-pool balance.
- Does this alter LUNC validator control or create new token issuance? **No.**
- Where do Phase 2 rewards come from? **Only a governance-approved debit from
  distribution's community pool; no contract or automated revenue route.**
- What proof will we receive before launch? **Test results, audit reports,
  governance proposal, authority configuration, reconciliation evidence, and
  pause/rollback drill evidence.**

## Reference material

- [Technical advisor briefing](technical-advisor-briefing.md)
- [Architecture foundation](../architecture-foundation.md)
- Approved Phase 1–3 specifications and implementation plans under
  `docs/superpowers/`.

This document is for internal strategy and risk discussion. It is not legal,
tax, accounting, investment, or financial advice.

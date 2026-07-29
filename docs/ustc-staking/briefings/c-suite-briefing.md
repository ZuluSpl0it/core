# USTC Staking: C-Suite Decision Briefing

**Purpose:** Explain what is being proposed, why it can be built, what it may
cost, and the risks leadership would accept. This is a decision document—not a
promise of profit, price support, or investment return.

## The decision in one page

We propose a three-step USTC program:

1. Let people lock USTC for a period of time.
2. Pay rewards only from USTC the program already holds.
3. Later, if approved, use selected protocol revenue to refill that reward
   pool and optionally offer a separate validator-incentive program.

The important safeguard is simple: the system cannot create rewards from
nothing. It can only distribute USTC that has first been funded.

## What this is—and is not

| This is | This is not |
|---|---|
| a controlled USTC utility and reward program | a promise of yield or profit |
| a way to direct approved revenue into transparent buckets | a guarantee that revenue will exist |
| a separate system for USTC locks | a change to LUNC validator power or voting |
| a staged plan with stop/go gates | an irreversible all-at-once launch |
| a design that keeps customer funds separate from operating funds | a replacement for audit, legal review, or treasury discipline |

## How money is protected

When a participant locks USTC, it is kept separately from rewards. Rewards are
kept separately from the treasury and from liquidity-management contracts.

In plain terms:

```text
customer locked USTC       → protected principal account
approved reward funding    → separate reward account
treasury / liquidity funds → separate contract and reserve accounts
```

A reward cannot be paid unless it is already in the reward account. A problem
in the later liquidity or treasury contract can stop new funding, but it
cannot take the USTC that participants locked in the core program.

## The three delivery stages

### Stage 1: Build the safe core

Build the on-chain recordkeeping for locks, release dates, and claims. Start
with a leadership/governance-funded reward pool if a pilot is approved.

**Why first:** it proves the customer-funds and rewards accounting before the
business depends on trading or liquidity revenue.

### Stage 2: Add a controlled funding engine

Build a small treasury manager that can receive approved USTC revenue, record
where it came from, and apply approved allocation rules. It can send a defined
share to the reward account; the rest can remain in reserve or be sent to an
approved buyback/burn process.

**Why separate:** it limits the impact of a contract, trading, or liquidity
problem. The core customer-locking system keeps working even if this stage is
paused.

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
- It keeps the complex treasury/liquidity work in a separate contract rather
  than mixing it into the customer-funds system.
- It defines specific tests for fund movements, permissions, claims, upgrades,
  contract failure, duplicate payments, and emergency pauses.
- It requires independent review of both the chain code and the contract code
  before activation.
- It can be piloted with controlled funding before relying on external revenue.

Buildable does not mean risk-free. It means there is a defined implementation,
test, audit, and launch path rather than an open-ended concept.

## Business case: what must be true

The program can be financially sustainable only if actual net USTC inflows are
enough to cover the rewards leadership chooses to fund, required reserves,
trading/liquidity costs, audits, engineering maintenance, monitoring, and any
buyback/burn allocation.

Before approving a launch budget, request a financial model that shows:

- each expected funding source and its evidence;
- gross inflow, transaction/routing costs, liquidity costs, and reserve needs;
- the amount available for rewards under low/base/high scenarios;
- one-time build/audit costs and ongoing operating costs;
- maximum exposure if revenue drops, a contract is paused, or participation is
  materially higher or lower than expected;
- the rule for reducing or stopping future rewards if funding is insufficient.

This plan intentionally contains no revenue, profit, or return forecast because
those values have not yet been independently established.

## Costs leadership should expect to authorize

- Core chain engineering and upgrade preparation.
- Treasury/liquidity contract engineering and reproducible build process.
- Independent security and accounting audits for both codebases.
- Economic modeling, testing, monitoring, reconciliation, and incident drills.
- Treasury or liquidity capital, external trading/liquidity expenses, and
  required legal, tax, or compliance work.
- Ongoing ownership of governance controls, authority changes, reporting, and
  software maintenance.

## Risks leadership would be accepting

| Risk | What it means | Planned control |
|---|---|---|
| Revenue risk | actual inflows may be too low or unreliable | rewards are limited to funded USTC; use pilot caps and reserve rules |
| Contract/liquidity risk | a treasury or liquidity contract may fail or need to pause | keep it separate from locked customer funds; audits, allow-lists, pause controls |
| Governance risk | a poor allocation or authority decision can damage trust | published limits, timelock, separate authorities, and event records |
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
4. Require a financial model and define the minimum evidence needed before
   Phase 2 revenue funding is activated.
5. Decide whether Phase 3 validator incentives are enabled at launch or held
   back until the core and funding stages prove stable.
6. Require all release gates: audits, testing, governance notice period,
   authority setup, monitoring, and incident rehearsal.

## Recommended leadership position

Approve planning and a tightly controlled Phase 1 pilot only after a budget,
ownership, and risk limit are agreed. Treat Phase 2 revenue funding and Phase
3 validator incentives as separate activation decisions—not automatic
consequences of building the core.

That approach preserves optionality: leadership can stop after a successful
pilot, delay the more complex revenue layer, or enable it only when the
business case and controls are ready.

## Questions leadership should ask the technical advisor

- Can any locked USTC be used to pay rewards, operating expenses, or buybacks?
  **No.** The design keeps those funds separate.
- Can the system pay more than it has funded? **No.** Claims are capped by the
  reward-pool balance.
- Does this alter LUNC validator control or create new token issuance? **No.**
- What happens if the treasury/liquidity contract fails? **New funding can be
  paused; the core lock and claim ledger remains separate.**
- What proof will we receive before launch? **Test results, audit reports,
  contract checksum, authority configuration, reconciliation dashboards, and
  pause/rollback drill evidence.**

## Reference material

- [Technical advisor briefing](technical-advisor-briefing.md)
- [Architecture foundation](../architecture-foundation.md)
- Approved Phase 1–3 specifications and implementation plans under
  `docs/superpowers/`.

This document is for internal strategy and risk discussion. It is not legal,
tax, accounting, investment, or financial advice.

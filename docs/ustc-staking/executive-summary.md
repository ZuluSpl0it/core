# USTC Staking: Executive Summary

**Status as of September 20, 2026**

## At a glance

USTC staking is being developed as a native Terra Classic feature: users stake
USTC directly from their own accounts, while rewards are paid only from USTC
that has already been funded. It does not change LUNC validator power or
standard validator staking.

Phase 1's core staking lifecycle has passed initial seven-validator functional
testing. Phase 2 adds governance-approved reward funding from the distribution
community pool. Its implementation and validator runbook are prepared; the
next step is independent testing on a seven-validator local network. A Phase 3
validator-incentive program has a design, but remains optional and has not been
implemented.

## What has been done

### Phase 1 — Native USTC staking

The native module supports direct USTC staking into configurable lock tiers,
reward claims, unbonding, and withdrawal after maturity. Principal and reward
balances are kept in separate module accounts. Governance controls module
parameters and pause/resume.

The initial local campaign, completed September 3, 2026, recorded a functional
pass on a seven-validator network. It exercised staking, reward funding and
share-weighted claims, governance changes, expected rejection cases,
unbonding and withdrawals, supply conservation, validator-state isolation, and
single-node and full-network restarts. Subsequent Phase 1 hardening expanded
state validation, owner queries, events, and application-level accounting
coverage. The Phase 2 validator campaign will retest the user lifecycle on the
current candidate while exercising the community-pool funding path.

### Phase 2 — Governance community-pool funding

The funding approach is now native and governance-controlled. Users continue
to stake directly with `MsgStake`; governance proposals execute
`MsgFundRewards` to transfer an approved amount of existing USTC from the
distribution community pool to the USTC reward pool. Reward accounting updates
only after that transfer succeeds. No reward is minted, and user principal is
not used to fund rewards.

The implementation and detailed validator runbook are ready for controlled
testing. The runbook covers successful funding, pool and supply reconciliation,
failure cases, claims, pause/resume, unbonding, withdrawals, and recovery after
node and network restarts.

## What is happening now

Independent validators are next being asked to run the Phase 2 community-pool
funding campaign on a disposable seven-validator local network. The runbook
includes the core staking lifecycle needed to verify that funding path. Their
evidence should identify the exact candidate revision, commands and proposals
tested, resulting balances and events, restart outcomes, and any unexpected
behavior.

After results arrive, the team will triage feedback, make any required changes,
and repeat affected tests on a clearly identified candidate. The next broader
milestone is production-style public-testnet preparation, including the
operator-led upgrade rehearsal and governance operating procedures. Local
functional testing is an important step toward that milestone, not activation
approval by itself.

## What may come next: Phase 3

Phase 3 is the final planned feature phase, but it is optional for a launch
offering native staking and community-pool-funded staker rewards. Its draft
design describes a separate validator incentive overlay: eligible validators
could enroll based on an active USTC position, and claim a share of a separately
funded performance pool based on published epoch scores. The overlay would not
change validator selection, LUNC self-bond, voting power, commission, slashing,
or jailing.

If governance and project leadership choose to proceed, the remaining work is
to approve the program policy and then implement and test it. Key decisions
include:

- which validators may participate and how any participation cap is set;
- the minimum active USTC position and ongoing eligibility rules;
- who reports performance, what evidence and scoring method are used, and how
  epochs are defined;
- how much may be assigned to a separate performance pool, and how payouts,
  rounding, pauses, and missed or ineligible claims are handled; and
- whether Phase 3 is included before initial activation or introduced later
  through a separately planned upgrade.

The draft currently uses an allow-list, a cap of up to 21 validators, a
1,000,000 USTC minimum position, and authority-submitted epoch scores as policy
starting points—not final commitments. Phase 3 should begin only after those
choices are explicitly approved. It can also be deferred without blocking the
Phase 1/2 native staking launch.

Other reward-funding sources or economic mechanisms are not part of the current
Phase 1/2 scope. Any future addition would need its own proposal and approval.

## Executive takeaway

The core user feature is built and has encouraging initial functional test
results. The team is now preparing independent validators to test the revised
governance funding path and the combined candidate locally. Validator feedback
will guide any remaining changes before public-testnet preparation. Phase 3 is
a defined but optional next opportunity—not a prerequisite for Phase 1/2.

## Related documents

- [Phase 2 community-pool validator campaign](phase-2-community-pool-localnet-testing.md)
- [Phase 2 community-pool release readiness](phase-2-community-pool-release-readiness.md)
- [C-suite decision briefing](briefings/c-suite-briefing.md)
- [Phase 3 validator-program design](../superpowers/specs/2026-07-29-ustc-staking-phase-3-validator-program-design.md)

# USTC Staking Phase 2 Community-Pool Release Readiness

This checklist defines evidence required before public-testnet or production
activation. Implementation completion alone is not release approval.

## Technical review — pending

- [ ] Governance authority validation for funding and parameter updates.
- [ ] FeePool `SafeSub` accounting and distribution-to-reward-pool
      module-to-module movement reviewed.
- [ ] Reward-pool direct account sends remain blocked.
- [ ] Community-pool decimal accounting and integer rounding reviewed.
- [ ] Transaction-cache rollback verified for every failure path.
- [ ] Principal and reward module accounts remain separate.
- [ ] Reward-index arithmetic and dust behavior reviewed.
- [ ] Direct staking, claim, unbonding, and withdrawal behavior unchanged.
- [ ] No mint, validator power, LUNC staking, slashing, or distribution-reward
      changes introduced.

Reviewer, revision, findings, and date:

```text
pending
```

## Controlled localnet campaign — pending

- [ ] Multi-validator successful governance funding and reconciliation.
- [ ] Wrong-denom, zero, no-active-shares, underfunded, unauthorized, and
      paused negative cases.
- [ ] Direct stake, claim, pause/resume, unbond, and withdrawal.
- [ ] Node restart and full-network restart.
- [ ] `terrad query ustcstaking validate-state` returns `.valid == true` before
      and after each campaign; real-BankKeeper integration checks cover every
      lifecycle path.
- [ ] Total USTC supply unchanged throughout the campaign.

Network, revision, evidence location, and date:

```text
pending
```

## Upgrade and operations — pending

- [ ] Production module activation status reconfirmed with authoritative
      chain/upgrade evidence; if already active, stop and approve a versioned
      state migration first.
- [ ] Live-chain upgrade rehearsal and rollback/recovery procedure completed.
- [ ] Governance proposal template, authority derivation, and reconciliation
      operator reviewed.
- [ ] Independent technical review completed and findings resolved.
- [ ] Release decision explicitly recorded by authorized stakeholders.

Until every applicable item is evidenced and approved, do not characterize
Phase 2 as production-ready or activate it on a public network.

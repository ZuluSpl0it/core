<!--
order: 3
-->

# EndBlock

## No periodic processing

The current module does not require an end-blocker. Lock maturity is checked against the block timestamp when the user submits `Withdraw`; no timer transaction, epoch transition, or automatic payout is required.

This differs from modules that settle rewards or policy at epoch boundaries. It keeps reward distribution deterministic around explicit funding transactions and avoids hidden periodic transfers. The trade-off is that a matured position remains in the `UNBONDING` state until its owner submits `Withdraw`.

The module is nevertheless registered in the application's begin/end blocker order lists because the Cosmos SDK requires every registered module to appear in those lists. Its blocker methods are no-ops.

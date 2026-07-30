## Abstract

The USTC Staking module allows users to lock `uusd` (USTC's on-chain denomination) for a governance-selected period and receive reward shares. Rewards are funded with existing USTC; the module does not mint USTC and does not burn deposited or reward coins.

The implementation uses explicit positions. A position records the user's principal, lock terms, reward-accounting snapshot, and lifecycle status. Users stake, begin unbonding, withdraw principal after maturity, and claim funded rewards.

## Contents

1. **[Concepts](01_concepts.md)**
   - [Lifecycle](01_concepts.md#position-lifecycle)
   - [Shares and lock tiers](01_concepts.md#shares-and-lock-tiers)
   - [Rewards](01_concepts.md#reward-accounting)
   - [No-minting model](01_concepts.md#no-minting-model)
2. **[State](02_state.md)**
   - [Positions](02_state.md#positions)
   - [Reward state](02_state.md#reward-state)
   - [Parameters](02_state.md#parameters)
3. **[EndBlock](03_end_block.md)**
   - [No periodic processing](03_end_block.md#no-periodic-processing)
4. **[Messages](04_messages.md)**
   - [User messages](04_messages.md#user-messages)
   - [Authority messages](04_messages.md#authority-messages)
5. **[Events](05_events.md)**
6. **[Parameters](06_params.md)**

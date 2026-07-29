# USTC Staking Phase 2 Funding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an audited CosmWasm TreasuryManager/POL funding adapter that can send realized USTC revenue to Phase 1 without gaining control of positions, rewards configuration, or user principal.

**Architecture:** `contracts/treasury-manager` is a Rust CosmWasm contract. It receives only allow-listed USTC revenue, records replay-safe receipts, applies a timelocked allocation policy, and emits one protobuf Stargate `MsgFundRewards` signed by its own contract address. `x/ustcstaking` verifies that address against `funding_authority`, transfers the contract's USTC into its reward pool, and updates the reward index atomically.

**Tech Stack:** Rust stable, `cosmwasm-std` with Stargate support compatible with wasmd `v0.61.8` / wasmvm `v3.0.3`, `cw-storage-plus`, `cw2`, `thiserror`, `prost`, `cw-multi-test`, Go, Cosmos SDK v0.50, Buf, Wasm keeper integration tests.

---

## Non-negotiable boundaries

- [ ] TreasuryManager may fund only through `/terra.ustcstaking.v1.MsgFundRewards`.
- [ ] TreasuryManager never receives a native staking private key or `authority` privileges.
- [ ] `funding_authority` is the only native permission granted to the contract.
- [ ] No module or contract mints USTC, alters LUNC validator state, or promises APR.
- [ ] `uusd` is the sole accepted revenue and funding denom.
- [ ] A failed native funding dispatch leaves the receipt unallocated and all balances unchanged.
- [ ] Phase 2 must not modify `custom/staking`, `x/staking`, distribution, mint, slashing, or validator governance paths.

## File structure

| File/group | Responsibility |
|---|---|
| `contracts/treasury-manager/` | isolated Rust contract crate and reproducible Wasm build |
| `contracts/treasury-manager/src/msg.rs` | instantiate/execute/query/migrate messages and response schema |
| `contracts/treasury-manager/src/state.rs` | config, policy, receipts, pending policy changes, allocation records |
| `contracts/treasury-manager/src/contract.rs` | authorization, receipt, allocation, pause, timelock, migration logic |
| `contracts/treasury-manager/src/stargate.rs` | minimal Prost definition and encoder for native `MsgFundRewards` |
| `contracts/treasury-manager/tests/` | multi-test behavior and replay/rollback cases |
| `proto/terra/ustcstaking/v1/tx.proto` | Phase 1 `funding_authority` and `MsgUpdateFundingAuthority` API |
| `x/ustcstaking/` | native authorization, funding transfer, events, and bridge tests |
| `wasmbinding/test/` | real chain-handler execution of the compiled contract |
| `docs/ustc-staking/` | operations, reconciliation, deployment, and audit evidence |

### Task 1: Establish an isolated, reproducible contract workspace

**Files:**
- Create: `contracts/treasury-manager/{Cargo.toml,Cargo.lock,rust-toolchain.toml}`
- Create: `contracts/treasury-manager/src/lib.rs`
- Modify: `.gitignore`
- Modify: `Makefile`

- [ ] **Step 1: Write the build contract.** Pin the Rust toolchain in
`rust-toolchain.toml`, define package metadata, and enable only
`cosmwasm-std`'s `stargate` feature. Keep the contract in this repository so
the native protobuf type URL, contract source, generated Wasm, and Go bridge
tests version together.

```toml
[dependencies]
cosmwasm-std = { version = "2", features = ["stargate"] }
cw-storage-plus = "2"
cw2 = "2"
prost = "0.13"
thiserror = "1"
```

- [ ] **Step 2: Add an ignored build target and Makefile targets.**

```make
contract-build:
	cargo build --release --target wasm32-unknown-unknown --manifest-path contracts/treasury-manager/Cargo.toml
contract-test:
	cargo test --manifest-path contracts/treasury-manager/Cargo.toml
```

Add `/contracts/treasury-manager/target/` to `.gitignore`; never ignore source,
`Cargo.lock`, or the explicitly versioned fixture Wasm used by Go tests.

- [ ] **Step 3: Run the empty crate test and commit.**

Run: `make contract-test`

```bash
git add .gitignore Makefile contracts/treasury-manager
git commit -m "build: add TreasuryManager contract workspace"
```

### Task 2: Define contract messages, state, and policy validation

**Files:**
- Create: `contracts/treasury-manager/src/{msg,state,error}.rs`
- Create: `contracts/treasury-manager/src/state_tests.rs`

- [ ] **Step 1: Write failing policy tests.**

```rust
#[test] fn rejects_non_uusd_receipt() {}
#[test] fn rejects_duplicate_receipt_id() {}
#[test] fn rejects_policy_with_treasury_share_below_500_or_above_5000_bps() {}
#[test] fn rejects_policy_change_with_less_than_thirty_days_delay() {}
```

- [ ] **Step 2: Define durable state.**

```rust
pub struct Config {
  pub governance: Addr,
  pub funding_module: String,
  pub paused: bool,
  pub code_id_guard: u64,
}
pub struct Policy {
  pub treasury_bps: u16,
  pub reward_bps: u16,
  pub buyback_bps: u16,
  pub activation_time: Timestamp,
}
pub struct Receipt { pub source: Addr, pub amount: Uint128, pub allocated: bool }
```

Require the three allocation shares to total 10,000 bps, treasury share to be
5–50%, a 30-day activation delay, valid non-empty addresses, and `uusd` only.
`reward_bps` defaults to zero: Phase 2 must not silently redirect the paper's
initial treasury/burn split into staking rewards.

- [ ] **Step 3: Add bounded state maps.** Use `cw_storage_plus::Map` keyed by
receipt ID and source address. Receipt IDs are supplied as non-empty byte
strings with a documented maximum length, preventing unbounded storage keys.

- [ ] **Step 4: Run tests and commit.**

Run: `make contract-test -- state_tests`

```bash
git add contracts/treasury-manager
git commit -m "feat: define TreasuryManager policy state"
```

### Task 3: Implement receipt intake and timelocked governance controls

**Files:**
- Create: `contracts/treasury-manager/src/contract.rs`
- Create: `contracts/treasury-manager/src/contract_tests.rs`

- [ ] **Step 1: Write authorization and pause tests.**

```rust
#[test] fn only_allowlisted_source_can_record_receipt() {}
#[test] fn paused_contract_rejects_receipts_and_allocations() {}
#[test] fn governance_can_schedule_but_not_early_execute_policy() {}
#[test] fn governance_can_remove_source_immediately() {}
```

- [ ] **Step 2: Implement instantiate and governance actions.** Instantiate
with governance, an initial allow-list, native module type URL, and policy.
Implement `SchedulePolicy`, `ExecutePolicy`, `AddSource`, `RemoveSource`,
`Pause`, `Unpause`, and `UpdateGovernance`. Only governance may issue them;
source removal and pause are immediate safety controls, while allocation and
code-policy changes are timelocked.

- [ ] **Step 3: Implement `RecordReceipt`.** Require exactly one attached
`uusd` coin, an allow-listed caller, an unused bounded receipt ID, and an
unpaused contract. Persist the receipt and emit `treasury_manager.receipt`
with ID, source, denom, and amount. Do not allocate at receipt time.

- [ ] **Step 4: Run tests and commit.**

Run: `make contract-test -- contract_tests`

```bash
git add contracts/treasury-manager
git commit -m "feat: add TreasuryManager receipt controls"
```

### Task 4: Encode the minimal native Stargate funding message

**Files:**
- Create: `contracts/treasury-manager/src/stargate.rs`
- Create: `contracts/treasury-manager/src/stargate_tests.rs`

- [ ] **Step 1: Write byte-level encoder tests.**

```rust
#[test]
fn fund_rewards_uses_the_native_type_url_and_contract_sender() {
  let msg = encode_fund_rewards("terra1contract", Uint128::new(42));
  assert_eq!(msg.type_url, "/terra.ustcstaking.v1.MsgFundRewards");
}
```

- [ ] **Step 2: Define only the required protobuf wire message.**

```rust
#[derive(Clone, PartialEq, prost::Message)]
pub struct MsgFundRewards {
  #[prost(string, tag = "1")] pub sender: String,
  #[prost(message, optional, tag = "2")] pub amount: Option<Coin>,
}
```

Use `CosmosMsg::Stargate { type_url, value }`; the sender is the contract
address from `env.contract.address`, never an execute caller supplied value.
This is deliberately not a new Terra custom binding: the existing custom
binding supports market swaps, while `custom/wasm/keeper/handler_plugin.go`
already dispatches SDK messages through the application router.

- [ ] **Step 3: Reject arbitrary type URLs.** Keep the encoder private and
make the contract emit only the fixed USTC funding type URL.

- [ ] **Step 4: Run tests and commit.**

Run: `make contract-test -- stargate_tests`

```bash
git add contracts/treasury-manager
git commit -m "feat: encode USTC funding stargate message"
```

### Task 5: Implement allocation and cross-layer reconciliation events

**Files:**
- Modify: `contracts/treasury-manager/src/{contract,msg,state}.rs`
- Create: `contracts/treasury-manager/src/allocation_tests.rs`
- Create: `docs/ustc-staking/phase-2-reconciliation.md`

- [ ] **Step 1: Write allocation tests.**

```rust
#[test] fn allocation_marks_receipt_only_after_stargate_message_is_emitted() {}
#[test] fn allocation_splits_exactly_by_bps_with_documented_dust_to_reserve() {}
#[test] fn second_allocation_of_same_receipt_fails() {}
#[test] fn zero_reward_bps_emits_no_funding_message() {}
```

- [ ] **Step 2: Implement `AllocateReceipt`.** Governance or a separately
timelocked executor may allocate only an unallocated receipt after policy
activation. Compute three integer shares; put division dust in reserve. Emit
bank sends only to fixed, governance-configured reserve and buyback executor
addresses. Emit exactly one fixed-type Stargate message for non-zero reward
share. Mark allocated in the same successful execution response.

- [ ] **Step 3: Write reconciliation documentation.** Specify the four matching
records: source transfer, `treasury_manager.receipt`,
`treasury_manager.allocation`, and `ustcstaking.reward_funded`. Define an
incident whenever receipt amount, allocation total, native funded amount, or
reward-pool balance differs.

- [ ] **Step 4: Run tests and commit.**

Run: `make contract-test -- allocation_tests`

```bash
git add contracts/treasury-manager docs/ustc-staking/phase-2-reconciliation.md
git commit -m "feat: allocate TreasuryManager revenue"
```

### Task 6: Align Phase 1 native authorization and events

**Files:**
- Modify: `proto/terra/ustcstaking/v1/{ustcstaking,tx,genesis}.proto`
- Modify: `x/ustcstaking/types/{params,codec,genesis,errors}.go`
- Modify: `x/ustcstaking/keeper/{msg_server,rewards,keeper}.go`
- Modify: `x/ustcstaking/keeper/{msg_server,rewards}_test.go`

- [ ] **Step 1: Write failing authorization tests.**

```go
func TestFundRewardsAcceptsOnlyFundingAuthority(t *testing.T) {}
func TestUpdateFundingAuthorityRequiresGovernanceAuthority(t *testing.T) {}
func TestFundingTransferAndIndexUpdateRollbackTogether(t *testing.T) {}
```

- [ ] **Step 2: Add `funding_authority`.** Extend params/genesis with a
bech32-validated funding authority. Add `MsgUpdateFundingAuthority` signed by
`authority`, include old/new addresses in its event, and initialize both
authorities to governance in Phase 1 genesis.

- [ ] **Step 3: Make `MsgFundRewards` contract-compatible.** Require
`msg.sender == params.funding_authority`; call
`SendCoinsFromAccountToModule(ctx, sender, RewardPoolName, uusd)` before reward
index mutation. Emit sender, amount, pool balance, and pre/post index. The
native code must not inspect contract storage or receipt IDs.

- [ ] **Step 4: Regenerate and verify.**

Run: `make proto-gen && go test ./x/ustcstaking/... -count=1`

```bash
git add proto/terra/ustcstaking x/ustcstaking
git commit -m "feat: authorize USTC funding adapter"
```

### Task 7: Prove execution through the real Wasm SDK message handler

**Files:**
- Create: `wasmbinding/test/ustc_staking_funding_test.go`
- Add generated fixture: `wasmbinding/testdata/treasury_manager.wasm`
- Modify: `contracts/treasury-manager/README.md`

- [ ] **Step 1: Build and checksum the fixture.** Build the contract with the
pinned toolchain; copy the release Wasm to the testdata path and record its
SHA-256 in the contract README. The fixture is versioned because Go tests need
the exact contract bytecode they execute.

- [ ] **Step 2: Write the chain-handler test.** Reuse `WasmTestSuite` setup to
store and instantiate the fixture, set the instantiated address as
`funding_authority`, send it USTC, record a receipt, activate a reward policy,
and allocate. Assert the native reward-pool balance and reward index increased
by the emitted reward share.

- [ ] **Step 3: Add negative cases.** Verify a non-authorized contract, wrong
denom, paused contract, duplicate receipt, and zero active shares all revert
the contract allocation and leave its receipt unallocated.

- [ ] **Step 4: Run bridge tests and commit.**

Run: `go test ./wasmbinding/test -run 'TestWasmTestSuite/TestUSTCStakingFunding' -count=1`

```bash
git add wasmbinding/test wasmbinding/testdata contracts/treasury-manager
git commit -m "test: cover TreasuryManager USTC funding bridge"
```

### Task 8: Complete contract migration, operational, and release gates

**Files:**
- Create: `contracts/treasury-manager/src/migrate.rs`
- Create: `contracts/treasury-manager/src/migrate_tests.rs`
- Create: `docs/ustc-staking/phase-2-release-readiness.md`

- [ ] **Step 1: Write migration tests.** Verify migration only accepts the
expected prior `cw2` contract name/version, preserves config/policy/receipts,
does not change governance or the native funding authority, and emits a
version event.

- [ ] **Step 2: Implement migration.** Use `cw2::set_contract_version` at
instantiate/migrate; reject unrecognized source versions. Migration must never
move funds or allocate a receipt.

- [ ] **Step 3: Write release readiness.** Require independent Rust and Go
audits, receipt/reconciliation monitoring, source allow-list review, contract
code checksum, funding-authority rotation procedure, 30-day timelock evidence,
pause drill, and a live rollback plan. State that native user withdrawal is
unaffected by a Phase 2 pause.

- [ ] **Step 4: Run complete verification.**

Run: `make contract-test && make contract-build && go test ./x/ustcstaking/... ./wasmbinding/test -count=1 && go test ./...`

Expected: contract, native module, real Wasm bridge, and repository tests pass;
`custom/staking` remains unmodified.

- [ ] **Step 5: Refresh local knowledge only and commit.**

Run: `scripts/graphify-refresh coding && python3 -m tools.graphify.validate graphify-out/coding --profile coding && git check-ignore graphify-out/coding/graph.json`

```bash
git add contracts/treasury-manager docs/ustc-staking proto/terra/ustcstaking x/ustcstaking wasmbinding/test wasmbinding/testdata
git commit -m "docs: verify USTC staking phase 2"
```

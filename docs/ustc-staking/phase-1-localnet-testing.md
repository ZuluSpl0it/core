# USTC Staking Phase 1 Localnet Test Plan

**Goal:** Prove that the Phase 1 `x/ustcstaking` module can be configured by governance, lock USTC, distribute only pre-funded USTC rewards by snapshotted shares, claim rewards, unbond, and return principal without changing LUNC validator staking.

**Architecture:** The test uses the repository's seven-validator Docker Compose localnet. Governance configures two short lock tiers and appoints node 0 as the local funding authority. Two validator accounts stake equal USTC principal with different multipliers, then node 0 transfers pre-existing USTC into the reward pool so the expected 1:2 payout can be verified exactly.

**Tech stack:** `terrad`, Docker Compose, Bash, `jq`, `curl`, and the local Cosmos SDK test keyring.

---

## Scope

This runbook tests the Phase 1 code currently present on branch `ustc_staking`:

- governance-controlled parameters;
- USTC-only (`uusd`) positions;
- lock-tier duration and multiplier snapshots;
- separate principal and reward module accounts;
- governance-approved funding authority and pre-funded rewards;
- proportional reward claims;
- unbonding and matured principal withdrawal;
- expected failure paths;
- isolation from normal validator staking.

It does not test TreasuryManager/POL funding or the validator incentive program. Those are Phase 2 and Phase 3 designs and are not implemented in this branch.

## Important behavior

- `1 USTC = 1,000,000 uusd`.
- Lock duration starts when `begin-unbonding` executes, not when `stake` executes.
- Active positions earn according to `principal × tier multiplier`.
- Funding fails when no active shares exist.
- Rewards are never minted. `MsgFundRewards` transfers existing `uusd` from the configured funding authority into `ustcstaking_reward_pool`.
- Active-position queries currently show stored `claimable_rewards`, not dynamically calculated pending rewards. Verify active rewards through `reward-state`, reward-pool balance, and actual claim balance deltas.
- This procedure creates a fresh localnet. `make localnet-stop` deletes `build/node*` and `build/gentxs`.

## Test values

| Item | Value |
|---|---:|
| Validators | 7 |
| Chain ID | `localterra` |
| Block commit timeout | `500ms` |
| Governance voting period | `30s` |
| USTC seeded per validator | `1,000,000 USTC` (`1000000000000uusd`) |
| Tier 1 | ID `1`, 5-second unbonding, `1.0x` shares |
| Tier 2 | ID `2`, 10-second unbonding, `2.0x` shares |
| Node 0 stake | `100 USTC` (`100000000uusd`) in tier 1 |
| Node 1 stake | `100 USTC` (`100000000uusd`) in tier 2 |
| Reward funding | `300 USTC` (`300000000uusd`) |
| Expected node 0 reward | `100 USTC` (`100000000uusd`) |
| Expected node 1 reward | `200 USTC` (`200000000uusd`) |

## 1. Prerequisites

Run from the repository root:

```bash
cd /home/sofoli/core
git branch --show-current
git status --short
```

Expected branch: `ustc_staking`.

Confirm tools and Docker daemon:

```bash
go version
docker version
docker compose version
jq --version
curl --version
```

`docker version` must show both client and server sections.

## 2. Build the Linux binary and localnet image

Warning: the first command removes existing generated localnet node data.

```bash
make localnet-stop
make build-linux
```

Verify binary:

```bash
file build/terrad
./build/terrad version --long --home /tmp/ustc-staking-version-check \
  | rg '^(name|server_name|version|commit|build_tags):'
```

Expected binary: Linux x86-64 executable with `server_name: terrad`.

Build the localnet runtime image when absent:

```bash
docker image inspect classic-terra/terrad-env >/dev/null 2>&1 \
  || make -C contrib/localnet terrad-env
```

## 3. Generate seven validator homes without starting containers

This reproduces the generation stage of `make localnet-start` but pauses before `docker compose up`, allowing genesis and timing changes.

```bash
docker run --platform linux/amd64 --rm \
  --user "$(id -u):$(id -g)" \
  -v "$PWD/build:/terrad:Z" \
  -v /etc/group:/etc/group:ro \
  -v /etc/passwd:/etc/passwd:ro \
  -v /etc/shadow:/etc/shadow:ro \
  classic-terra/terrad-env \
  testnet \
  --chain-id localterra \
  --v 7 \
  -o . \
  --starting-ip-address 192.168.10.2 \
  --keyring-backend test
```

Expected directories:

```bash
find build -maxdepth 2 -type d -name terrad | sort
```

Expected: `build/node0/terrad` through `build/node6/terrad`.

## 4. Seed USTC and shorten governance timing in genesis

The stock `terrad testnet` generator funds `stake` and per-node test denoms, but it does not fund `uusd`. Add `1,000,000 USTC` to every validator and update total supply in the canonical node 0 genesis.

```bash
GENESIS="$PWD/build/node0/terrad/config/genesis.json"

jq '
  (.app_state.bank.balances[].coins) += [
    {"denom":"uusd","amount":"1000000000000"}
  ]
  | (.app_state.bank.balances[].coins) |= sort_by(.denom)
  | .app_state.bank.supply += [
      {"denom":"uusd","amount":"7000000000000"}
    ]
  | .app_state.bank.supply |= sort_by(.denom)
  | .app_state.gov.params.voting_period = "30s"
  | .app_state.gov.params.expedited_voting_period = "10s"
  | .app_state.gov.params.max_deposit_period = "30s"
' "$GENESIS" > "$GENESIS.tmp"

mv "$GENESIS.tmp" "$GENESIS"

for i in {1..6}; do
  cp "$GENESIS" "$PWD/build/node${i}/terrad/config/genesis.json"
done
```

Verify supply, accounts, governance timing, and default USTC-staking state:

```bash
jq '.app_state.bank.supply[] | select(.denom == "uusd")' "$GENESIS"
jq '[.app_state.bank.balances[] | .coins[] | select(.denom == "uusd")] | length' "$GENESIS"
jq '.app_state.gov.params | {max_deposit_period, voting_period, expedited_voting_period, min_deposit}' "$GENESIS"
jq '.app_state.ustcstaking' "$GENESIS"

for i in {1..6}; do
  cmp "$GENESIS" "$PWD/build/node${i}/terrad/config/genesis.json"
done
```

Expected:

- USTC supply: `7000000000000uusd`;
- seven USTC-funded validator accounts;
- empty `lock_tiers` before governance;
- zero reward index and shares;
- every validator has identical genesis.

The current binary's standalone `validate-genesis` command panics inside the Cosmos SDK genutil validator because its message-validator callback is nil. Do not use that command as this run's gate. Successful InitChain in section 6 is the effective full-genesis validation.

## 5. Configure 500 ms block commits

```bash
for config_file in build/node*/terrad/config/config.toml; do
  sed -i 's/^timeout_commit = ".*"/timeout_commit = "500ms"/' "$config_file"
done

rg '^timeout_commit' build/node*/terrad/config/config.toml
```

Expected: every node reports `timeout_commit = "500ms"`.

## 6. Start and verify the localnet

```bash
docker compose up -d
docker compose ps
```

Wait for node 0 RPC:

```bash
until curl -fsS http://localhost:26657/status \
  | jq -e '.result.sync_info.catching_up == false' >/dev/null; do
  sleep 1
done
```

Verify block production:

```bash
HEIGHT_1=$(curl -fsS http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
sleep 3
HEIGHT_2=$(curl -fsS http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
printf 'height before=%s after=%s\n' "$HEIGHT_1" "$HEIGHT_2"
test "$HEIGHT_2" -gt "$HEIGHT_1"
```

If a node exits, inspect logs:

```bash
docker compose ps -a
docker compose logs --tail=100 terradnode0
```

## 7. Define session helpers

These helpers use node 0 RPC while signing from each validator's own test keyring.

```bash
export TERRAD="$PWD/build/terrad"
export CHAIN_ID="localterra"
export RPC="tcp://localhost:26657"
export KEYRING_BACKEND="test"
export NODE0_HOME="$PWD/build/node0/terrad"
export TX_FEE="1000000stake"

QUERY_FLAGS=(
  --chain-id "$CHAIN_ID"
  --node "$RPC"
  --home "$NODE0_HOME"
  --output json
)

TX_FLAGS=(
  --chain-id "$CHAIN_ID"
  --node "$RPC"
  --keyring-backend "$KEYRING_BACKEND"
  --gas auto
  --gas-adjustment 1.5
  --fees "$TX_FEE"
  --broadcast-mode sync
  --yes
  --output json
)

wait_for_tx() {
  local tx_hash="$1"
  local result
  for _ in $(seq 1 40); do
    if result=$("$TERRAD" query tx "$tx_hash" "${QUERY_FLAGS[@]}" 2>/dev/null); then
      printf '%s\n' "$result"
      return 0
    fi
    sleep 0.5
  done
  printf 'transaction %s was not indexed in time\n' "$tx_hash" >&2
  return 1
}

submit_tx() {
  local output check_code tx_hash
  if ! output=$("$@" "${TX_FLAGS[@]}" 2>&1); then
    printf '%s\n' "$output"
    return 1
  fi
  if ! jq -e . >/dev/null 2>&1 <<<"$output"; then
    printf '%s\n' "$output"
    return 1
  fi
  check_code=$(jq -r '.code // 0' <<<"$output")
  if [ "$check_code" != "0" ]; then
    printf '%s\n' "$output"
    return 0
  fi
  tx_hash=$(jq -r '.txhash // empty' <<<"$output")
  if [ -z "$tx_hash" ]; then
    printf '%s\n' "$output"
    return 1
  fi
  wait_for_tx "$tx_hash"
}

expect_ok() {
  local result code
  if ! result=$(submit_tx "$@"); then
    printf 'UNEXPECTED CLI FAILURE\n%s\n' "$result" >&2
    return 1
  fi
  printf '%s\n' "$result" | jq '{height, txhash, code, codespace, raw_log}'
  code=$(jq -r '.code // 0' <<<"$result")
  test "$code" = "0"
}

expect_fail() {
  local result code
  if ! result=$(submit_tx "$@"); then
    printf 'EXPECTED CLI/CHECKTX FAILURE\n%s\n' "$result"
    return 0
  fi
  printf '%s\n' "$result" | jq '{height, txhash, code, codespace, raw_log}'
  code=$(jq -r '.code // 0' <<<"$result")
  test "$code" != "0"
}

latest_proposal_id() {
  "$TERRAD" query gov proposals "${QUERY_FLAGS[@]}" \
    | jq -r '[.proposals[].id | tonumber] | max'
}

vote_all_yes() {
  local proposal_id="$1"
  local i
  for i in {0..6}; do
    expect_ok "$TERRAD" tx gov vote "$proposal_id" yes \
      --from "node${i}" \
      --home "$PWD/build/node${i}/terrad"
  done
}

wait_for_proposal() {
  local proposal_id="$1"
  local result status
  for _ in $(seq 1 90); do
    result=$("$TERRAD" query gov proposal "$proposal_id" "${QUERY_FLAGS[@]}")
    status=$(jq -r '.proposal.status' <<<"$result")
    printf 'proposal %s: %s\n' "$proposal_id" "$status" >&2
    case "$status" in
      PROPOSAL_STATUS_PASSED|PROPOSAL_STATUS_REJECTED|PROPOSAL_STATUS_FAILED)
        printf '%s\n' "$result"
        return 0
        ;;
    esac
    sleep 1
  done
  return 1
}

uusd_balance() {
  "$TERRAD" query bank balance "$1" uusd "${QUERY_FLAGS[@]}" \
    | jq -r '.balance.amount // "0"'
}

module_address() {
  "$TERRAD" query auth module-account "$1" "${QUERY_FLAGS[@]}" \
    | jq -r '.account.base_account.address // .account.base_vesting_account.base_account.address // empty'
}
```

## 8. Verify keys, balances, and validator baseline

```bash
for i in {0..6}; do
  node_home="$PWD/build/node${i}/terrad"
  address=$("$TERRAD" keys show "node${i}" -a \
    --keyring-backend "$KEYRING_BACKEND" \
    --home "$node_home")
  printf 'node%s %s %s uusd\n' "$i" "$address" "$(uusd_balance "$address")"
done
```

Expected: every address has `1000000000000uusd`.

Save normal validator-staking state for the final isolation check:

```bash
"$TERRAD" query staking validators "${QUERY_FLAGS[@]}" \
  | jq '[.validators[] | {
      operator_address,
      status,
      jailed,
      tokens,
      delegator_shares
    }] | sort_by(.operator_address)' \
  > /tmp/ustc-phase1-validators-before.json
```

Define test addresses and module accounts:

```bash
NODE0_ADDR=$("$TERRAD" keys show node0 -a --keyring-backend test --home "$PWD/build/node0/terrad")
NODE1_ADDR=$("$TERRAD" keys show node1 -a --keyring-backend test --home "$PWD/build/node1/terrad")

GOV_AUTH=$("$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" | jq -r '.params.authority')
FUNDING_AUTH=$("$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" | jq -r '.params.funding_authority')
PRINCIPAL_POOL=$(module_address ustcstaking)
REWARD_POOL=$(module_address ustcstaking_reward_pool)

printf 'node0=%s\nnode1=%s\ngov=%s\nfunding=%s\nprincipal_pool=%s\nreward_pool=%s\n' \
  "$NODE0_ADDR" "$NODE1_ADDR" "$GOV_AUTH" "$FUNDING_AUTH" "$PRINCIPAL_POOL" "$REWARD_POOL"

test "$GOV_AUTH" = "$FUNDING_AUTH"
test -n "$PRINCIPAL_POOL"
test -n "$REWARD_POOL"
test "$PRINCIPAL_POOL" != "$REWARD_POOL"
```

## 9. Configure lock tiers through governance

The module starts with no tiers. Create a governance proposal carrying `MsgUpdateParams`. The complete parameter set must be supplied because this message replaces all params.

```bash
jq -n --arg authority "$GOV_AUTH" --arg funding_authority "$NODE0_ADDR" '
{
  messages: [
    {
      "@type": "/terra.ustcstaking.v1.MsgUpdateParams",
      authority: $authority,
      params: {
        bond_denom: "uusd",
        lock_tiers: [
          {
            id: 1,
            duration: "5s",
            multiplier: "1.000000000000000000"
          },
          {
            id: 2,
            duration: "10s",
            multiplier: "2.000000000000000000"
          }
        ],
        authority: $authority,
        funding_authority: $funding_authority,
        paused: false
      }
    }
  ],
  metadata: "",
  deposit: "10000000stake",
  title: "Configure USTC staking test tiers",
  summary: "Enable short USTC staking tiers for local Phase 1 testing",
  expedited: false
}
' > /tmp/ustc-params-proposal.json

jq . /tmp/ustc-params-proposal.json
```

Submit, vote, and wait:

```bash
expect_ok "$TERRAD" tx gov submit-proposal /tmp/ustc-params-proposal.json \
  --from node0 \
  --home "$PWD/build/node0/terrad"

PARAMS_PROPOSAL_ID=$(latest_proposal_id)
vote_all_yes "$PARAMS_PROPOSAL_ID"
PARAMS_RESULT=$(wait_for_proposal "$PARAMS_PROPOSAL_ID")
printf '%s\n' "$PARAMS_RESULT" | jq '.proposal | {id, status, title}'
test "$(jq -r '.proposal.status' <<<"$PARAMS_RESULT")" = "PROPOSAL_STATUS_PASSED"
```

Verify applied parameters:

```bash
"$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" | jq .
```

Expected: `bond_denom=uusd`, two tiers, governance authority unchanged, funding authority set to node 0, and `paused=false`.

```bash
FUNDING_AUTH=$("$TERRAD" query ustcstaking params "${QUERY_FLAGS[@]}" | jq -r '.params.funding_authority')
test "$FUNDING_AUTH" = "$NODE0_ADDR"
```

## 10. Check rejection paths before staking

Wrong denomination:

```bash
expect_fail "$TERRAD" tx ustcstaking stake 1000000stake 1 \
  --from node0 \
  --home "$PWD/build/node0/terrad"
```

Unknown tier:

```bash
expect_fail "$TERRAD" tx ustcstaking stake 1000000uusd 99 \
  --from node0 \
  --home "$PWD/build/node0/terrad"
```

Verify neither failure created a position:

```bash
"$TERRAD" query ustcstaking positions "$NODE0_ADDR" "${QUERY_FLAGS[@]}" | jq .
```

Expected: empty `positions` array.

## 11. Create two weighted positions

Node 0 stakes 100 USTC at `1.0x`:

```bash
expect_ok "$TERRAD" tx ustcstaking stake 100000000uusd 1 \
  --from node0 \
  --home "$PWD/build/node0/terrad"
```

Node 1 stakes 100 USTC at `2.0x`:

```bash
expect_ok "$TERRAD" tx ustcstaking stake 100000000uusd 2 \
  --from node1 \
  --home "$PWD/build/node1/terrad"
```

Resolve position IDs from owner queries:

```bash
NODE0_POSITION=$("$TERRAD" query ustcstaking positions "$NODE0_ADDR" "${QUERY_FLAGS[@]}" | jq -r '.positions[-1].id')
NODE1_POSITION=$("$TERRAD" query ustcstaking positions "$NODE1_ADDR" "${QUERY_FLAGS[@]}" | jq -r '.positions[-1].id')

printf 'node0 position=%s\nnode1 position=%s\n' "$NODE0_POSITION" "$NODE1_POSITION"

"$TERRAD" query ustcstaking position "$NODE0_POSITION" "${QUERY_FLAGS[@]}" | jq .
"$TERRAD" query ustcstaking position "$NODE1_POSITION" "${QUERY_FLAGS[@]}" | jq .
"$TERRAD" query ustcstaking reward-state "${QUERY_FLAGS[@]}" | jq .
```

Expected:

- both positions are `POSITION_STATUS_ACTIVE`;
- node 0 principal is `100000000uusd`, shares are `100000000`, multiplier is `1.0`;
- node 1 principal is `100000000uusd`, shares are `200000000`, multiplier is `2.0`;
- total active shares are `300000000`;
- reward index remains zero.

Verify principal custody:

```bash
test "$(uusd_balance "$PRINCIPAL_POOL")" = "200000000"
test "$(uusd_balance "$REWARD_POOL")" = "0"
```

## 12. Fund rewards from the governance-approved funding authority

The module intentionally omits authority-only funding from its ordinary user CLI. Create a protobuf JSON transaction containing `MsgFundRewards`, sign it with node 0—the funding authority governance approved in section 9—and broadcast it using `terrad`.

Define a reusable raw funding-transaction builder:

```bash
make_fund_tx() {
  local sender="$1"
  local key_name="$2"
  local key_home="$3"
  local amount="$4"
  local output_prefix="$5"

  jq -n --arg sender "$sender" --arg amount "$amount" '
  {
    body: {
      messages: [
        {
          "@type": "/terra.ustcstaking.v1.MsgFundRewards",
          sender: $sender,
          amount: {denom: "uusd", amount: $amount}
        }
      ],
      memo: "",
      timeout_height: "0",
      extension_options: [],
      non_critical_extension_options: []
    },
    auth_info: {
      signer_infos: [],
      fee: {
        amount: [{denom: "stake", amount: "1000000"}],
        gas_limit: "300000",
        payer: "",
        granter: ""
      }
    },
    signatures: []
  }
  ' > "${output_prefix}-unsigned.json"

  "$TERRAD" tx sign "${output_prefix}-unsigned.json" \
    --from "$key_name" \
    --home "$key_home" \
    --keyring-backend "$KEYRING_BACKEND" \
    --chain-id "$CHAIN_ID" \
    --node "$RPC" \
    --output json \
    --output-document "${output_prefix}-signed.json" \
    --overwrite
}
```

First prove that node 1 cannot fund rewards:

```bash
make_fund_tx \
  "$NODE1_ADDR" \
  node1 \
  "$PWD/build/node1/terrad" \
  1000 \
  /tmp/ustc-unauthorized-fund

expect_fail "$TERRAD" tx broadcast /tmp/ustc-unauthorized-fund-signed.json \
  --home "$NODE0_HOME"

test "$(uusd_balance "$REWARD_POOL")" = "0"
```

Fund `300000000uusd` from authorized node 0:

```bash
make_fund_tx \
  "$NODE0_ADDR" \
  node0 \
  "$PWD/build/node0/terrad" \
  300000000 \
  /tmp/ustc-authorized-fund

expect_ok "$TERRAD" tx broadcast /tmp/ustc-authorized-fund-signed.json \
  --home "$NODE0_HOME"
```

Verify funded state:

```bash
"$TERRAD" query ustcstaking reward-state "${QUERY_FLAGS[@]}" | jq .
test "$(uusd_balance "$REWARD_POOL")" = "300000000"
```

Expected reward state:

- total shares remain `300000000`;
- reward index becomes `1.000000000000000000`;
- node 0's existing USTC moved to the reward pool;
- principal pool remains `200000000uusd`.

## 13. Claim rewards and verify exact 1:2 allocation

Record balances immediately before claims:

```bash
NODE0_BEFORE_CLAIM=$(uusd_balance "$NODE0_ADDR")
NODE1_BEFORE_CLAIM=$(uusd_balance "$NODE1_ADDR")
```

Claim both positions:

```bash
expect_ok "$TERRAD" tx ustcstaking claim-rewards "$NODE0_POSITION" \
  --from node0 \
  --home "$PWD/build/node0/terrad"

expect_ok "$TERRAD" tx ustcstaking claim-rewards "$NODE1_POSITION" \
  --from node1 \
  --home "$PWD/build/node1/terrad"
```

Verify balance deltas:

```bash
NODE0_AFTER_CLAIM=$(uusd_balance "$NODE0_ADDR")
NODE1_AFTER_CLAIM=$(uusd_balance "$NODE1_ADDR")

test "$((NODE0_AFTER_CLAIM - NODE0_BEFORE_CLAIM))" -eq 100000000
test "$((NODE1_AFTER_CLAIM - NODE1_BEFORE_CLAIM))" -eq 200000000
test "$(uusd_balance "$REWARD_POOL")" = "0"
test "$(uusd_balance "$PRINCIPAL_POOL")" = "200000000"
```

Claim node 0 again. It should succeed with zero payout and no balance change:

```bash
NODE0_BEFORE_EMPTY_CLAIM=$(uusd_balance "$NODE0_ADDR")

expect_ok "$TERRAD" tx ustcstaking claim-rewards "$NODE0_POSITION" \
  --from node0 \
  --home "$PWD/build/node0/terrad"

test "$(uusd_balance "$NODE0_ADDR")" = "$NODE0_BEFORE_EMPTY_CLAIM"
```

## 14. Test ownership, unbonding, maturity, and withdrawal

Node 0 must not control node 1's position:

```bash
expect_fail "$TERRAD" tx ustcstaking begin-unbonding "$NODE1_POSITION" \
  --from node0 \
  --home "$PWD/build/node0/terrad"
```

Begin node 0's 5-second unbonding period:

```bash
expect_ok "$TERRAD" tx ustcstaking begin-unbonding "$NODE0_POSITION" \
  --from node0 \
  --home "$PWD/build/node0/terrad"

"$TERRAD" query ustcstaking position "$NODE0_POSITION" "${QUERY_FLAGS[@]}" \
  | jq '.position | {id, status, shares, unbonding_end_time, principal, claimable_rewards}'
```

Expected: status `POSITION_STATUS_UNBONDING`, shares `0`, principal unchanged, and a future maturity timestamp.

Immediate withdrawal must fail:

```bash
expect_fail "$TERRAD" tx ustcstaking withdraw "$NODE0_POSITION" \
  --from node0 \
  --home "$PWD/build/node0/terrad"
```

Wait for tier 1 maturity, then withdraw:

```bash
sleep 6
NODE0_BEFORE_WITHDRAW=$(uusd_balance "$NODE0_ADDR")

expect_ok "$TERRAD" tx ustcstaking withdraw "$NODE0_POSITION" \
  --from node0 \
  --home "$PWD/build/node0/terrad"

NODE0_AFTER_WITHDRAW=$(uusd_balance "$NODE0_ADDR")
test "$((NODE0_AFTER_WITHDRAW - NODE0_BEFORE_WITHDRAW))" -eq 100000000
```

Duplicate withdrawal must fail:

```bash
expect_fail "$TERRAD" tx ustcstaking withdraw "$NODE0_POSITION" \
  --from node0 \
  --home "$PWD/build/node0/terrad"
```

Repeat with node 1's 10-second tier:

```bash
expect_ok "$TERRAD" tx ustcstaking begin-unbonding "$NODE1_POSITION" \
  --from node1 \
  --home "$PWD/build/node1/terrad"

sleep 11
NODE1_BEFORE_WITHDRAW=$(uusd_balance "$NODE1_ADDR")

expect_ok "$TERRAD" tx ustcstaking withdraw "$NODE1_POSITION" \
  --from node1 \
  --home "$PWD/build/node1/terrad"

NODE1_AFTER_WITHDRAW=$(uusd_balance "$NODE1_ADDR")
test "$((NODE1_AFTER_WITHDRAW - NODE1_BEFORE_WITHDRAW))" -eq 100000000
```

Verify final positions and custody:

```bash
"$TERRAD" query ustcstaking position "$NODE0_POSITION" "${QUERY_FLAGS[@]}" | jq .
"$TERRAD" query ustcstaking position "$NODE1_POSITION" "${QUERY_FLAGS[@]}" | jq .
"$TERRAD" query ustcstaking reward-state "${QUERY_FLAGS[@]}" | jq .

test "$(uusd_balance "$PRINCIPAL_POOL")" = "0"
test "$(uusd_balance "$REWARD_POOL")" = "0"
```

Expected:

- both positions are `POSITION_STATUS_WITHDRAWN`;
- both position principals are zero;
- total active shares are zero;
- all funded rewards were claimed;
- both pools are empty.

## 15. Verify funding rejection with zero active shares

Create another authorized funding transaction while total active shares are zero:

```bash
NODE0_BEFORE_ZERO_SHARE_FUND=$(uusd_balance "$NODE0_ADDR")

make_fund_tx \
  "$NODE0_ADDR" \
  node0 \
  "$PWD/build/node0/terrad" \
  1000 \
  /tmp/ustc-zero-shares-fund

expect_fail "$TERRAD" tx broadcast /tmp/ustc-zero-shares-fund-signed.json \
  --home "$NODE0_HOME"
```

Expected: transaction fails with `no active shares`, reward state and reward pool remain unchanged, and node 0 retains the `1000uusd` because execution is atomic.

```bash
test "$("$TERRAD" query ustcstaking reward-state "${QUERY_FLAGS[@]}" | jq -r '.reward_state.total_shares')" = "0"
test "$(uusd_balance "$REWARD_POOL")" = "0"
test "$(uusd_balance "$NODE0_ADDR")" = "$NODE0_BEFORE_ZERO_SHARE_FUND"
```

## 16. Prove validator-staking isolation

Capture the same validator fields and compare them with the baseline:

```bash
"$TERRAD" query staking validators "${QUERY_FLAGS[@]}" \
  | jq '[.validators[] | {
      operator_address,
      status,
      jailed,
      tokens,
      delegator_shares
    }] | sort_by(.operator_address)' \
  > /tmp/ustc-phase1-validators-after.json

diff -u \
  /tmp/ustc-phase1-validators-before.json \
  /tmp/ustc-phase1-validators-after.json
```

Expected: no differences. USTC positions must not change validator power, LUNC/stake delegation, status, or jailing.

Also confirm module accounts have no mint or burn permissions:

```bash
"$TERRAD" query auth module-account ustcstaking "${QUERY_FLAGS[@]}" | jq '.account | {name, permissions}'
"$TERRAD" query auth module-account ustcstaking_reward_pool "${QUERY_FLAGS[@]}" | jq '.account | {name, permissions}'
```

Expected: empty permissions arrays.

## 17. Pass criteria

Phase 1 passes this local session when all conditions hold:

- all seven containers remain running and blocks advance;
- governance installs both lock tiers;
- wrong denom and unknown tier are rejected;
- equal principal with `1x` and `2x` tiers produces `100M` and `200M` shares;
- governance-approved node 0 funding of `300M uusd` moves into the reward pool before allocation;
- reward index becomes exactly `1.0`;
- claims pay exactly `100M` and `200M uusd`;
- second claim pays zero;
- wrong-owner and premature withdrawals fail;
- each matured withdrawal returns exactly `100M uusd` principal;
- duplicate withdrawal fails;
- funding with zero active shares fails atomically;
- principal and reward pools reconcile to zero after completion;
- validator staking state remains unchanged;
- neither USTC module account has mint or burn permissions.

## 18. Preserve or reset the environment

Stop containers while preserving generated state:

```bash
docker compose stop
```

Restart preserved state:

```bash
docker compose start
```

Destroy generated node state and containers:

```bash
make localnet-stop
```

`make localnet-stop` removes `build/node*` and `build/gentxs`. It does not remove source code.

## Troubleshooting

### Docker daemon unavailable

```text
Cannot connect to the Docker daemon at unix:///var/run/docker.sock
```

Start Docker, then rerun the failed command.

### Binary rejected by container

```text
Binary needs to be OS linux, ARCH amd64
```

Run `make build-linux`; do not substitute a macOS or ARM binary for `build/terrad`.

### Host client reads a broken default config

Every command in this runbook supplies `--home`. Keep using the generated node home or the isolated `/tmp` home for version checks.

### Proposal remains in voting

```bash
"$TERRAD" query gov proposal "$PARAMS_PROPOSAL_ID" "${QUERY_FLAGS[@]}" | jq .
"$TERRAD" query gov votes "$PARAMS_PROPOSAL_ID" "${QUERY_FLAGS[@]}" | jq .
```

Check that the initial deposit reached `10000000stake` and that validator vote transactions have code zero.

### Transaction not found

The helper waits up to 20 seconds for transaction indexing. Confirm blocks advance and inspect node 0 logs:

```bash
curl -fsS http://localhost:26657/status | jq '.result.sync_info'
docker compose logs --tail=100 terradnode0
```

## Future Phase 2 test inventory

Phase 2 will test TreasuryManager/POL funding integration. Planned coverage:

1. Only allow-listed revenue sources can record `uusd` receipts.
2. Wrong denoms, malformed receipt IDs, and duplicate receipt IDs are rejected.
3. Receipt, reserve, reward, and buyback allocations sum exactly to received funds; integer dust follows the documented policy.
4. Allocation-policy changes cannot execute before the governance timelock.
5. Governance can rotate or revoke `funding_authority` without changing the module governance authority.
6. TreasuryManager emits the fixed `/terra.ustcstaking.v1.MsgFundRewards` Stargate message with its own contract address as sender.
7. Non-authorized contracts cannot fund the native reward pool.
8. Failed native dispatch rolls back the contract allocation, receipt state, balances, and native reward index atomically.
9. Contract pause stops new receipts and allocations but does not block Phase 1 claims or withdrawals.
10. Source transfers, contract receipt/allocation events, native funding events, and reward-pool balances reconcile exactly.
11. Contract migration preserves governance, policy, allow-list, receipts, balances, and funding authorization.
12. No Phase 2 operation can access user principal, mint USTC, or alter validator staking.

## Future Phase 3 test inventory

Phase 3 will test the optional validator incentive overlay. Planned coverage:

1. Enrollment requires an allow-listed, active validator and an eligible active Phase 1 USTC position owned by the matching account.
2. Enrollment cap and minimum USTC self-bond are enforced.
3. Enrollment does not create validators or change power, commission, LUNC self-bond, slashing, or jailing.
4. Beginning Phase 1 unbonding removes future program eligibility without blocking withdrawal rights.
5. Performance pool accepts only governed, pre-funded `uusd`; no mint or distribution-module funds are used.
6. Epoch finalization rejects duplicate epochs, duplicate validators, invalid intervals, invalid scores, and allocations above pool balance.
7. Ineligible validators receive no allocation at finalization.
8. Valid scores produce deterministic proportional allocations; rounding dust remains in the performance pool.
9. A validator can claim its own allocation exactly once.
10. Program pause blocks enrollment and epoch actions but not Phase 1 claims, unbonding, or withdrawals.
11. Performance-authority rotation and program parameter changes obey governance authorization and timelocks.
12. Genesis export/import and upgrade rehearsal preserve enrollments, epochs, allocations, claims, and pool accounting.
13. The `ustc_staking` upgrade leaves existing LUNC validator state unchanged.

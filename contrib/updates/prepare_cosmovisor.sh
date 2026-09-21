#!/bin/bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "$SCRIPT_DIR/old_terrad_cache.sh"

# this bash will prepare cosmosvisor to the build folder so that it can perform upgrade
# this script is supposed to be run by Makefile

# These fields should be fetched automatically in the future
# Need to do more upgrade to see upgrade patterns
OLD_VERSION=${OLD_VERSION:-v3.1.6}
# this command will retrieve the folder with the largest number in format v<number>
SOFTWARE_UPGRADE_NAME=${SOFTWARE_UPGRADE_NAME:-$(ls -d -- ./app/upgrades/v* | sort -Vr | head -n 1 | xargs basename)}
BUILDDIR=${1:-}
TESTNET_NVAL=${2:-}
TESTNET_CHAINID=${3:-}

# check if BUILDDIR is set
if [ -z "$BUILDDIR" ]; then
    echo "BUILDDIR is not set"
    exit 1
fi

# install old version of terrad

## check if _build/classic-${OLD_VERSION} exists
if [ ! -d "_build/core-${OLD_VERSION#v}" ]; then
    mkdir -p _build
    wget -c "https://github.com/classic-terra/core/archive/refs/tags/${OLD_VERSION}.zip" -O _build/${OLD_VERSION}.zip
    unzip _build/${OLD_VERSION}.zip -d _build
fi

## Reuse the cached binary only if its requested source version and contents
## match the metadata written after a successful build. Legacy/unverifiable
## cache entries are rebuilt rather than trusted.
OLD_BINARY="$BUILDDIR/old/terrad"
OLD_BINARY_STAMP="$BUILDDIR/old/terrad.version"
if ! old_terrad_cache_valid "$OLD_BINARY" "$OLD_BINARY_STAMP" "$OLD_VERSION"; then
    mkdir -p "$BUILDDIR/old"
    rm -f "$OLD_BINARY" "$OLD_BINARY_STAMP"
    OLD_SOURCE="_build/core-${OLD_VERSION#v}"
    OLD_VERSION_NUMBER=${OLD_VERSION#v}
    docker build --platform linux/amd64 --no-cache \
        --build-arg "source=./${OLD_SOURCE}/" \
        --build-arg "GIT_VERSION=$OLD_VERSION_NUMBER" \
        --tag classic-terra/terraclassic.terrad-binary.old \
        -f contrib/updates/Dockerfile.old .
    docker create --platform linux/amd64 --name old-temp classic-terra/terraclassic.terrad-binary.old:latest
    docker cp old-temp:/usr/local/bin/terrad "$OLD_BINARY"
    docker rm old-temp
    OLD_BINARY_SHA256=$(sha256sum "$OLD_BINARY" | awk '{print $1}')
    printf 'version=%s\nsha256=%s\n' "$OLD_VERSION" "$OLD_BINARY_SHA256" > "$OLD_BINARY_STAMP"
fi

# prepare cosmovisor config in TESTNET_NVAL nodes
if [ ! -f "$BUILDDIR/node0/terrad/config/genesis.json" ]; then docker run --rm \
    --user $(id -u):$(id -g) \
    -v "$BUILDDIR:/terrad:Z" \
    -v /etc/group:/etc/group:ro \
    -v /etc/passwd:/etc/passwd:ro \
    -v /etc/shadow:/etc/shadow:ro \
    --entrypoint /terrad/old/terrad \
    --platform linux/amd64 \
    classic-terra/terrad-upgrade-env testnet --v $TESTNET_NVAL --chain-id $TESTNET_CHAINID -o . --starting-ip-address 192.168.10.2 --keyring-backend=test --home=temp; \
fi

for (( i=0; i<$TESTNET_NVAL; i++ )); do
    CURRENT=$BUILDDIR/node$i/terrad

    # change gov params voting_period
    jq '.app_state.gov.voting_params.voting_period = "50s"' $CURRENT/config/genesis.json > $CURRENT/config/genesis.json.tmp && mv $CURRENT/config/genesis.json.tmp $CURRENT/config/genesis.json

    docker run --rm \
        --user $(id -u):$(id -g) \
        -v "$BUILDDIR:/terrad:Z" \
        -v /etc/group:/etc/group:ro \
        -v /etc/passwd:/etc/passwd:ro \
        -v /etc/shadow:/etc/shadow:ro \
        -e DAEMON_HOME=/terrad/node$i/terrad \
        -e DAEMON_NAME=terrad \
        -e DAEMON_RESTART_AFTER_UPGRADE=true \
        --entrypoint /terrad/cosmovisor \
        --platform linux/amd64 \
        classic-terra/terrad-upgrade-env init /terrad/old/terrad
    mkdir -p $CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin
    cp "$BUILDDIR/terrad" "$CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin"
    touch $CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/upgrade-info.json
done

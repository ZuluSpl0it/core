#!/bin/bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "$script_dir/old_terrad_cache.sh"
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' EXIT

binary="$test_dir/terrad"
stamp="$test_dir/terrad.version"
printf 'test binary v4.0.1\n' > "$binary"
binary_sha=$(sha256sum "$binary" | awk '{print $1}')
printf 'version=v4.0.1\nsha256=%s\n' "$binary_sha" > "$stamp"

old_terrad_cache_valid "$binary" "$stamp" v4.0.1
if old_terrad_cache_valid "$binary" "$stamp" v3.1.6; then
    echo 'cache unexpectedly accepted for a different OLD_VERSION' >&2
    exit 1
fi

printf 'modified binary\n' > "$binary"
if old_terrad_cache_valid "$binary" "$stamp" v4.0.1; then
    echo 'cache unexpectedly accepted after binary contents changed' >&2
    exit 1
fi

rm "$stamp"
if old_terrad_cache_valid "$binary" "$stamp" v4.0.1; then
    echo 'cache unexpectedly accepted without verifiable version metadata' >&2
    exit 1
fi

echo 'old terrad cache checks passed'

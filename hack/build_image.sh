#!/usr/bin/env bash

set -euo pipefail

target=$(mktemp)
truncate -s 21MiB "${target}"
parted -s -a optimal "${target}" -- \
    mklabel msdos mkpart primary ext4 1MiB 100%
echo "${target}"

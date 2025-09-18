#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
    echo "Illegal number of parameters" >&2
    exit 1
fi

PLATFORM=$1
ARCH=$2
SOPS_VERSION=$3

if [[ ${PLATFORM} == "windows" ]]; then
    wget -O "dist/sops.exe" "https://github.com/getsops/sops/releases/download/${SOPS_VERSION}/sops-${SOPS_VERSION}.${ARCH}.exe"
    chmod +x "dist/sops.exe"
else
    wget -O "dist/sops" "https://github.com/getsops/sops/releases/download/${SOPS_VERSION}/sops-${SOPS_VERSION}.${PLATFORM}.${ARCH}"
    chmod +x "dist/sops"
fi

#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

id() { echo "${MOCK_UID:-0}"; }
dpkg() { echo amd64; }
uname() { echo x86_64; }
sudo() { echo "sudo $*"; "$@"; }
apt-get() { echo "apt-get $*"; }
dnf() { echo "dnf $*"; }
install() { echo "install $*"; }
chmod() { echo "chmod $*"; }
curl() {
    if [[ ${MOCK_CURL_FAIL:-0} == 1 ]]; then
        echo "mock download failure" >&2
        return 1
    fi
    echo mock
}
tee() { cat >/dev/null; }
brew() {
    if [[ $1 == --prefix ]]; then
        echo "/mock/$2"
    else
        echo "brew $*"
    fi
}
export -f id dpkg uname sudo apt-get dnf install chmod curl tee brew

output=$(printf 'mock input\n' | sudo cat)
[[ $output == *"mock input"* ]]
if sudo false >/dev/null; then
    echo "The sudo mock must propagate command failures." >&2
    exit 1
fi

linux() {
    local release=$1
    bash <(sed 's@/etc/os-release@/dev/stdin@g' "$root/scripts/install-prerequisites-linux.sh") <<< "$release"
}

for distribution in ubuntu debian; do
    output=$(linux "ID=$distribution
VERSION_CODENAME=bookworm")
    [[ $output == *"apt-get install -y make python3 temurin-21-jdk nodejs=24.* docker-ce"* ]]
done

output=$(linux 'ID=fedora')
[[ $output == *"dnf install -y make python3 java-21-openjdk-devel nodejs-24.* docker-ce"* ]]

output=$(MOCK_UID=1000 linux "ID=ubuntu
VERSION_CODENAME=bookworm")
[[ $output == *"sudo apt-get install -y make python3 temurin-21-jdk nodejs=24.* docker-ce"* ]]

if linux 'ID=unsupported' >/dev/null 2>&1; then
    echo "Unsupported distributions must fail." >&2
    exit 1
fi
if MOCK_CURL_FAIL=1 linux 'ID=fedora' >/dev/null 2>&1; then
    echo "Failed repository downloads must fail the installer." >&2
    exit 1
fi

output=$(bash "$root/scripts/install-prerequisites-macos.sh")
[[ $output == *"brew install make openjdk@21 node@24 python"* ]]
[[ $output == *"brew install --cask docker-desktop"* ]]
[[ $output == *"/mock/node@24/bin:"* ]]

echo "Prerequisite installer tests passed."

#!/usr/bin/env bash
set -euo pipefail

if [[ ! -r /etc/os-release ]]; then
    echo "Cannot identify this Linux distribution: /etc/os-release is missing." >&2
    exit 1
fi
. /etc/os-release

if [[ $(id -u) -eq 0 ]]; then
    run_privileged() { "$@"; }
elif command -v sudo >/dev/null 2>&1; then
    run_privileged() { sudo "$@"; }
else
    echo "This installer requires root or sudo." >&2
    exit 1
fi

case "$ID" in
    ubuntu|debian)
        arch=$(dpkg --print-architecture)
        case "$arch" in
            amd64|arm64) ;;
            *) echo "NodeSource does not support architecture $arch in this installer." >&2; exit 1 ;;
        esac
        codename=${VERSION_CODENAME:?Missing VERSION_CODENAME in /etc/os-release}
        run_privileged apt-get update
        run_privileged apt-get install -y ca-certificates curl gnupg
        run_privileged install -d -m 0755 /etc/apt/keyrings
        curl -fsSL "https://download.docker.com/linux/$ID/gpg" | run_privileged tee /etc/apt/keyrings/mimir-docker.asc >/dev/null
        curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | run_privileged tee /etc/apt/keyrings/mimir-nodesource.asc >/dev/null
        curl -fsSL https://packages.adoptium.net/artifactory/api/gpg/key/public | run_privileged tee /etc/apt/keyrings/mimir-adoptium.asc >/dev/null
        run_privileged chmod a+r /etc/apt/keyrings/mimir-{docker,nodesource,adoptium}.asc
        printf 'deb [arch=%s signed-by=/etc/apt/keyrings/mimir-docker.asc] https://download.docker.com/linux/%s %s stable\n' "$arch" "$ID" "$codename" | run_privileged tee /etc/apt/sources.list.d/mimir-docker.list >/dev/null
        printf 'deb [arch=%s signed-by=/etc/apt/keyrings/mimir-nodesource.asc] https://deb.nodesource.com/node_24.x nodistro main\n' "$arch" | run_privileged tee /etc/apt/sources.list.d/mimir-nodesource.list >/dev/null
        printf 'deb [signed-by=/etc/apt/keyrings/mimir-adoptium.asc] https://packages.adoptium.net/artifactory/deb %s main\n' "$codename" | run_privileged tee /etc/apt/sources.list.d/mimir-adoptium.list >/dev/null
        run_privileged apt-get update
        run_privileged apt-get install -y make python3 temurin-21-jdk 'nodejs=24.*' \
            docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
        ;;
    fedora)
        arch=$(uname -m)
        case "$arch" in
            x86_64|aarch64) ;;
            *) echo "NodeSource does not support architecture $arch in this installer." >&2; exit 1 ;;
        esac
        run_privileged dnf install -y curl
        curl -fsSL https://download.docker.com/linux/fedora/docker-ce.repo | run_privileged tee /etc/yum.repos.d/mimir-docker.repo >/dev/null
        printf '%s\n' '[mimir-nodesource]' 'name=NodeSource Node.js 24' \
            "baseurl=https://rpm.nodesource.com/pub_24.x/nodistro/nodejs/$arch" \
            'enabled=1' 'gpgcheck=1' 'gpgkey=https://rpm.nodesource.com/gpgkey/ns-operations-public.key' \
            'module_hotfixes=1' | run_privileged tee /etc/yum.repos.d/mimir-nodesource.repo >/dev/null
        run_privileged dnf install -y make python3 java-21-openjdk-devel 'nodejs-24.*' \
            docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
        ;;
    *)
        echo "Unsupported Linux distribution: $ID. Supported: Ubuntu, Debian, Fedora." >&2
        exit 1
        ;;
esac

echo "Start Docker with: sudo systemctl enable --now docker"
echo "Configure Docker access for your user and set JAVA_HOME to the JDK 21 directory."

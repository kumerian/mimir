#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

case "$(uname -s)" in
    Linux)
        if [ ! -r /etc/os-release ]; then
            echo "ERROR: cannot identify Linux distribution; see the manual setup guide." >&2
            exit 1
        fi
        . /etc/os-release
        if [ "${ID:-}" != ubuntu ] || [ "${VERSION_ID:-}" != 24.04 ]; then
            echo "ERROR: automatic Linux setup requires Ubuntu 24.04; see the manual setup guide." >&2
            exit 1
        fi
        sudo apt-get update
        sudo apt-get install -y git make python3.12
        ;;
    Darwin)
        if ! command -v brew >/dev/null 2>&1; then
            echo "ERROR: Homebrew is required; install it first or see the manual setup guide." >&2
            exit 1
        fi
        for package in git make python@3.12; do
            if ! brew list --versions "$package" >/dev/null; then
                brew install "$package"
            fi
        done
        export PATH="$(brew --prefix make)/libexec/gnubin:$(brew --prefix python@3.12)/bin:$PATH"
        ;;
    *)
        echo "ERROR: unsupported OS; see the manual setup guide." >&2
        exit 1
        ;;
esac

git --version
if ! make --version | grep -q 'GNU Make'; then
    echo "ERROR: GNU Make is required." >&2
    exit 1
fi
python3.12 "$root/scripts/setup_dev.py"

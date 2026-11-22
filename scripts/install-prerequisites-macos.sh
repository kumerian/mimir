#!/usr/bin/env bash
set -euo pipefail

if ! command -v brew >/dev/null 2>&1; then
    echo "Homebrew is required; install it from https://brew.sh/ first." >&2
    exit 1
fi

brew install make openjdk@21 node@24 python
brew install --cask docker-desktop

echo "Add the following to your shell configuration, then open a new terminal:"
echo "export JAVA_HOME=\"$(brew --prefix openjdk@21)/libexec/openjdk.jdk/Contents/Home\""
echo "export PATH=\"$(brew --prefix node@24)/bin:\$JAVA_HOME/bin:\$PATH\""
echo "Start Docker Desktop and complete its first-run setup."

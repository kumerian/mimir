# Local development setup

The setup scripts install Git, Python 3.12, the Go toolchain pinned in
`go.mod`, and the Go module dependencies. They install GNU Make on macOS and
Ubuntu. On Windows, GNU Make must already be on PATH: no verified Windows Make
package is currently configured. By default, Make must have a valid
Authenticode signature; an unsigned installation requires an explicit opt-in.
No cloud account, service,
production data, or secret is required. The repository checks use only Python's
standard library. CI pins Python 3.12.3 in `.python-version`; package managers
may install a later 3.12 patch locally.

Automatic bootstrap is available for macOS with Homebrew and Ubuntu 24.04.
Windows bootstrap requires the GNU Make prerequisite above. These paths
target amd64 and arm64. Other Linux distributions can use the manual
prerequisites below. Internet access is required for package repositories, Go
downloads, and module dependencies. The scripts do not install optional
container tooling or the C compiler required by `make test-race`.

The race detector is an optional development check that instruments concurrent
Go code and reports unsynchronized shared-memory access. Install a C compiler
and ensure its executable is on PATH before using `make test-race`; Go can then
enable CGO for that command. Plain `make test` does not require a C compiler.
When the compiler is absent, `make test-race` exits with an explicit
prerequisite error instead of attempting race instrumentation.

## Automatic setup

Run from a checkout. Review the scripts before running them: package managers
may install system packages, request administrator privileges, and update
their package indexes. On Windows, install Windows Package Manager (`winget`)
first. On macOS, install Homebrew and its prerequisites first. Ubuntu requires
`sudo` and the standard Ubuntu package repositories.

Windows PowerShell:

```powershell
& .\scripts\setup-dev.ps1
```

If GNU Make is unsigned and you have independently verified and trust its
source, opt in explicitly:

```powershell
& .\scripts\setup-dev.ps1 -AllowUnsignedMake
```

This allows only unsigned Make, not an executable with an invalid signature.
It does not bypass Windows application-control policies.

macOS or Ubuntu:

```sh
sh scripts/setup-dev.sh
```

Windows uses winget for missing Git and Python 3.12 and verifies that the
existing GNU Make has a valid Authenticode signature (unless explicitly opted
in to unsigned Make) and runs. A valid
signature does not guarantee that Smart App Control will permit an executable.
macOS uses Homebrew for these packages; Ubuntu uses `apt-get`. The scripts
download the exact Go release named by the `toolchain` directive into
`~/.mimir-dev/go<version>` and verify its SHA-256 against the official Go
release listing before extracting it. The checksum and archive are both
obtained from Go's HTTPS distribution service; this is not independent
signature verification. Existing Go installations are not changed. The
scripts then run `go mod download`, using the normal Go module cache and proxy
configuration. After successful setup, Windows adds the installed Go `bin`
directory to the front of the persistent **user** PATH, preserving other
entries, and to the current setup PowerShell process. Commands launched from
that PowerShell can use the pinned toolchain immediately. Make also resolves
the installed toolchain named by `go.mod` directly, so it remains usable from
a process with an older PATH. macOS and Ubuntu do not change persistent PATH
settings.

On Windows, local state directories use a restricted ACL for the current user,
SYSTEM, and local Administrators. Existing files are rejected if their ACL
grants access to other principals.

## Configure and verify a development shell

On macOS and Ubuntu, add the installed Go `bin` directory to the **front** of
PATH in each new shell; replace the version below if `go.mod` changes. Set
`GOTOOLCHAIN=local` to keep commands on that installation. On macOS, also add
Homebrew's GNU Make and Python directories:

```sh
export PATH="$HOME/.mimir-dev/go1.27.4/bin:$PATH"
export GOTOOLCHAIN=local
# macOS only:
export PATH="$(brew --prefix make)/libexec/gnubin:$(brew --prefix python@3.12)/bin:$PATH"
go version
python3.12 --version
make --version
make test
make lint
make docs
make build
make install-smoke
```

Windows PowerShell: restart applications or sign out and back in to pick up
the updated user PATH. System PATH entries can precede the user PATH, and an
already-open shell retains its previous PATH. To ensure the pinned Go takes
precedence in the current shell:

```powershell
$env:Path = "$HOME\.mimir-dev\go1.27.4\bin;$env:Path"
$env:GOTOOLCHAIN = 'local'
go version
py -3.12 --version
make --version
make test
make lint
make docs
make build
make install-smoke
```

`make install-smoke` uses the same repository-pinned Go executable as the other
Make tasks. It builds into a temporary clean install, verifies the CLI's health
and validation contracts, and runs a synthetic local ingestion. Formatting
checks likewise use the pinned toolchain's `gofmt`.

Open a new terminal after package installation if Git, Make, or the Python
launcher is not on PATH. If the Windows installer reports a GNU Make directory
to add to PATH, add that directory in each development shell as well. Setup
checks that the discovered Make executable actually runs; if an existing
executable is blocked by application-control policy, contact the local
administrator. The smoke test uses a temporary install and synthetic local data; see the
[local reference guide](../user/local-reference-ingestion.md). The automatic
setup itself does not run the checks above.

## Manual prerequisites and recovery

For other Linux distributions, install Git, GNU Make, and Python 3.12 using
your distribution's trusted packages, then run `python3.12 scripts/setup_dev.py`
from the checkout. The Go installer also works directly on macOS and Windows
(`py -3.12 scripts\setup_dev.py`) after these prerequisites are installed.
Unsupported OS or architecture combinations fail explicitly.

If a download, checksum, extraction, or module download fails, correct network,
proxy, permissions, or disk-space issues and rerun. An incomplete existing
`~/.mimir-dev/go<version>` directory is reported rather than overwritten;
remove that specific directory only after confirming it contains no data you
need, then retry. If an installed tool is blocked by an application-control
policy, contact the local administrator rather than bypassing the policy.

To undo the user-local Go installation, remove the specific
`~/.mimir-dev/go<version>` directory and remove its entry from PATH, including
the persistent Windows user PATH if setup added it.
Uninstall Git, Make, or Python through the package manager only if they are no
longer needed by other projects. The Go module cache is shared with other Go
projects and is not removed by these scripts. Updating `go.mod` to a new
toolchain and rerunning installs it alongside the old one; switch PATH back to
the old version to roll back. No Mimir runtime state is migrated or removed.

param(
    [switch]$AllowUnsignedMake
)

$ErrorActionPreference = 'Stop'

if ($env:OS -ne 'Windows_NT') {
    throw 'This script is for Windows. See the local development setup guide.'
}
if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    throw 'Windows Package Manager (winget) is required. See the local development setup guide.'
}

function Install-PackageIfMissing([string]$Command, [string]$Package) {
    if (-not (Get-Command $Command -ErrorAction SilentlyContinue)) {
        & winget install --id $Package --exact --silent --accept-package-agreements --accept-source-agreements
        if ($LASTEXITCODE -ne 0) {
            throw "winget failed to install $Package (exit $LASTEXITCODE)."
        }
        $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' +
            [Environment]::GetEnvironmentVariable('Path', 'User') + ';' + $env:Path
    }
}

Install-PackageIfMissing 'git.exe' 'Git.Git'

$gnuMakeBin = Join-Path ${env:ProgramFiles(x86)} 'GnuWin32\bin'
function Find-Make {
    $command = Get-Command make.exe -ErrorAction SilentlyContinue
    if ($command) { return $command.Source }
    $installedMake = Join-Path $gnuMakeBin 'make.exe'
    if (Test-Path $installedMake) {
        $env:Path = "$gnuMakeBin;$env:Path"
        Write-Host "Add $gnuMakeBin to PATH in new development shells."
        return $installedMake
    }
}

$make = Find-Make
if (-not $make) {
    throw 'A signed GNU Make is required on PATH. No verified Windows Make package is configured; see the local development setup guide.'
}
$signature = Get-AuthenticodeSignature -FilePath $make
if ($signature.Status -eq 'NotSigned' -and $AllowUnsignedMake) {
    Write-Warning "GNU Make at $make is unsigned; proceeding because -AllowUnsignedMake was specified. Verify that you trust its source."
} elseif ($signature.Status -ne 'Valid') {
    throw "GNU Make at $make does not have a valid Authenticode signature (status: $($signature.Status)). For an unsigned executable from a trusted source, see -AllowUnsignedMake in the local development setup guide."
}
try {
    $makeVersion = & $make --version 2>&1
} catch {
    throw "GNU Make was found but could not be run at ${make}: $($_.Exception.Message). Check application-control policy and PATH."
}
if ($LASTEXITCODE -ne 0 -or ($makeVersion -join "`n") -notmatch 'GNU Make') {
    throw "The executable at $make is not a working GNU Make. Check its installation and PATH."
}

$pythonAvailable = $false
if (Get-Command py.exe -ErrorAction SilentlyContinue) {
    & py -3.12 -c 'import sys; sys.exit(sys.version_info[:2] != (3, 12))'
    $pythonAvailable = $LASTEXITCODE -eq 0
}
if (-not $pythonAvailable) {
    & winget install --id Python.Python.3.12 --exact --silent --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) {
        throw "winget failed to install Python 3.12 (exit $LASTEXITCODE)."
    }
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' +
        [Environment]::GetEnvironmentVariable('Path', 'User') + ';' + $env:Path
}

& git --version
if ($LASTEXITCODE -ne 0) { throw 'Git is not executable.' }
& py -3.12 (Join-Path $PSScriptRoot 'setup_dev.py')
if ($LASTEXITCODE -ne 0) {
    throw "Development setup failed (exit $LASTEXITCODE)."
}

$goMod = Get-Content (Join-Path $PSScriptRoot '..\go.mod') -Raw
if ($goMod -notmatch '(?m)^toolchain go(\d+\.\d+\.\d+)\s*$') {
    throw 'go.mod does not contain a pinned Go toolchain; user PATH was not changed.'
}
if (-not $env:USERPROFILE) {
    throw 'USERPROFILE is not set; user PATH was not changed.'
}
$goBin = Join-Path $env:USERPROFILE ".mimir-dev\go$($Matches[1])\bin"
$userEnvironment = Get-Item 'HKCU:\Environment'
$userPath = $userEnvironment.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
if ($null -eq $userPath) {
    $pathKind = [Microsoft.Win32.RegistryValueKind]::ExpandString
    $remainingPaths = @()
} else {
    $pathKind = $userEnvironment.GetValueKind('Path')
    if ($pathKind -notin @([Microsoft.Win32.RegistryValueKind]::String, [Microsoft.Win32.RegistryValueKind]::ExpandString)) {
        throw "User PATH has unsupported registry type $pathKind; it was not changed."
    }
    $remainingPaths = @($userPath -split ';' | Where-Object { $_.TrimEnd('\') -ine $goBin.TrimEnd('\') })
}
$newUserPath = (@($goBin) + $remainingPaths) -join ';'
if ($newUserPath.Length -ge 32767) {
    throw 'The updated user PATH would exceed the Windows environment variable limit; it was not changed.'
}
if ($newUserPath -cne $userPath) {
    Set-ItemProperty -Path 'HKCU:\Environment' -Name 'Path' -Value $newUserPath -Type $pathKind
    Write-Host "Added $goBin to the user PATH."
} else {
    Write-Host "$goBin is already at the front of the user PATH."
}
Write-Host 'Restart applications or sign out and back in to pick up the user PATH. System PATH entries can take precedence.'

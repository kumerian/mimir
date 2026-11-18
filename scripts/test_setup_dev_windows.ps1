$ErrorActionPreference = 'Stop'
$setup = Join-Path $PSScriptRoot 'setup-dev.ps1'

class MockEnvironmentKey {
    [object] GetValue([string]$name, [object]$default, [Microsoft.Win32.RegistryValueOptions]$options) {
        return $global:mockUserPath
    }

    [Microsoft.Win32.RegistryValueKind] GetValueKind([string]$name) {
        return $global:mockUserPathKind
    }
}

$global:mockUserPath = $null
$global:mockUserPathKind = [Microsoft.Win32.RegistryValueKind]::ExpandString
$global:mockPathWrites = 0
function global:Get-Item {
    param([string]$Path)
    if ($Path -eq 'HKCU:\Environment') { return [MockEnvironmentKey]::new() }
    Microsoft.PowerShell.Management\Get-Item $Path
}
function global:Set-ItemProperty {
    param([string]$Path, [string]$Name, [object]$Value, [object]$Type)
    if ($Path -ne 'HKCU:\Environment' -or $Name -ne 'Path') {
        throw "Unexpected registry write: $Path $Name"
    }
    $global:mockUserPath = $Value
    $global:mockUserPathKind = $Type
    $global:mockPathWrites++
}

function global:Test-Path {
    param([string]$Path)
    if ($Path -eq (Join-Path ${env:ProgramFiles(x86)} 'GnuWin32\bin\make.exe')) { return $false }
    Microsoft.PowerShell.Management\Test-Path $Path
}

function Assert-Contains([string]$Actual, [string]$Expected) {
    if (-not $Actual.Contains($Expected)) {
        throw "Expected '$Expected' in '$Actual'."
    }
}

function Test-MissingMake {
    $global:mockInstalled = $false
    function global:Get-Command {
        param([string]$Name)
        if ($Name -eq 'make.exe') { return }
        Microsoft.PowerShell.Core\Get-Command $Name
    }
    function global:winget {
        $global:mockInstalled = $true
        $global:LASTEXITCODE = 0
    }
    try {
        & $setup
        throw 'Setup unexpectedly succeeded without Make.'
    } catch {
        if ($global:mockInstalled) { throw 'Setup attempted to install an unverified Make package.' }
        Assert-Contains $_.Exception.Message 'signed GNU Make is required'
    }
}

function Test-UnusableMake {
    function global:Get-Command {
        param([string]$Name)
        if ($Name -eq 'make.exe') {
            return [pscustomobject]@{ Source = 'make' }
        }
        Microsoft.PowerShell.Core\Get-Command $Name
    }
    function global:Get-AuthenticodeSignature { [pscustomobject]@{ Status = 'Valid' } }
    function global:make { throw 'simulated launch failure' }
    try {
        & $setup
        throw 'Setup unexpectedly succeeded with an unusable Make.'
    } catch {
        Assert-Contains $_.Exception.Message 'GNU Make was found but could not be run'
    }
}

function Test-ExistingMakeOutsidePath {
    $global:mockInstalled = $false
    $originalPath = $env:Path
    function global:Get-Command {
        param([string]$Name)
        if ($Name -eq 'make.exe') { return }
        Microsoft.PowerShell.Core\Get-Command $Name
    }
    function global:Test-Path {
        param([string]$Path)
        if ($Path -eq (Join-Path ${env:ProgramFiles(x86)} 'GnuWin32\bin\make.exe')) { return $true }
        Microsoft.PowerShell.Management\Test-Path $Path
    }
    function global:winget {
        $global:mockInstalled = $true
        $global:LASTEXITCODE = 0
    }
    function global:Get-AuthenticodeSignature { [pscustomobject]@{ Status = 'Valid' } }
    try {
        & $setup
        throw 'Setup unexpectedly ran a nonexistent test executable.'
    } catch {
        Assert-Contains $_.Exception.Message 'GNU Make was found but could not be run'
        if ($global:mockInstalled) { throw 'Setup tried to reinstall an existing Make.' }
    } finally {
        $env:Path = $originalPath
        function global:Test-Path {
            param([string]$Path)
            if ($Path -eq (Join-Path ${env:ProgramFiles(x86)} 'GnuWin32\bin\make.exe')) { return $false }
            Microsoft.PowerShell.Management\Test-Path $Path
        }
    }
}

function Test-UnsignedMake {
    $global:makeExecuted = $false
    function global:Get-Command {
        param([string]$Name)
        if ($Name -eq 'make.exe') {
            return [pscustomobject]@{ Source = 'make' }
        }
        Microsoft.PowerShell.Core\Get-Command $Name
    }
    function global:Get-AuthenticodeSignature { [pscustomobject]@{ Status = 'NotSigned' } }
    function global:make {
        $global:makeExecuted = $true
    }
    try {
        & $setup
        throw 'Setup unexpectedly accepted unsigned Make.'
    } catch {
        Assert-Contains $_.Exception.Message 'valid Authenticode signature'
        if ($global:makeExecuted) { throw 'Setup executed unsigned Make.' }
    }
}

function Test-UnsignedMakeOptIn {
    $global:makeExecuted = $false
    function global:Get-Command {
        param([string]$Name)
        if ($Name -eq 'make.exe') { return [pscustomobject]@{ Source = 'make' } }
        Microsoft.PowerShell.Core\Get-Command $Name
    }
    function global:Get-AuthenticodeSignature { [pscustomobject]@{ Status = 'NotSigned' } }
    function global:make {
        $global:makeExecuted = $true
        $global:LASTEXITCODE = 0
        'GNU Make 4.4.1'
    }
    function global:py { $global:LASTEXITCODE = 0 }

    & $setup -AllowUnsignedMake
    if (-not $global:makeExecuted) { throw 'Setup did not verify opted-in unsigned Make.' }
}

function Test-InvalidSignatureWithOptIn {
    $global:makeExecuted = $false
    function global:Get-Command {
        param([string]$Name)
        if ($Name -eq 'make.exe') { return [pscustomobject]@{ Source = 'make' } }
        Microsoft.PowerShell.Core\Get-Command $Name
    }
    function global:Get-AuthenticodeSignature { [pscustomobject]@{ Status = 'HashMismatch' } }
    function global:make { $global:makeExecuted = $true }
    try {
        & $setup -AllowUnsignedMake
        throw 'Setup unexpectedly accepted Make with an invalid signature.'
    } catch {
        Assert-Contains $_.Exception.Message 'valid Authenticode signature'
        if ($global:makeExecuted) { throw 'Setup executed Make with an invalid signature.' }
    }
}

function Test-SignedMake {
    $global:makeExecuted = $false
    function global:Get-Command {
        param([string]$Name)
        if ($Name -eq 'make.exe') { return [pscustomobject]@{ Source = 'make' } }
        Microsoft.PowerShell.Core\Get-Command $Name
    }
    function global:Get-AuthenticodeSignature { [pscustomobject]@{ Status = 'Valid' } }
    function global:make {
        $global:makeExecuted = $true
        $global:LASTEXITCODE = 0
        'GNU Make 4.4'
    }
    function global:py { $global:LASTEXITCODE = 0 }

    & $setup
    if (-not $global:makeExecuted) { throw 'Setup did not verify signed Make.' }
}

function Test-PersistentGoPath {
    $goMod = Get-Content (Join-Path $PSScriptRoot '..\go.mod') -Raw
    if ($goMod -notmatch '(?m)^toolchain go(\d+\.\d+\.\d+)\s*$') { throw 'No pinned toolchain.' }
    $goBin = Join-Path $env:USERPROFILE ".mimir-dev\go$($Matches[1])\bin"
    $global:mockUserPath = "%LOCALAPPDATA%\Tools;$($goBin.ToUpper())\;C:\other"
    $global:mockUserPathKind = [Microsoft.Win32.RegistryValueKind]::ExpandString
    $global:mockPathWrites = 0

    & $setup
    if ($global:mockUserPath -cne "$goBin;%LOCALAPPDATA%\Tools;C:\other") {
        throw "Incorrect user PATH: $global:mockUserPath"
    }
    if ($global:mockUserPathKind -ne [Microsoft.Win32.RegistryValueKind]::ExpandString) {
        throw 'Setup changed the user PATH registry type.'
    }
    & $setup
    if ($global:mockPathWrites -ne 1) { throw 'Setup wrote an unchanged user PATH.' }

    $global:mockUserPath = $null
    $global:mockPathWrites = 0
    & $setup
    if ($global:mockUserPath -cne $goBin) { throw 'Setup did not create an absent user PATH.' }
}

function Test-NoPathChangeOnSetupFailure {
    $global:mockUserPath = 'C:\existing'
    $global:mockPathWrites = 0
    function global:py {
        if ($args[1] -eq '-c') { $global:LASTEXITCODE = 0 }
        else { $global:LASTEXITCODE = 1 }
    }
    try {
        & $setup
        throw 'Setup unexpectedly succeeded after installer failure.'
    } catch {
        Assert-Contains $_.Exception.Message 'Development setup failed'
        if ($global:mockPathWrites -ne 0) { throw 'Setup changed PATH after installer failure.' }
    }
}

Test-MissingMake
Test-UnusableMake
Test-ExistingMakeOutsidePath
Test-UnsignedMake
Test-UnsignedMakeOptIn
Test-InvalidSignatureWithOptIn
Test-SignedMake
Test-PersistentGoPath
Test-NoPathChangeOnSetupFailure
Write-Host 'PASS: Windows setup handles missing, unsigned, invalidly signed, and unusable Make'

$ErrorActionPreference = 'Stop'
$global:PrerequisiteTestCalls = @()
$global:PrerequisiteTestFailAt = 0

function winget.exe {
    $global:PrerequisiteTestCalls += ,@($args)
    if ($global:PrerequisiteTestCalls.Count -eq $global:PrerequisiteTestFailAt) {
        $global:LASTEXITCODE = 1
    } else {
        $global:LASTEXITCODE = 0
    }
}

$installer = Join-Path $PSScriptRoot '..\scripts\install-prerequisites-windows.ps1'
& $installer
if ($global:PrerequisiteTestCalls.Count -ne 5 -or $global:PrerequisiteTestCalls[0][2] -ne 'ezwinports.make' -or
    $global:PrerequisiteTestCalls[4][2] -ne 'Docker.DockerDesktop') {
    throw 'The Windows installer did not invoke the expected packages.'
}

$global:PrerequisiteTestCalls = @()
$global:PrerequisiteTestFailAt = 3
$failed = $false
try {
    & $installer
} catch {
    $failed = $true
}
if (-not $failed -or $global:PrerequisiteTestCalls.Count -ne 3) {
    throw 'The Windows installer did not stop on a package installation failure.'
}
$global:LASTEXITCODE = 0
Write-Host 'Windows prerequisite installer tests passed.'

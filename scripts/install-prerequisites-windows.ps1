$ErrorActionPreference = 'Stop'

if (-not (Get-Command winget.exe -ErrorAction SilentlyContinue)) {
    throw 'Windows Package Manager (winget) is required. Install it via App Installer first.'
}

$packages = @(
    'ezwinports.make',
    'EclipseAdoptium.Temurin.21.JDK',
    'OpenJS.NodeJS.24',
    'Python.Python.3.13',
    'Docker.DockerDesktop'
)

foreach ($package in $packages) {
    Write-Host "Installing $package..."
    & winget.exe install --id $package --exact --source winget --no-upgrade `
        --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) {
        throw "winget failed to install $package (exit code $LASTEXITCODE)."
    }
}

Write-Host 'Open a new terminal, set JAVA_HOME to the JDK 21 directory, and start Docker Desktop.'

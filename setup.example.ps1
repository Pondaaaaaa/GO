# Copy this file to setup.ps1 in the repository root after cloning.
# The copy is ignored by Git and can be adapted to a local Go installation.
$ErrorActionPreference = 'Stop'

$go = Get-Command go.exe -ErrorAction SilentlyContinue
if (-not $go) {
    $localBin = Join-Path $PSScriptRoot '..\.local\go\bin'
    if (Test-Path -LiteralPath (Join-Path $localBin 'go.exe')) {
        $env:Path = "$localBin;$env:Path"
        $go = Get-Command go.exe -ErrorAction SilentlyContinue
    }
}

if (-not $go) {
    throw 'Go не найден. Установите Go и откройте новое окно PowerShell.'
}

# Practice 12 uses GOPATH when installing the Swagger generator.
if (-not $env:GOPATH) {
    $env:GOPATH = (& go env GOPATH).Trim()
}

Write-Host "Go готов к работе: $(& go version)"

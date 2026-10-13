$ErrorActionPreference = 'Stop'
& (Join-Path $PSScriptRoot '..\..\setup.ps1')
$toolDir = Join-Path $env:GOPATH 'bin\pz12-swag-v1.16.6'
$swag = Join-Path $toolDir 'swag.exe'
$installed = $false
if (Test-Path -LiteralPath $swag) {
    $metadata = & go version -m $swag
    $installed = $LASTEXITCODE -eq 0 -and (($metadata -join "`n") -match 'mod\s+github.com/swaggo/swag\s+v1\.16\.6\s')
}
if (-not $installed) {
    $oldGoBin = $env:GOBIN
    try {
        $env:GOBIN = $toolDir
        go install github.com/swaggo/swag/cmd/swag@v1.16.6
        if ($LASTEXITCODE -ne 0) { throw 'Installing swag v1.16.6 failed' }
    }
    finally { $env:GOBIN = $oldGoBin }
}
Push-Location $PSScriptRoot
try {
    & $swag init --dir ./cmd/api,./internal/core,./internal/http --generalInfo main.go --parseInternal --output ./docs
    if ($LASTEXITCODE -ne 0) { throw 'Swagger generation failed' }
}
finally { Pop-Location }

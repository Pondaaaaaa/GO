param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('run', 'build', 'test')]
    [string]$Command
)

$moduleRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $moduleRoot
try {
    switch ($Command) {
        'run' { go run ./cmd/server }
        'build' {
            New-Item -ItemType Directory -Force .\bin | Out-Null
            go build -o .\bin\server.exe .\cmd\server
        }
        'test' { go test ./... }
    }
    if ($LASTEXITCODE -ne 0) {
        throw "go $Command failed with exit code $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}

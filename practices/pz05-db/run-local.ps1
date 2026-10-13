$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$psqlPath = 'D:\programDev\bin\psql.exe'
$previousDatabaseURL = $env:DATABASE_URL
$previousPGPassword = $env:PGPASSWORD
if (-not (Test-Path -LiteralPath $psqlPath)) {
    $psqlCommand = Get-Command psql.exe -ErrorAction SilentlyContinue
    if ($null -eq $psqlCommand) {
        throw 'psql.exe not found. Install PostgreSQL Command Line Tools or set its path in run-local.ps1.'
    }
    $psqlPath = $psqlCommand.Source
}

& (Join-Path $repoRoot 'setup.ps1')
if ($LASTEXITCODE -and $LASTEXITCODE -ne 0) { throw 'setup.ps1 failed.' }

$user = $env:PGUSER
if ([string]::IsNullOrWhiteSpace($user)) { $user = 'postgres' }
$securePassword = Read-Host 'PostgreSQL password (input is hidden)' -AsSecureString
$passwordPtr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($securePassword)
$password = $null
try {
    $password = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($passwordPtr)
    $env:PGPASSWORD = $password

    Push-Location $PSScriptRoot
    try {
        # Create the database only when absent; never drop databases or rows.
        @("SELECT 'CREATE DATABASE todo' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'todo');", '\gexec') -join "`n" |
            & $psqlPath -X -w -h 127.0.0.1 -p 5432 -U $user -d postgres -v ON_ERROR_STOP=1
        if ($LASTEXITCODE -ne 0) { throw 'Could not check or create database todo.' }

        & $psqlPath -X -w -h 127.0.0.1 -p 5432 -U $user -d todo -v ON_ERROR_STOP=1 -c 'CREATE TABLE IF NOT EXISTS tasks (id SERIAL PRIMARY KEY, title TEXT NOT NULL, done BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());'
        if ($LASTEXITCODE -ne 0) { throw 'Could not check or create table tasks.' }

        $encodedUser = [Uri]::EscapeDataString($user)
        $encodedPassword = [Uri]::EscapeDataString($password)
        $env:DATABASE_URL = "postgres://${encodedUser}:${encodedPassword}@127.0.0.1:5432/todo?sslmode=disable"
        go run .
        if ($LASTEXITCODE -ne 0) { throw 'go run failed.' }

        Write-Host "`nChecking rows in todo.tasks with psql:"
        & $psqlPath -X -w -h 127.0.0.1 -p 5432 -U $user -d todo -v ON_ERROR_STOP=1 -c 'SELECT * FROM tasks ORDER BY id;'
        if ($LASTEXITCODE -ne 0) { throw 'The verification SELECT from tasks failed.' }
    }
    finally {
        if ($null -eq $previousDatabaseURL) { Remove-Item Env:DATABASE_URL -ErrorAction SilentlyContinue }
        else { $env:DATABASE_URL = $previousDatabaseURL }
        Pop-Location
    }
}
finally {
    if ($null -eq $previousPGPassword) { Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue }
    else { $env:PGPASSWORD = $previousPGPassword }
    $encodedPassword = $null
    $encodedUser = $null
    $password = $null
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($passwordPtr)
    $securePassword.Dispose()
}

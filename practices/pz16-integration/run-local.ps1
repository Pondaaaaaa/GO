$ErrorActionPreference = 'Stop'
$module = $PSScriptRoot
$originalLocation = Get-Location
$root = Split-Path (Split-Path $module -Parent) -Parent
$logs = Join-Path $root 'reports\logs\pz16'
$project = 'pz16-' + [Guid]::NewGuid().ToString('N')
$names = @('PZ16_PROJECT', 'PZ16_USER', 'PZ16_PASSWORD', 'PZ16_DATABASE', 'DB_DSN')
$previous = @{}
foreach ($name in $names) { $previous[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
$started = $false
$password = [Guid]::NewGuid().ToString('N') + [Guid]::NewGuid().ToString('N')
$dsn = ''
$compose = @('compose', '--project-name', $project, '--project-directory', $module, '-f', (Join-Path $module 'compose.yaml'))
New-Item -ItemType Directory -Force -Path $logs | Out-Null

function Invoke-Checked {
    param([string]$Program, [string[]]$Arguments, [string]$LogName)
    $savedPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $raw = (& $Program @Arguments 2>&1 | ForEach-Object { $_.ToString() } | Out-String)
        $code = $LASTEXITCODE
    } finally { $ErrorActionPreference = $savedPreference }
    $safe = $raw.Replace($password, '<redacted>')
    if ($dsn) { $safe = $safe.Replace($dsn, '<redacted>') }
    if ($LogName) { [IO.File]::WriteAllText((Join-Path $logs $LogName), $safe, [Text.UTF8Encoding]::new($false)) }
    if ($safe.Trim()) { Write-Host $safe.TrimEnd() }
    if ($code -ne 0) { throw "$Program failed (exit $code); see sanitized output above." }
    return $safe
}

try {
    & (Join-Path $root 'setup.ps1')
    Set-Location $module
    $versions = "$(go version)`nPowerShell $($PSVersionTable.PSVersion)`n"
    $versions += Invoke-Checked 'docker' @('version', '--format', 'Client {{.Client.Version}}; Engine {{.Server.Version}}') 'docker-version.txt'
    $versions += Invoke-Checked 'docker' @('compose', 'version') ''
    [IO.File]::WriteAllText((Join-Path $logs 'versions.txt'), $versions, [Text.UTF8Encoding]::new($false))
    $env:PZ16_PROJECT = $project
    $env:PZ16_USER = 'pz16_user'
    $env:PZ16_PASSWORD = $password
    $env:PZ16_DATABASE = 'notes_test'
    $started = $true # cleanup also runs after partially successful compose up
    $null = Invoke-Checked 'docker' ($compose + @('up', '-d', '--wait', '--wait-timeout', '120')) 'compose-up.txt'
    $null = Invoke-Checked 'docker' ($compose + @('ps')) 'compose-ps.txt'
    $binding = (Invoke-Checked 'docker' ($compose + @('port', 'pg', '5432')) '').Trim()
    if ($binding -notmatch '^127\.0\.0\.1:(\d+)$') { throw 'Expected an ephemeral loopback PostgreSQL port.' }
    $port = $Matches[1]
    $dsn = "postgres://pz16_user:$password@127.0.0.1:$port/notes_test?sslmode=disable&connect_timeout=5"
    $env:DB_DSN = $dsn
    $null = Invoke-Checked 'docker' ($compose + @('exec', '-T', 'pg', 'psql', '-U', 'pz16_user', '-d', 'notes_test', '-Atc', 'SHOW server_version')) 'postgres-version.txt'
    $null = Invoke-Checked 'go' @('test', '-count=1', '-timeout=120s', './...') 'test.txt'
    $verbose = Invoke-Checked 'go' @('test', '-count=1', '-timeout=120s', '-v', './integration/...') 'verbose.txt'
    if ($verbose -match '--- SKIP:' -or $verbose -notmatch '--- PASS: TestCreateAndGetNote' -or $verbose -notmatch '--- PASS: TestDeleteNote') {
        throw 'Required integration tests did not execute.'
    }
    $null = Invoke-Checked 'go' @('test', '-count=2', '-shuffle=on', '-timeout=120s', './integration/...') 'repeat.txt'
    $checks = ''
    foreach ($arguments in @(@('fmt','./...'), @('vet','./...'), @('build','./...'), @('mod','verify'))) {
        $checks += "go $($arguments -join ' ')`n"
        $checks += Invoke-Checked 'go' $arguments ''
        $checks += "exit=0`n"
    }
    [IO.File]::WriteAllText((Join-Path $logs 'checks.txt'), $checks, [Text.UTF8Encoding]::new($false))
    if (Test-Path -LiteralPath (Join-Path $logs 'last-error.txt')) { Remove-Item -LiteralPath (Join-Path $logs 'last-error.txt') }
    Write-Host 'PZ16 passed: real PostgreSQL, all integration tests executed without skips.'
} catch {
    $message = 'PZ16 failed: ' + $_.Exception.Message.Replace($password, '<redacted>')
    if ($dsn) { $message = $message.Replace($dsn, '<redacted>') }
    [IO.File]::WriteAllText((Join-Path $logs 'last-error.txt'), $message, [Text.UTF8Encoding]::new($false))
    throw $message
} finally {
    try {
        if ($started) {
            $null = Invoke-Checked 'docker' ($compose + @('down', '--volumes', '--remove-orphans', '--timeout', '10')) 'cleanup.txt'
            $remaining = (Invoke-Checked 'docker' @('ps', '-aq', '--filter', "label=com.docker.compose.project=$project") '').Trim()
            $volumes = (Invoke-Checked 'docker' @('volume', 'ls', '-q', '--filter', "label=com.docker.compose.project=$project") '').Trim()
            if ($remaining -or $volumes) { throw 'Owned Docker resources remain after cleanup.' }
            Add-Content -LiteralPath (Join-Path $logs 'cleanup.txt') -Value 'Verified: owned containers and volumes absent.'
        }
    } finally {
        foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name, $previous[$name], 'Process') }
        Set-Location $originalLocation
    }
}

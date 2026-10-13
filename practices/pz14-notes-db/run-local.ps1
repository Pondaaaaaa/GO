param([string]$OutputDirectory = '')
$ErrorActionPreference = 'Stop'
$moduleRoot = $PSScriptRoot
$repoRoot = (Resolve-Path (Join-Path $moduleRoot '..\..')).Path
& (Join-Path $repoRoot 'setup.ps1')
$psql = Get-Command psql.exe -ErrorAction SilentlyContinue
$psqlPath = if ($psql) { $psql.Source } else { 'D:\programDev\bin\psql.exe' }
if (-not (Test-Path -LiteralPath $psqlPath)) { throw 'Install PostgreSQL command line tools; psql.exe was not found.' }
$run = [guid]::NewGuid().ToString('N')
$tempDir = Join-Path $env:TEMP ('pz14-' + $run)
$outputRoot = if ($OutputDirectory) { [IO.Path]::GetFullPath($OutputDirectory) } else { Join-Path $repoRoot ("..\.local\pz14-results\$run") }
$oldPG = @{}
foreach ($key in @('PGPASSWORD','PGHOST','PGPORT','PGDATABASE','PGUSER','PGSSLMODE','PGCONNECT_TIMEOUT')) { $oldPG[$key] = [Environment]::GetEnvironmentVariable($key) }
$processes = [Collections.Generic.List[System.Diagnostics.Process]]::new()
$passwordPtr = [IntPtr]::Zero
$secure = $null
function SaveLines([string]$Name, [object[]]$Lines) { $Lines | Out-File -LiteralPath (Join-Path $outputRoot $Name) -Encoding utf8; $Lines | ForEach-Object { Write-Host $_ } }
function Lab([string[]]$Arguments, [string]$Name) {
    $lines = & (Join-Path $tempDir 'lab.exe') @Arguments
    if ($LASTEXITCODE -ne 0) { throw 'Lab check failed; connection data omitted.' }
    SaveLines $Name @($lines)
}
function HTTP([string]$Method,[string]$Path,[string]$Data,[int]$Expected) {
    $body = Join-Path $tempDir 'response.json'
    $headers = Join-Path $tempDir 'headers.txt'
    $curlArgs = @('--noproxy','*','-sS','--max-time','15','-X',$Method,'-o',$body,'-D',$headers,'-w','%{http_code}')
    if ($Data) { $file = Join-Path $tempDir 'request.json'; [IO.File]::WriteAllText($file,$Data,[Text.UTF8Encoding]::new($false)); $curlArgs += @('-H','Content-Type: application/json','--data-binary',"@$file") }
    $status = & curl.exe @curlArgs "$base$Path"
    if ($LASTEXITCODE -ne 0 -or [int]$status -ne $Expected) { throw "HTTP $Method $Path failed (status $status)" }
    $raw = [IO.File]::ReadAllText($body,[Text.Encoding]::UTF8)
    if ($Expected -eq 204 -and $raw.Length -ne 0) { throw 'DELETE must have zero-byte body' }
    if ($Expected -ne 204 -and -not ([IO.File]::ReadAllText($headers) -match '(?im)^Content-Type: application/json')) { throw 'JSON Content-Type missing' }
    $line = "$Method $Path -> $status $($raw.Trim())"
    Write-Host $line
    Add-Content -LiteralPath (Join-Path $outputRoot 'http.txt') -Value $line -Encoding utf8
    return $raw
}
try {
    New-Item -ItemType Directory -Path $tempDir,$outputRoot -Force | Out-Null
    $env:PGHOST = '127.0.0.1'; $env:PGPORT = '5432'; $env:PGDATABASE = 'postgres'; $env:PGSSLMODE = 'disable'; $env:PGCONNECT_TIMEOUT = '5'
    if (-not $env:PGUSER) { $env:PGUSER = 'postgres' }
    Push-Location $moduleRoot
    try {
        go build -o (Join-Path $tempDir 'api.exe') ./cmd/api
        if ($LASTEXITCODE -ne 0) { throw 'API build failed' }
        go build -o (Join-Path $tempDir 'lab.exe') ./cmd/lab
        if ($LASTEXITCODE -ne 0) { throw 'Lab build failed' }
    } finally { Pop-Location }
    $secure = Read-Host 'PostgreSQL password (input is hidden)' -AsSecureString
    $passwordPtr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
    $env:PGPASSWORD = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($passwordPtr)
    $exists = & $psqlPath -X -w -At -v ON_ERROR_STOP=1 -c "SELECT 1 FROM pg_database WHERE datname='pz14_notes';"
    if ($LASTEXITCODE -ne 0) { throw 'PostgreSQL authentication failed; no password was saved.' }
    if ($exists -notcontains '1') { & $psqlPath -X -w -v ON_ERROR_STOP=1 -c 'CREATE DATABASE pz14_notes;'; if ($LASTEXITCODE -ne 0) { throw 'Cannot create owned database' } }
    $env:PGDATABASE = 'pz14_notes'
    SaveLines 'versions.txt' @((& go version),(& $psqlPath --version),"PowerShell $($PSVersionTable.PSVersion)","driver pgx/v5 v5.7.2","run_id=$run","Data: 100000 rows per schema; no DROP/TRUNCATE; 200 requests per scenario, concurrency 8, warmup 10; pools 1 and 8")
    Lab @('-action','prepare','-run',$run) 'prepare.txt'
    Lab @('-action','plans','-run',$run) 'plans-before-load.txt'
    Lab @('-action','check','-run',$run) 'db-checks.txt'
    foreach ($kind in @('base','opt')) {
        foreach ($pool in @(1,8)) {
            $schema = "pz14_${run}_$kind"
            $stdout = Join-Path $tempDir "$kind-$pool-out.txt"
            $stderr = Join-Path $tempDir "$kind-$pool-err.txt"
            $argument = "-schema $schema -pool $pool"
            if ($kind -eq 'opt') { $argument += ' -optimized' }
            $server = Start-Process -FilePath (Join-Path $tempDir 'api.exe') -ArgumentList $argument -WindowStyle Hidden -PassThru -RedirectStandardOutput $stdout -RedirectStandardError $stderr
            $processes.Add($server)
            $port = 0
            for ($i=0;$i -lt 100;$i++) {
                $text = Get-Content -LiteralPath $stdout -Raw -ErrorAction SilentlyContinue
                if ($text -match 'listening=127\.0\.0\.1:(\d+)') { $port=[int]$Matches[1]; break }
                if ($server.HasExited) { throw 'Own server exited; check DB settings (credentials omitted)' }
                Start-Sleep -Milliseconds 100
            }
            if (-not $port) { throw 'API did not start' }
            $base = "http://127.0.0.1:$port"
            [void](HTTP 'GET' '/health' '' 200)
            if ($kind -eq 'opt' -and $pool -eq 8) {
                $created = HTTP 'POST' '/api/v1/notes' '{"title":"HTTP probe","content":"test"}' 201 | ConvertFrom-Json
                [void](HTTP 'GET' "/api/v1/notes/$($created.id)" '' 200)
                [void](HTTP 'PATCH' "/api/v1/notes/$($created.id)" '{"content":"changed"}' 200)
                [void](HTTP 'DELETE' "/api/v1/notes/$($created.id)" '' 204)
                [void](HTTP 'GET' "/api/v1/notes/$($created.id)" '' 404)
                [void](HTTP 'POST' '/api/v1/notes' '{"title":" "}' 400)
                [void](HTTP 'GET' '/api/v1/notes/abc' '' 400)
                [void](HTTP 'POST' '/api/v1/notes/many' '[{"title":"HTTP rollback"},{"title":" "}]' 400)
                $count = & $psqlPath -X -w -At -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM $schema.notes WHERE title='HTTP rollback';"
                if ($LASTEXITCODE -ne 0 -or $count -ne '0') { throw 'HTTP rollback probe remained' }
            }
            $loadArgs = @('-action','load','-run',$run,'-url',$base)
            if ($kind -eq 'opt') { $loadArgs += '-optimized' }
            Lab $loadArgs "$kind-pool$pool-load.jsonl"
            Stop-Process -Id $server.Id -Force; $server.WaitForExit()
        }
    }
    Lab @('-action','plans','-run',$run) 'plans-after-load.txt'
    Write-Host "PZ14 scenario passed. Small evidence files: $outputRoot"
    Write-Host 'Owned per-run schemas remain for inspection; no old data or global PostgreSQL settings were removed or changed.'
}
finally {
    foreach ($process in $processes) { if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force; $process.WaitForExit() } }
    foreach ($key in $oldPG.Keys) { [Environment]::SetEnvironmentVariable($key,$oldPG[$key],'Process') }
    if ($passwordPtr -ne [IntPtr]::Zero) { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($passwordPtr) }
    if ($secure) { $secure.Dispose() }
    if (Test-Path -LiteralPath $tempDir) {
        $resolved = (Resolve-Path -LiteralPath $tempDir).Path
        $root = (Resolve-Path -LiteralPath $env:TEMP).Path
        if (-not $resolved.StartsWith($root+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Split-Path -Leaf $resolved) -notlike 'pz14-*') { throw 'Unsafe cleanup path' }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}

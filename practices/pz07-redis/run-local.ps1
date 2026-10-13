param()

$ErrorActionPreference = 'Stop'
$moduleDir = $PSScriptRoot
$repoRoot = (Resolve-Path (Join-Path $moduleDir '..\..')).Path
$setup = Join-Path $repoRoot 'setup.ps1'
& $setup
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'Docker CLI was not found.' }
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go was not found after setup.ps1.' }
& docker info --format '{{.ServerVersion}}' *> $null
if ($LASTEXITCODE -ne 0) { throw 'Docker Engine is unavailable. Start Docker Desktop and retry.' }

function Get-FreePort {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    try { return ([System.Net.IPEndPoint]$listener.LocalEndpoint).Port }
    finally { $listener.Stop() }
}

$redisPort = Get-FreePort
$apiPort = Get-FreePort
while ($apiPort -eq $redisPort) { $apiPort = Get-FreePort }
$suffix = [Guid]::NewGuid().ToString('N').Substring(0, 12)
$containerName = "pz07-redis-$suffix"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) "pz07-$suffix"
$server = $null
$http = $null
$containerCreated = $false
$oldRedis = $env:REDIS_ADDR
$oldPort = $env:PORT
New-Item -ItemType Directory -Path $tempDir | Out-Null
try {
    Push-Location $moduleDir
    try {
        Write-Host 'Building PZ07 API...'
        & go build -o (Join-Path $tempDir 'pz07-server.exe') ./cmd/server
        if ($LASTEXITCODE -ne 0) { throw 'go build failed.' }
    } finally { Pop-Location }

    Write-Host "Starting owned Redis container $containerName on 127.0.0.1:$redisPort..."
    & docker run --detach --rm --name $containerName --publish "127.0.0.1:${redisPort}:6379" redis:7-alpine
    if ($LASTEXITCODE -ne 0) { throw 'Could not start the dedicated Redis container.' }
    $containerCreated = $true
    $ready = $false
    for ($i = 0; $i -lt 40; $i++) {
        & docker exec $containerName redis-cli ping *> $null
        if ($LASTEXITCODE -eq 0) { $ready = $true; break }
        Start-Sleep -Milliseconds 500
    }
    if (-not $ready) { throw 'The dedicated Redis container did not become ready within 20 seconds.' }

    $env:REDIS_ADDR = "127.0.0.1:$redisPort"
    $env:PORT = [string]$apiPort
    $stdout = Join-Path $tempDir 'server-out.log'
    $stderr = Join-Path $tempDir 'server-err.log'
    $server = Start-Process -FilePath (Join-Path $tempDir 'pz07-server.exe') -WorkingDirectory $moduleDir -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    Add-Type -AssemblyName System.Net.Http
    $http = [System.Net.Http.HttpClient]::new()
    $http.Timeout = [TimeSpan]::FromSeconds(5)
    $base = "http://127.0.0.1:$apiPort"
    $healthy = $false
    for ($i = 0; $i -lt 40; $i++) {
        try {
            $probe = $http.GetAsync("$base/get?key=readiness-probe").GetAwaiter().GetResult()
            if ([int]$probe.StatusCode -eq 404) { $healthy = $true; $probe.Dispose(); break }
            $probe.Dispose()
        } catch { }
        Start-Sleep -Milliseconds 250
    }
    if (-not $healthy) { throw "API did not become ready. See temporary server log: $stderr" }

    $key = "pz07-$suffix"
    $encodedKey = [uri]::EscapeDataString($key)
    $encodedValue = [uri]::EscapeDataString('hello redis')
    $set = $http.GetAsync("$base/set?key=$encodedKey&value=$encodedValue").GetAwaiter().GetResult()
    $setBody = $set.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    Write-Host "GET /set?key=$key&value=hello%20redis -> $([int]$set.StatusCode) $($setBody.Trim())"
    if ([int]$set.StatusCode -ne 200) { throw '/set returned an unexpected status.' }
    $set.Dispose()
    $get = $http.GetAsync("$base/get?key=$encodedKey").GetAwaiter().GetResult()
    $getBody = $get.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    Write-Host "GET /get?key=$key -> $([int]$get.StatusCode) $($getBody.Trim())"
    if ([int]$get.StatusCode -ne 200 -or (ConvertFrom-Json $getBody).value -ne 'hello redis') { throw '/get returned an unexpected result.' }
    $get.Dispose()
    $ttl = $http.GetAsync("$base/ttl?key=$encodedKey").GetAwaiter().GetResult()
    $ttlBody = $ttl.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    $ttlValue = (ConvertFrom-Json $ttlBody).ttl_seconds
    Write-Host "GET /ttl?key=$key -> $([int]$ttl.StatusCode) $($ttlBody.Trim())"
    if ([int]$ttl.StatusCode -ne 200 -or $ttlValue -le 0 -or $ttlValue -gt 10) { throw '/ttl was not within the expected range.' }
    $ttl.Dispose()
    Write-Host 'Waiting for the 10-second key TTL to expire...'
    $expired = $false
    for ($i = 0; $i -lt 15; $i++) {
        Start-Sleep -Seconds 1
        try {
            $check = $http.GetAsync("$base/get?key=$encodedKey").GetAwaiter().GetResult()
            if ([int]$check.StatusCode -eq 404) { $expired = $true; $check.Dispose(); break }
            $check.Dispose()
        } catch { }
    }
    if (-not $expired) { throw 'The key did not expire within the allowed 15-second window.' }
    Write-Host "GET /get?key=$key after expiry -> 404 (key not found)"
    Write-Host 'Scenario passed. Capture the three preceding real endpoint lines if screenshots are required.'
} finally {
    if ($http) { $http.Dispose() }
    if ($server -and -not $server.HasExited) { Stop-Process -Id $server.Id -Force -ErrorAction SilentlyContinue }
    if ($containerCreated) { & docker rm --force $containerName *> $null }
    if ($null -eq $oldRedis) { Remove-Item Env:REDIS_ADDR -ErrorAction SilentlyContinue } else { $env:REDIS_ADDR = $oldRedis }
    if ($null -eq $oldPort) { Remove-Item Env:PORT -ErrorAction SilentlyContinue } else { $env:PORT = $oldPort }
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}

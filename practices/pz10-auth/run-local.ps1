$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Net.Http

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$oldSecret = $env:JWT_SECRET
$oldPort = $env:APP_PORT
$oldHost = $env:APP_HOST
$oldTTL = $env:JWT_TTL
$oldRefreshTTL = $env:JWT_REFRESH_TTL
$tempDir = Join-Path $env:TEMP ('pz10-auth-' + [Guid]::NewGuid().ToString('N'))
$server = $null
$client = $null

function Invoke-API {
    param([string]$Method, [string]$Path, [object]$Body = $null, [string]$Token = '')
    $request = [System.Net.Http.HttpRequestMessage]::new([System.Net.Http.HttpMethod]::new($Method), "$baseURL$Path")
    try {
        if ($null -ne $Body) {
            $json = ConvertTo-Json -InputObject $Body -Compress
            $request.Content = [System.Net.Http.StringContent]::new($json, [Text.Encoding]::UTF8, 'application/json')
        }
        if ($Token) { $request.Headers.Authorization = [System.Net.Http.Headers.AuthenticationHeaderValue]::new('Bearer', $Token) }
        $response = $client.SendAsync($request).GetAwaiter().GetResult()
        try {
            $raw = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
            return @{ Status = [int]$response.StatusCode; Body = (ConvertFrom-Json -InputObject $raw) }
        } finally { $response.Dispose() }
    } finally { $request.Dispose() }
}
function Assert-Status {
    param([string]$Label, [object]$Response, [int]$Expected)
    if ($Response.Status -ne $Expected) { throw "$Label expected $Expected, got $($Response.Status): $($Response.Body | ConvertTo-Json -Compress)" }
    Write-Host "$Label -> $($Response.Status)"
}

try {
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    & (Join-Path $repoRoot 'setup.ps1')
    $goPath = (Get-Command go.exe -ErrorAction Stop).Source
    Write-Host "Go: $(& $goPath version)"
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start(); $port = $listener.LocalEndpoint.Port; $listener.Stop()
    $env:APP_HOST = '127.0.0.1'; $env:APP_PORT = [string]$port
    $env:JWT_TTL = '15m'; $env:JWT_REFRESH_TTL = '168h'
    $bytes = New-Object byte[] 48
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    $env:JWT_SECRET = [Convert]::ToBase64String($bytes)
    $exe = Join-Path $tempDir 'pz10-server.exe'
    Push-Location $PSScriptRoot
    try {
        & $goPath build -o $exe ./cmd/server
        if ($LASTEXITCODE -ne 0) { throw 'Go build failed.' }
    } finally { Pop-Location }
    $baseURL = "http://127.0.0.1:$port"
    $server = Start-Process -FilePath $exe -WorkingDirectory $PSScriptRoot -PassThru -WindowStyle Hidden -RedirectStandardOutput (Join-Path $tempDir 'out.log') -RedirectStandardError (Join-Path $tempDir 'err.log')
    $client = [System.Net.Http.HttpClient]::new()
    $client.Timeout = [TimeSpan]::FromSeconds(10)
    $ready = $false
    for ($i=0; $i -lt 60; $i++) {
        if ($server.HasExited) { break }
        try { if ((Invoke-API GET '/health').Status -eq 200) { $ready = $true; break } } catch {}
        Start-Sleep -Milliseconds 250
    }
    if (-not $ready) { throw 'Server did not become ready.' }
    Write-Host "API ready at $baseURL"
    $admin = Invoke-API POST '/api/v1/login' @{email='admin@example.com';password='secret123'}
    Assert-Status 'POST /api/v1/login admin' $admin 200
    if (-not $admin.Body.access_token -or -not $admin.Body.refresh_token) { throw 'Missing token pair.' }
    Write-Host 'admin: access and refresh tokens issued (values hidden)'
    $me = Invoke-API GET '/api/v1/me' $null $admin.Body.access_token
    Assert-Status 'GET /api/v1/me admin' $me 200
    Write-Host ($me.Body | ConvertTo-Json -Compress)
    $stats = Invoke-API GET '/api/v1/admin/stats' $null $admin.Body.access_token
    Assert-Status 'GET /api/v1/admin/stats admin' $stats 200
    Write-Host ($stats.Body | ConvertTo-Json -Compress)
    $user = Invoke-API POST '/api/v1/login' @{email='user@example.com';password='secret123'}
    Assert-Status 'POST /api/v1/login user' $user 200
    $denied = Invoke-API GET '/api/v1/admin/stats' $null $user.Body.access_token
    Assert-Status 'GET /api/v1/admin/stats user' $denied 403
    $own = Invoke-API GET '/api/v1/users/2' $null $user.Body.access_token
    Assert-Status 'GET /api/v1/users/2 user' $own 200
    $other = Invoke-API GET '/api/v1/users/1' $null $user.Body.access_token
    Assert-Status 'GET /api/v1/users/1 user' $other 403
    $missing = Invoke-API GET '/api/v1/me'
    Assert-Status 'GET /api/v1/me without token' $missing 401
    $rotated = Invoke-API POST '/api/v1/refresh' @{refresh_token=$admin.Body.refresh_token}
    Assert-Status 'POST /api/v1/refresh' $rotated 200
    if ($rotated.Body.access_token -eq $admin.Body.access_token) { throw 'Access token was not rotated.' }
    Write-Host 'old/new access tokens differ (values hidden)'
    $oldMe = Invoke-API GET '/api/v1/me' $null $admin.Body.access_token
    Assert-Status 'GET /api/v1/me with old access' $oldMe 200
    $newMe = Invoke-API GET '/api/v1/me' $null $rotated.Body.access_token
    Assert-Status 'GET /api/v1/me with new access' $newMe 200
    $replay = Invoke-API POST '/api/v1/refresh' @{refresh_token=$admin.Body.refresh_token}
    Assert-Status 'POST /api/v1/refresh replay' $replay 401
    Write-Host 'PZ10 scenario passed.'
} finally {
    if ($client) { $client.Dispose() }
    if ($server -and -not $server.HasExited) { Stop-Process -Id $server.Id -Force }
    $env:JWT_SECRET = $oldSecret; $env:APP_PORT = $oldPort; $env:APP_HOST = $oldHost
    $env:JWT_TTL = $oldTTL; $env:JWT_REFRESH_TTL = $oldRefreshTTL
    if (Test-Path -LiteralPath $tempDir) {
        $resolved = (Resolve-Path -LiteralPath $tempDir).Path
        $tempRoot = (Resolve-Path -LiteralPath $env:TEMP).Path
        if ($resolved.StartsWith($tempRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { Remove-Item -LiteralPath $resolved -Recurse -Force }
    }
}

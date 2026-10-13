$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Net.Http

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$psqlPath = 'D:\programDev\bin\psql.exe'
if (-not (Test-Path -LiteralPath $psqlPath)) {
    $psqlPath = (Get-Command psql.exe -ErrorAction Stop).Source
}
$oldDSN = $env:DB_DSN
$oldPGPassword = $env:PGPASSWORD
$oldAddr = $env:APP_ADDR
$oldCost = $env:BCRYPT_COST
$tempDir = Join-Path $env:TEMP ('pz09-auth-' + [Guid]::NewGuid().ToString('N'))
$serverExe = Join-Path $tempDir 'pz09-api.exe'
$stdoutPath = Join-Path $tempDir 'stdout.log'
$stderrPath = Join-Path $tempDir 'stderr.log'
$serverProcess = $null
$securePassword = $null
$passwordPtr = [IntPtr]::Zero
$password = $null

try {
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    & (Join-Path $repoRoot 'setup.ps1')
    if ($LASTEXITCODE -ne 0) { throw 'Go setup failed.' }
    $goPath = (Get-Command go.exe -ErrorAction Stop).Source
    Write-Host "Go: $(& $goPath version)"
    Write-Host "psql: $(& $psqlPath --version)"

    $postgresUser = $env:PGUSER
    if ([string]::IsNullOrWhiteSpace($postgresUser)) { $postgresUser = 'postgres' }
    $postgresPort = $env:PGPORT
    if ([string]::IsNullOrWhiteSpace($postgresPort)) { $postgresPort = '5432' }
    if ([string]::IsNullOrEmpty($env:PGPASSWORD)) {
        $securePassword = Read-Host 'PostgreSQL password (input is hidden)' -AsSecureString
        $passwordPtr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($securePassword)
        $password = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($passwordPtr)
        $env:PGPASSWORD = $password
    } else {
        $password = $env:PGPASSWORD
    }

    $dbExists = @(& $psqlPath -X -w -h 127.0.0.1 -p $postgresPort -U $postgresUser -d postgres -v ON_ERROR_STOP=1 -At -c "SELECT 1 FROM pg_database WHERE datname = 'pz9_auth';")
    if ($LASTEXITCODE -ne 0) { throw 'Could not connect to PostgreSQL.' }
    if ($dbExists -notcontains '1') {
        & $psqlPath -X -w -h 127.0.0.1 -p $postgresPort -U $postgresUser -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE pz9_auth;'
        if ($LASTEXITCODE -ne 0) { throw 'Could not create pz9_auth.' }
    }
    $existingTables = @(& $psqlPath -X -w -h 127.0.0.1 -p $postgresPort -U $postgresUser -d pz9_auth -v ON_ERROR_STOP=1 -At -c "SELECT tablename FROM pg_tables WHERE schemaname = 'public';")
    if ($LASTEXITCODE -ne 0) { throw 'Could not inspect pz9_auth.' }
    if (@($existingTables | Where-Object { $_ -and $_ -ne 'users' }).Count -gt 0) {
        throw 'pz9_auth contains unrelated tables; migration was not run.'
    }

    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    $port = $listener.LocalEndpoint.Port
    $listener.Stop()
    $env:APP_ADDR = "127.0.0.1:$port"
    if ([string]::IsNullOrEmpty($env:BCRYPT_COST)) { $env:BCRYPT_COST = '12' }

    Push-Location $PSScriptRoot
    try {
        & $goPath build -o $serverExe ./cmd/api
        if ($LASTEXITCODE -ne 0) { throw 'Go build failed.' }
    } finally { Pop-Location }

    $encodedUser = [Uri]::EscapeDataString($postgresUser)
    $encodedPassword = [Uri]::EscapeDataString($password)
    $env:DB_DSN = "postgres://${encodedUser}:${encodedPassword}@127.0.0.1:${postgresPort}/pz9_auth?sslmode=disable"
    $baseURL = "http://127.0.0.1:$port"
    Write-Host "Starting API on $baseURL"
    $serverProcess = Start-Process -FilePath $serverExe -WorkingDirectory $PSScriptRoot -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath
    $ready = $false
    for ($attempt = 0; $attempt -lt 60; $attempt++) {
        if ($serverProcess.HasExited) { break }
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Uri "$baseURL/health" -TimeoutSec 2
            if ($response.StatusCode -eq 200) { $ready = $true; break }
        } catch { Start-Sleep -Milliseconds 250 }
    }
    if (-not $ready) {
        if (Test-Path -LiteralPath $stderrPath) { Get-Content -LiteralPath $stderrPath }
        throw 'API did not become ready within 15 seconds; check PostgreSQL access.'
    }

    function Invoke-CheckedAuth {
        param([string]$Path, [object]$Body, [int]$ExpectedStatus)
        $client = [System.Net.Http.HttpClient]::new()
        $client.Timeout = [TimeSpan]::FromSeconds(15)
        $payload = [System.Net.Http.StringContent]::new(($Body | ConvertTo-Json -Compress), [System.Text.Encoding]::UTF8, 'application/json')
        $response = $null
        try {
            $response = $client.PostAsync("$baseURL$Path", $payload).GetAwaiter().GetResult()
            $status = [int]$response.StatusCode
            $content = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        } finally {
            if ($null -ne $response) { $response.Dispose() }
            $payload.Dispose()
            $client.Dispose()
        }
        if ($status -ne $ExpectedStatus) { throw "$Path returned $status (expected $ExpectedStatus): $content" }
        $json = $content | ConvertFrom-Json
        if ($content -match '(?i)password|\$2[aby]\$') { throw 'Response contains password material.' }
        Write-Host "HTTP POST $Path -> $status $content"
        return [PSCustomObject]@{ Status = $status; Json = $json; Content = $content }
    }

    $unique = [Guid]::NewGuid().ToString('N')
    $email = "pz09-$unique@example.test"
    $testPassword = 'Practice-' + [Guid]::NewGuid().ToString('N')
    $register = Invoke-CheckedAuth -Path '/auth/register' -ExpectedStatus 201 -Body @{ email = $email; password = $testPassword }
    if ($register.Json.status -ne 'ok' -or $register.Json.user.email -ne $email) { throw 'Registration response is invalid.' }
    $duplicate = Invoke-CheckedAuth -Path '/auth/register' -ExpectedStatus 409 -Body @{ email = $email; password = $testPassword }
    if ($duplicate.Json.error -ne 'email_taken') { throw 'Duplicate email response is invalid.' }
    $login = Invoke-CheckedAuth -Path '/auth/login' -ExpectedStatus 200 -Body @{ email = $email; password = $testPassword }
    if ($login.Json.status -ne 'ok' -or $login.Json.user.id -ne $register.Json.user.id) { throw 'Login response is invalid.' }
    $wrong = Invoke-CheckedAuth -Path '/auth/login' -ExpectedStatus 401 -Body @{ email = $email; password = 'wrong-password' }
    $missing = Invoke-CheckedAuth -Path '/auth/login' -ExpectedStatus 401 -Body @{ email = "missing-$unique@example.test"; password = 'wrong-password' }
    if ($wrong.Content -ne $missing.Content -or $wrong.Json.error -ne 'invalid_credentials') { throw 'Login error reveals email existence.' }

    $schema = @(& $psqlPath -X -w -h 127.0.0.1 -p $postgresPort -U $postgresUser -d pz9_auth -v ON_ERROR_STOP=1 -At -c "SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name='users' ORDER BY ordinal_position;")
    if ($LASTEXITCODE -ne 0 -or $schema -notcontains 'password_hash') { throw 'AutoMigrate did not create password_hash.' }
    $hashCheck = @(& $psqlPath -X -w -h 127.0.0.1 -p $postgresPort -U $postgresUser -d pz9_auth -v ON_ERROR_STOP=1 -At -c "SELECT count(*) FROM users WHERE email = '$email' AND left(password_hash, 1) = chr(36) AND substring(password_hash from 2 for 1) = '2';")
    if ($LASTEXITCODE -ne 0 -or $hashCheck -notcontains '1') { throw 'No bcrypt hash was found for the test user.' }
    Write-Host 'AutoMigrate: users.password_hash exists; stored value has bcrypt prefix.'
    Write-Host 'PZ09 scenario passed.'
} finally {
    if ($null -ne $serverProcess -and -not $serverProcess.HasExited) { Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue }
    $env:DB_DSN = $oldDSN
    $env:PGPASSWORD = $oldPGPassword
    $env:APP_ADDR = $oldAddr
    $env:BCRYPT_COST = $oldCost
    if ($passwordPtr -ne [IntPtr]::Zero) { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($passwordPtr) }
    $password = $null
    $testPassword = $null
    $securePassword = $null
    $resolvedTempRoot = [System.IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\'
    $resolvedTempDir = [System.IO.Path]::GetFullPath($tempDir)
    if ($resolvedTempDir.StartsWith($resolvedTempRoot, [System.StringComparison]::OrdinalIgnoreCase) -and (Test-Path -LiteralPath $tempDir)) {
        Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

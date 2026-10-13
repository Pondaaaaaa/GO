$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$moduleDir = $PSScriptRoot
$psqlPath = 'D:\programDev\bin\psql.exe'
if (-not (Test-Path -LiteralPath $psqlPath)) {
    $psqlCommand = Get-Command psql.exe -ErrorAction SilentlyContinue
    if ($null -eq $psqlCommand) {
        throw 'psql.exe was not found. Install PostgreSQL Command Line Tools or add psql.exe to PATH.'
    }
    $psqlPath = $psqlCommand.Source
}

$previousDBDSN = $env:DB_DSN
$previousPGPassword = $env:PGPASSWORD
$previousPort = $env:PORT
$runId = [Guid]::NewGuid().ToString('N')
$tempDir = Join-Path $env:TEMP "pz06-gorm-$runId"
$serverExe = Join-Path $tempDir 'pz06-server.exe'
$stdoutPath = Join-Path $tempDir 'server.stdout.log'
$stderrPath = Join-Path $tempDir 'server.stderr.log'
$serverProcess = $null
$securePassword = $null
$passwordPtr = [IntPtr]::Zero
$password = $null

try {
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

    & (Join-Path $repoRoot 'setup.ps1')
    if ($LASTEXITCODE -ne 0) { throw 'setup.ps1 failed.' }

    $goCommand = Get-Command go.exe -ErrorAction Stop
    $goVersion = & $goCommand.Source version
    $psqlVersion = & $psqlPath --version
    Write-Host "Go: $goVersion"
    Write-Host "psql: $psqlVersion"

    $postgresUser = $env:PGUSER
    if ([string]::IsNullOrWhiteSpace($postgresUser)) { $postgresUser = 'postgres' }
    $securePassword = Read-Host 'PostgreSQL password (input is hidden)' -AsSecureString
    $passwordPtr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($securePassword)
    $password = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($passwordPtr)
    $env:PGPASSWORD = $password

    $dbExists = & $psqlPath -X -w -h 127.0.0.1 -p 5432 -U $postgresUser -d postgres -v ON_ERROR_STOP=1 -At -c "SELECT 1 FROM pg_database WHERE datname = 'pz6_gorm';"
    if ($LASTEXITCODE -ne 0) { throw 'Could not inspect PostgreSQL databases.' }
    if ($dbExists -notcontains '1') {
        & $psqlPath -X -w -h 127.0.0.1 -p 5432 -U $postgresUser -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE pz6_gorm;'
        if ($LASTEXITCODE -ne 0) { throw 'Could not create database pz6_gorm.' }
    }

    $existingTables = @(& $psqlPath -X -w -h 127.0.0.1 -p 5432 -U $postgresUser -d pz6_gorm -v ON_ERROR_STOP=1 -At -c "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;")
    if ($LASTEXITCODE -ne 0) { throw 'Could not inspect the pz6_gorm schema.' }
    $allowedTables = @('users', 'notes', 'tags', 'note_tags')
    $unexpectedTables = @($existingTables | Where-Object { $_ -and $_ -notin $allowedTables })
    if ($unexpectedTables.Count -gt 0) {
        throw "Database pz6_gorm contains unrelated public tables ($($unexpectedTables -join ', ')); no migration was run."
    }
    if ($existingTables.Count -gt 0) {
        Write-Host 'Existing practice tables found; AutoMigrate will apply additive changes only.'
    }

    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    $port = $listener.LocalEndpoint.Port
    $listener.Stop()
    $env:PORT = [string]$port

    Push-Location $moduleDir
    try {
        & $goCommand.Source build -o $serverExe ./cmd/server
        if ($LASTEXITCODE -ne 0) { throw 'Building the local API server failed.' }
    }
    finally {
        Pop-Location
    }

    $encodedUser = [Uri]::EscapeDataString($postgresUser)
    $encodedPassword = [Uri]::EscapeDataString($password)
    $env:DB_DSN = "postgres://${encodedUser}:${encodedPassword}@127.0.0.1:5432/pz6_gorm?sslmode=disable"

    Write-Host "Starting this practice server at http://127.0.0.1:$port/"
    $serverProcess = Start-Process -FilePath $serverExe -WorkingDirectory $moduleDir -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath

    $baseURL = "http://127.0.0.1:$port"
    $ready = $false
    $readyTimer = [System.Diagnostics.Stopwatch]::StartNew()
    for ($attempt = 0; $attempt -lt 60 -and $readyTimer.Elapsed.TotalSeconds -lt 15; $attempt++) {
        if ($serverProcess.HasExited) { break }
        try {
            $health = Invoke-WebRequest -UseBasicParsing -Uri "$baseURL/health" -TimeoutSec 2
            if ([int]$health.StatusCode -eq 200 -and ($health.Content | ConvertFrom-Json).status -eq 'ok') {
                $ready = $true
                break
            }
        }
        catch {
            Start-Sleep -Milliseconds 250
        }
    }
    $readyTimer.Stop()
    if (-not $ready) {
        if (Test-Path $stdoutPath) { Get-Content $stdoutPath }
        if (Test-Path $stderrPath) { Get-Content $stderrPath }
        throw 'The local server did not become healthy within 15 seconds.'
    }

    function Invoke-CheckedApi {
        param(
            [Parameter(Mandatory = $true)][string]$Method,
            [Parameter(Mandatory = $true)][string]$Path,
            [Parameter(Mandatory = $true)][int]$ExpectedStatus,
            [AllowNull()][object]$Body
        )

        $requestParams = @{
            Method = $Method
            Uri = "$baseURL$Path"
            UseBasicParsing = $true
            TimeoutSec = 10
            ErrorAction = 'Stop'
        }
        if ($null -ne $Body) {
            $requestParams.ContentType = 'application/json; charset=utf-8'
            $requestParams.Body = $Body | ConvertTo-Json -Depth 8 -Compress
        }

        $status = 0
        $content = ''
        $contentType = ''
        try {
            $response = Invoke-WebRequest @requestParams
            $status = [int]$response.StatusCode
            $content = [string]$response.Content
            $contentType = [string]$response.Headers['Content-Type']
        }
        catch {
            $response = $_.Exception.Response
            if ($null -eq $response) { throw }
            $status = [int]$response.StatusCode
            $contentType = [string]$response.Headers['Content-Type']
            $reader = [System.IO.StreamReader]::new($response.GetResponseStream())
            try { $content = $reader.ReadToEnd() }
            finally { $reader.Dispose() }
        }

        if ($status -ne $ExpectedStatus) {
            throw "$Method $Path returned HTTP $status; expected $ExpectedStatus. Body: $content"
        }
        if ($contentType -notmatch 'application/json') {
            throw "$Method $Path did not return a JSON Content-Type."
        }
        $json = $null
        if ($content) { $json = $content | ConvertFrom-Json }
        Write-Host "HTTP $Method $Path -> $status"
        if ($content) { Write-Host ($json | ConvertTo-Json -Depth 8) }
        return [PSCustomObject]@{ Status = $status; Json = $json; Content = $content }
    }

    Write-Host "`n=== Health check ==="
    $healthResult = Invoke-CheckedApi -Method GET -Path '/health' -ExpectedStatus 200 -Body $null
    if ($healthResult.Json.status -ne 'ok') { throw 'Health JSON did not contain status=ok.' }

    $unique = [Guid]::NewGuid().ToString('N')
    $email = "pz06-$unique@example.test"
    $displayName = "PZ06 User $unique"

    Write-Host "`n=== Create user (1:N parent) ==="
    $userResult = Invoke-CheckedApi -Method POST -Path '/users' -ExpectedStatus 201 -Body @{ name = $displayName; email = $email }
    $userId = [uint64]$userResult.Json.id
    if ($userId -eq 0 -or $userResult.Json.email -ne $email) { throw 'POST /users returned an invalid user.' }

    Write-Host "`n=== Duplicate email error ==="
    $null = Invoke-CheckedApi -Method POST -Path '/users' -ExpectedStatus 409 -Body @{ name = $displayName; email = $email }

    Write-Host "`n=== Create note with two tags (M:N) ==="
    $tagOne = "pz06-a-$unique"
    $tagTwo = "pz06-b-$unique"
    $noteResult = Invoke-CheckedApi -Method POST -Path '/notes' -ExpectedStatus 201 -Body @{
        title = "PZ06 note $unique"
        content = 'Created by run-local.ps1 for the database relations practice.'
        userId = $userId
        tags = @($tagOne, $tagTwo)
    }
    $noteId = [uint64]$noteResult.Json.id
    if ($noteId -eq 0 -or [uint64]$noteResult.Json.userId -ne $userId) { throw 'POST /notes returned invalid identifiers.' }

    Write-Host "`n=== Read note with Preload(User, Tags) ==="
    $loadedNote = Invoke-CheckedApi -Method GET -Path "/notes/$noteId" -ExpectedStatus 200 -Body $null
    if ([uint64]$loadedNote.Json.user.id -ne $userId -or $loadedNote.Json.user.email -ne $email) {
        throw 'GET /notes/{id} did not preload the expected author.'
    }
    $loadedTags = @($loadedNote.Json.tags | ForEach-Object { $_.name })
    if ($loadedTags.Count -ne 2 -or $tagOne -notin $loadedTags -or $tagTwo -notin $loadedTags) {
        throw 'GET /notes/{id} did not preload both tags.'
    }
    $firstTagIds = @($loadedNote.Json.tags | ForEach-Object { [uint64]$_.id } | Sort-Object)

    Write-Host "`n=== Reuse existing tags on another note ==="
    $secondNote = Invoke-CheckedApi -Method POST -Path '/notes' -ExpectedStatus 201 -Body @{
        title = "PZ06 second note $unique"
        content = 'The existing tag rows should be reused.'
        userId = $userId
        tags = @($tagOne, $tagTwo)
    }
    $secondNoteId = [uint64]$secondNote.Json.id
    $secondLoaded = Invoke-CheckedApi -Method GET -Path "/notes/$secondNoteId" -ExpectedStatus 200 -Body $null
    $secondTagIds = @($secondLoaded.Json.tags | ForEach-Object { [uint64]$_.id } | Sort-Object)
    if (Compare-Object -ReferenceObject $firstTagIds -DifferenceObject $secondTagIds) {
        throw 'Reusing tag names created different tag rows.'
    }

    Write-Host "`n=== Read user with related notes (1:N) ==="
    $loadedUser = Invoke-CheckedApi -Method GET -Path "/users/$userId" -ExpectedStatus 200 -Body $null
    $relatedNoteIds = @($loadedUser.Json.notes | ForEach-Object { [uint64]$_.id })
    if ($noteId -notin $relatedNoteIds -or $secondNoteId -notin $relatedNoteIds) {
        throw 'GET /users/{id} did not include both related notes.'
    }

    Write-Host "`n=== Error responses ==="
    $null = Invoke-CheckedApi -Method POST -Path '/users' -ExpectedStatus 400 -Body @{ name = ''; email = 'not-an-email' }
    $null = Invoke-CheckedApi -Method GET -Path '/notes/999999999' -ExpectedStatus 404 -Body $null

    Write-Host "`n=== PostgreSQL schema from psql ==="
    foreach ($table in @('users', 'notes', 'tags', 'note_tags')) {
        Write-Host "`n--- \d $table ---"
        & $psqlPath -X -w -h 127.0.0.1 -p 5432 -U $postgresUser -d pz6_gorm -v ON_ERROR_STOP=1 -c "\d $table"
        if ($LASTEXITCODE -ne 0) { throw "Could not inspect table $table." }
    }
    Write-Host "`nPZ06 local scenario passed. User ID=$userId; note IDs=$noteId,$secondNoteId; tags=$($loadedTags -join ', ')."
}
finally {
    if ($null -ne $serverProcess -and -not $serverProcess.HasExited) {
        Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue
        $serverProcess.WaitForExit(5000) | Out-Null
        Write-Host 'Stopped only the PZ06 server process started by this script.'
    }
    if ($null -ne $serverProcess) {
        if (Test-Path $stdoutPath) {
            $serverStdout = Get-Content $stdoutPath -ErrorAction SilentlyContinue
            if ($serverStdout) { Write-Host "`n=== Server stdout ==="; $serverStdout }
        }
        if (Test-Path $stderrPath) {
            $serverStderr = Get-Content $stderrPath -ErrorAction SilentlyContinue
            if ($serverStderr) { Write-Host "`n=== Server logs ==="; $serverStderr }
        }
    }

    if ($null -eq $previousDBDSN) { Remove-Item Env:DB_DSN -ErrorAction SilentlyContinue }
    else { $env:DB_DSN = $previousDBDSN }
    if ($null -eq $previousPGPassword) { Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue }
    else { $env:PGPASSWORD = $previousPGPassword }
    if ($null -eq $previousPort) { Remove-Item Env:PORT -ErrorAction SilentlyContinue }
    else { $env:PORT = $previousPort }

    $password = $null
    $encodedPassword = $null
    if ($passwordPtr -ne [IntPtr]::Zero) { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($passwordPtr) }
    if ($null -ne $securePassword) { $securePassword.Dispose() }
    if (Test-Path $tempDir) { Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue }
}

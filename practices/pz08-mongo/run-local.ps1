param()

$ErrorActionPreference = 'Stop'
$moduleDir = $PSScriptRoot
$repoRoot = (Resolve-Path (Join-Path $moduleDir '..\..')).Path
& (Join-Path $repoRoot 'setup.ps1')
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'Docker CLI was not found.' }
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go was not found after setup.ps1.' }
& docker compose version *> $null
if ($LASTEXITCODE -ne 0) { throw 'Docker Compose is unavailable.' }
& docker info --format '{{.ServerVersion}}' *> $null
if ($LASTEXITCODE -ne 0) { throw 'Docker Engine is unavailable. Start Docker Desktop and retry.' }

function Get-FreePort {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    try { return ([System.Net.IPEndPoint]$listener.LocalEndpoint).Port }
    finally { $listener.Stop() }
}

function Invoke-Api {
    param([System.Net.Http.HttpClient]$Client, [string]$Method, [string]$Uri, [string]$Json)
    $request = [System.Net.Http.HttpRequestMessage]::new([System.Net.Http.HttpMethod]::new($Method), $Uri)
    if ($PSBoundParameters.ContainsKey('Json')) { $request.Content = [System.Net.Http.StringContent]::new($Json, [System.Text.Encoding]::UTF8, 'application/json') }
    try {
        $response = $Client.SendAsync($request).GetAwaiter().GetResult()
        try {
            $body = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
            return @{ Status = [int]$response.StatusCode; Body = $body.Trim() }
        } finally { $response.Dispose() }
    } finally { $request.Dispose() }
}

function Assert-Status {
    param([hashtable]$Response, [int]$Expected, [string]$Label)
    if ($Response.Status -ne $Expected) { throw "$Label returned HTTP $($Response.Status), expected $Expected." }
}

$mongoPort = Get-FreePort
$apiPort = Get-FreePort
while ($apiPort -eq $mongoPort) { $apiPort = Get-FreePort }
$suffix = [Guid]::NewGuid().ToString('N').Substring(0, 12)
$project = "pz08mongo$suffix"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) "pz08-$suffix"
$composeFile = Join-Path $moduleDir 'docker-compose.yml'
$oldEnvironment = @{}
foreach ($name in @('MONGO_USER', 'MONGO_PASSWORD', 'MONGO_PORT', 'MONGO_URI', 'MONGO_DB', 'HTTP_ADDR')) {
    $oldEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
$server = $null
$http = $null
$projectCreated = $false
New-Item -ItemType Directory -Path $tempDir | Out-Null
try {
    Push-Location $moduleDir
    try {
        Write-Host 'Building PZ08 API...'
        & go build -o (Join-Path $tempDir 'pz08-api.exe') ./cmd/api
        if ($LASTEXITCODE -ne 0) { throw 'go build failed.' }
    } finally { Pop-Location }

    $env:MONGO_USER = "pz08$suffix"
    $env:MONGO_PASSWORD = [Guid]::NewGuid().ToString('N') + [Guid]::NewGuid().ToString('N')
    $env:MONGO_PORT = [string]$mongoPort
    $env:MONGO_DB = 'pz8'
    $env:MONGO_URI = "mongodb://$($env:MONGO_USER):$($env:MONGO_PASSWORD)@127.0.0.1:$mongoPort/?authSource=admin"
    $env:HTTP_ADDR = "127.0.0.1:$apiPort"
    Write-Host "Starting owned Compose project $project with MongoDB on a free local port..."
    $projectCreated = $true
    $savedErrorAction = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        & docker compose --progress quiet --project-name $project --file $composeFile up --detach --wait --wait-timeout 120 *> $null
        $composeExit = $LASTEXITCODE
    } finally { $ErrorActionPreference = $savedErrorAction }
    if ($composeExit -ne 0) { throw 'The dedicated MongoDB Compose project did not become healthy.' }

    $stdout = Join-Path $tempDir 'api-out.log'
    $stderr = Join-Path $tempDir 'api-err.log'
    $server = Start-Process -FilePath (Join-Path $tempDir 'pz08-api.exe') -WorkingDirectory $moduleDir -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    Add-Type -AssemblyName System.Net.Http
    $http = [System.Net.Http.HttpClient]::new()
    $http.Timeout = [TimeSpan]::FromSeconds(7)
    $base = "http://127.0.0.1:$apiPort"
    $healthy = $false
    $lastHealthError = ''
    for ($i = 0; $i -lt 40; $i++) {
        try {
            $probe = Invoke-Api -Client $http -Method GET -Uri "$base/health"
            if ($probe.Status -eq 200) { $healthy = $true; break }
            $lastHealthError = "HTTP $($probe.Status)"
        } catch { $lastHealthError = $_.Exception.Message }
        Start-Sleep -Milliseconds 500
    }
    if (-not $healthy) {
        if ($server.HasExited) {
            Write-Host "API exited with code $($server.ExitCode)."
            if (Test-Path $stderr) { Get-Content -Path $stderr -Tail 5 | ForEach-Object { Write-Host $_ } }
        } else { Write-Host "Last health check: $lastHealthError" }
        throw 'API did not become ready after 20 seconds.'
    }
    Write-Host "GET /health -> $($probe.Status) $($probe.Body)"

    $titleA = "PZ08 $suffix A"
    $titleB = "PZ08 $suffix [B]"
    $createdA = Invoke-Api -Client $http -Method POST -Uri "$base/api/v1/notes" -Json (@{title=$titleA; content='first content'} | ConvertTo-Json -Compress)
    Write-Host "POST /api/v1/notes -> $($createdA.Status) $($createdA.Body)"
    Assert-Status $createdA 201 'Create A'
    $noteA = ConvertFrom-Json $createdA.Body
    if ($noteA.title -ne $titleA -or $noteA.id -notmatch '^[0-9a-f]{24}$') { throw 'Create A returned an invalid note.' }
    $createdB = Invoke-Api -Client $http -Method POST -Uri "$base/api/v1/notes" -Json (@{title=$titleB; content='second content'} | ConvertTo-Json -Compress)
    Write-Host "POST /api/v1/notes (second note) -> $($createdB.Status) $($createdB.Body)"
    Assert-Status $createdB 201 'Create B'
    $noteB = ConvertFrom-Json $createdB.Body

    $q = [Uri]::EscapeDataString("PZ08 $suffix")
    $list = Invoke-Api -Client $http -Method GET -Uri "$base/api/v1/notes?q=$q&limit=1&skip=1"
    Write-Host "GET /api/v1/notes?q=...&limit=1&skip=1 -> $($list.Status) $($list.Body)"
    Assert-Status $list 200 'List with filter and pagination'
    $listData = ConvertFrom-Json $list.Body
    if ($listData.items.Count -ne 1 -or $listData.items[0].id -ne $noteA.id -or $listData.limit -ne 1 -or $listData.skip -ne 1) { throw 'Pagination result was invalid.' }
    $literalQ = [Uri]::EscapeDataString('[')
    $filtered = Invoke-Api -Client $http -Method GET -Uri "$base/api/v1/notes?q=$literalQ&limit=10&skip=0"
    Write-Host "GET /api/v1/notes?q=%5B -> $($filtered.Status) $($filtered.Body)"
    Assert-Status $filtered 200 'Literal title filter'
    $filteredData = ConvertFrom-Json $filtered.Body
    if ($filteredData.items.Count -ne 1 -or $filteredData.items[0].id -ne $noteB.id) { throw 'Literal title filter returned an unexpected note.' }

    $byID = Invoke-Api -Client $http -Method GET -Uri "$base/api/v1/notes/$($noteA.id)"
    Write-Host "GET /api/v1/notes/{id} -> $($byID.Status) $($byID.Body)"
    Assert-Status $byID 200 'Get by ID'
    if ((ConvertFrom-Json $byID.Body).title -ne $titleA) { throw 'Get by ID returned an unexpected note.' }

    $patched = Invoke-Api -Client $http -Method PATCH -Uri "$base/api/v1/notes/$($noteA.id)" -Json (@{content='updated content'} | ConvertTo-Json -Compress)
    Write-Host "PATCH /api/v1/notes/{id} -> $($patched.Status) $($patched.Body)"
    Assert-Status $patched 200 'Partial update'
    $patchedNote = ConvertFrom-Json $patched.Body
    if ($patchedNote.title -ne $titleA -or $patchedNote.content -ne 'updated content') { throw 'Partial update changed an unexpected field.' }

    $duplicate = Invoke-Api -Client $http -Method POST -Uri "$base/api/v1/notes" -Json (@{title=$titleA; content='duplicate'} | ConvertTo-Json -Compress)
    Write-Host "POST duplicate title -> $($duplicate.Status) $($duplicate.Body)"
    Assert-Status $duplicate 409 'Duplicate title'
    $duplicatePatch = Invoke-Api -Client $http -Method PATCH -Uri "$base/api/v1/notes/$($noteB.id)" -Json (@{title=$titleA} | ConvertTo-Json -Compress)
    Write-Host "PATCH duplicate title -> $($duplicatePatch.Status) $($duplicatePatch.Body)"
    Assert-Status $duplicatePatch 409 'Duplicate update'
    $invalidID = Invoke-Api -Client $http -Method GET -Uri "$base/api/v1/notes/not-an-object-id"
    Write-Host "GET invalid ObjectID -> $($invalidID.Status) $($invalidID.Body)"
    Assert-Status $invalidID 400 'Invalid ObjectID'
    $missing = Invoke-Api -Client $http -Method GET -Uri "$base/api/v1/notes/000000000000000000000000"
    Write-Host "GET missing note -> $($missing.Status) $($missing.Body)"
    Assert-Status $missing 404 'Missing note'

    $deleted = Invoke-Api -Client $http -Method DELETE -Uri "$base/api/v1/notes/$($noteA.id)"
    Write-Host "DELETE /api/v1/notes/{id} -> $($deleted.Status) body-length=$($deleted.Body.Length)"
    Assert-Status $deleted 204 'Delete note'
    if ($deleted.Body.Length -ne 0) { throw 'DELETE 204 contained a body.' }
    $afterDelete = Invoke-Api -Client $http -Method GET -Uri "$base/api/v1/notes/$($noteA.id)"
    Write-Host "GET deleted note -> $($afterDelete.Status) $($afterDelete.Body)"
    Assert-Status $afterDelete 404 'Deleted note lookup'
    Write-Host 'Scenario passed: five CRUD operations, filter, pagination, duplicates, ObjectID errors, 404 and empty 204.'
} finally {
    if ($http) { $http.Dispose() }
    if ($server -and -not $server.HasExited) { Stop-Process -Id $server.Id -Force -ErrorAction SilentlyContinue }
    if ($projectCreated) {
        $savedErrorAction = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        try {
            & docker compose --progress quiet --project-name $project --file $composeFile down --volumes --remove-orphans *> $null
            $downExit = $LASTEXITCODE
        } finally { $ErrorActionPreference = $savedErrorAction }
        if ($downExit -ne 0) { Write-Warning "Could not fully remove owned Compose project $project." }
    }
    foreach ($name in $oldEnvironment.Keys) {
        if ($null -eq $oldEnvironment[$name]) { Remove-Item -Path "Env:$name" -ErrorAction SilentlyContinue }
        else { Set-Item -Path "Env:$name" -Value $oldEnvironment[$name] }
    }
    $oldEnvironment.Clear()
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}

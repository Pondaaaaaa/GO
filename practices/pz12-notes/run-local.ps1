param([switch]$KeepOpen)

$ErrorActionPreference = 'Stop'
$moduleRoot = $PSScriptRoot
$repoRoot = (Resolve-Path (Join-Path $moduleRoot '..\..')).Path
& (Join-Path $repoRoot 'setup.ps1')
$oldPort = $env:PORT

$listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
$listener.Start()
$port = $listener.LocalEndpoint.Port
$listener.Stop()
$env:PORT = [string]$port
$sessionDir = Join-Path $env:TEMP ('pz12-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $sessionDir | Out-Null
$server = $null
$ready = $false

function Request([string]$method, [string]$path, [string]$data, [int]$expected) {
    $headers = Join-Path $sessionDir 'headers.txt'
    $body = Join-Path $sessionDir 'body.txt'
    $curlArgs = @('--noproxy', '*', '-sS', '--connect-timeout', '2', '--max-time', '5', '-X', $method, '-D', $headers, '-o', $body, '-w', '%{http_code}')
    if ($data -ne '') {
        $payload = Join-Path $sessionDir 'request.json'
        [System.IO.File]::WriteAllText($payload, $data, [System.Text.UTF8Encoding]::new($false))
        $curlArgs += @('-H', 'Content-Type: application/json', '--data-binary', "@$payload")
    }
    $status = & curl.exe @curlArgs ("http://127.0.0.1:$port$path")
    if ($LASTEXITCODE -ne 0) { throw "curl failed for $method $path" }
    $raw = [System.IO.File]::ReadAllText($body, [System.Text.Encoding]::UTF8)
    $text = $raw.Trim()
    if ($path -eq '/docs/doc.json') { Write-Host "$method $path -> $status (Swagger 2.0 JSON)" }
    elseif ($text -eq '') { Write-Host "$method $path -> $status" }
    else { Write-Host "$method $path -> $status $text" }
    if ([int]$status -ne $expected) { throw "Expected $expected, received $status" }
    if ($expected -ne 204 -and -not ([IO.File]::ReadAllText($headers) -match '(?im)^Content-Type: application/json')) { throw 'Missing JSON Content-Type' }
    if ($expected -eq 204 -and $raw.Length -ne 0) { throw 'DELETE 204 had a response body' }
    return $text
}

try {
    Push-Location $moduleRoot
    try { go build -o (Join-Path $sessionDir 'notes-api.exe') ./cmd/api; if ($LASTEXITCODE -ne 0) { throw 'go build failed' } }
    finally { Pop-Location }
    $server = Start-Process -FilePath (Join-Path $sessionDir 'notes-api.exe') -PassThru -WindowStyle Hidden -RedirectStandardOutput (Join-Path $sessionDir 'server-out.txt') -RedirectStandardError (Join-Path $sessionDir 'server-err.txt')
    for ($i = 0; $i -lt 50; $i++) {
        try {
            $probe = & curl.exe --noproxy '*' -sS --max-time 1 -o NUL -w '%{http_code}' "http://127.0.0.1:$port/api/v1/notes" 2>$null
            if ($probe -eq '200') { $ready = $true; break }
        } catch { }
        Start-Sleep -Milliseconds 100
    }
    if (-not $ready) { throw 'API did not start' }
    Write-Host "API started on 127.0.0.1:$port"
    [void](Request 'GET' '/api/v1/notes' '' 200)
    $created = Request 'POST' '/api/v1/notes' '{"title":"First note","content":"Test body"}' 201 | ConvertFrom-Json
    if ($created.id -ne 1) { throw 'Unexpected new note ID' }
    [void](Request 'GET' '/api/v1/notes' '' 200)
    [void](Request 'GET' '/api/v1/notes/1' '' 200)
    $updated = Request 'PATCH' '/api/v1/notes/1' '{"content":"Changed body"}' 200 | ConvertFrom-Json
    if ($updated.content -ne 'Changed body' -or -not $updated.updatedAt) { throw 'PATCH result mismatch' }
    [void](Request 'POST' '/api/v1/notes' '{"title":" "}' 400)
    [void](Request 'GET' '/api/v1/notes/abc' '' 400)
    [void](Request 'GET' '/api/v1/notes/99' '' 404)
    [void](Request 'DELETE' '/api/v1/notes/1' '' 204)
    [void](Request 'GET' '/api/v1/notes/1' '' 404)
    $docsPath = Join-Path $sessionDir 'ui.html'
    $docsStatus = & curl.exe --noproxy '*' -sS --max-time 5 -o $docsPath -w '%{http_code}' "http://127.0.0.1:$port/docs/index.html"
    if ($LASTEXITCODE -ne 0 -or $docsStatus -ne '200' -or -not ([IO.File]::ReadAllText($docsPath).Contains('SwaggerUIBundle'))) { throw 'Swagger UI unavailable' }
    Write-Host "GET /docs/index.html -> $docsStatus (Swagger UI HTML)"
    $spec = Request 'GET' '/docs/doc.json' '' 200 | ConvertFrom-Json
    if ($spec.swagger -ne '2.0' -or $spec.basePath -ne '/api/v1') { throw 'Unexpected Swagger version/basePath' }
    $operations = @('/notes:get','/notes:post','/notes/{id}:get','/notes/{id}:patch','/notes/{id}:delete')
    foreach ($operation in $operations) {
        $parts = $operation.Split(':')
        if (-not $spec.paths.($parts[0]).($parts[1])) { throw "Missing operation $operation" }
    }
    if ($spec.paths.'/notes/{id}'.delete.responses.'204'.schema) { throw '204 must not declare a schema' }
    if ($spec.paths.'/notes'.get.parameters) { throw 'List must not advertise pagination' }
    Write-Host 'Specification: Swagger 2.0, basePath /api/v1, all 5 operations, 204 without schema.'
    Write-Host 'PZ12 scenario passed.'
    if ($KeepOpen) {
        Write-Host "Swagger UI: http://127.0.0.1:$port/docs/index.html"
        [void](Read-Host 'Open this URL, capture the real UI, then press Enter to stop this server')
    }
}
finally {
    if ($server -and -not $server.HasExited) { Stop-Process -Id $server.Id -Force; $server.WaitForExit() }
    if (-not $ready -and (Test-Path -LiteralPath (Join-Path $sessionDir 'server-err.txt'))) { Write-Host ([System.IO.File]::ReadAllText((Join-Path $sessionDir 'server-err.txt'))) }
    $env:PORT = $oldPort
    if (Test-Path -LiteralPath $sessionDir) {
        $resolved = (Resolve-Path -LiteralPath $sessionDir).Path
        $tempRoot = (Resolve-Path -LiteralPath $env:TEMP).Path
        if (-not $resolved.StartsWith($tempRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Unsafe cleanup path' }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}

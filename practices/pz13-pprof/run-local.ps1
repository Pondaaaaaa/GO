param([switch]$OpenUI, [string]$OutputDirectory = '')

$ErrorActionPreference = 'Stop'
$moduleRoot = $PSScriptRoot
$repoRoot = (Resolve-Path (Join-Path $moduleRoot '..\..')).Path
& (Join-Path $repoRoot 'setup.ps1')
$oldPath = $env:PATH
$oldProcs = $env:GOMAXPROCS
$env:GOMAXPROCS = '1'
$sessionDir = Join-Path $env:TEMP ('pz13-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $sessionDir | Out-Null
$ownedProcesses = [Collections.Generic.List[System.Diagnostics.Process]]::new()
$dot = Get-Command dot -ErrorAction SilentlyContinue
if (-not $dot) {
    $localDot = Join-Path $repoRoot '..\.local\graphviz-16.1.0'
    if (Test-Path -LiteralPath $localDot) { $dot = Get-ChildItem -LiteralPath $localDot -Filter dot.exe -Recurse | Select-Object -First 1 }
}
if ($dot) {
    $dotPath = if ($dot.Source) { $dot.Source } else { $dot.FullName }
    $env:PATH = (Split-Path -Parent $dotPath) + ';' + $env:PATH
}

function RunGo([string[]]$Arguments, [string]$LogName, [bool]$AllowNoMatches = $false) {
    $errorFile = Join-Path $sessionDir 'analysis-error.txt'
    $savedErrorAction = $ErrorActionPreference
    try {
        # PowerShell 5.1 turns native stderr into ErrorRecords; inspect it below.
        $ErrorActionPreference = 'Continue'
        $output = & go @Arguments 2> $errorFile
        $exitCode = $LASTEXITCODE
    }
    finally { $ErrorActionPreference = $savedErrorAction }
    $errorText = [IO.File]::ReadAllText($errorFile)
    if ($exitCode -ne 0) {
        if ($AllowNoMatches -and $errorText.Contains('no matches found for regexp: Fib')) { $output = @('no matches found for regexp: Fib (too few CPU samples to list FibFast)') }
        else { throw "go analysis failed: $errorText" }
    }
    # Windows pprof synthesizes a build identifier containing the temp path.
    $output = $output | ForEach-Object { ([string]$_).Replace($sessionDir, '<temporary-pz13-dir>') }
    $output | Out-File -LiteralPath (Join-Path $sessionDir $LogName) -Encoding utf8
    $output | ForEach-Object { Write-Host $_ }
}

function GetHTTP([string]$Url, [string]$File, [string]$Contains = '') {
    $status = & curl.exe --noproxy '*' -sS --connect-timeout 2 --max-time 10 -o $File -w '%{http_code}' $Url
    if ($LASTEXITCODE -ne 0 -or $status -ne '200') { throw "HTTP failed: $Url status=$status" }
    if ($Contains -and -not ([IO.File]::ReadAllText($File).Contains($Contains))) { throw "Unexpected body: $Url" }
    Write-Host "GET $(([uri]$Url).PathAndQuery) -> $status"
}

try {
    Push-Location $moduleRoot
    try {
        & go build -o (Join-Path $sessionDir 'load.exe') ./cmd/load
        if ($LASTEXITCODE -ne 0) { throw 'load build failed' }
        foreach ($mode in @('slow', 'fast')) {
            Write-Host "Building and measuring mode=$mode n=38 GOMAXPROCS=1"
            & go build -ldflags "-X main.defaultMode=$mode" -o (Join-Path $sessionDir "$mode.exe") ./cmd/api
            if ($LASTEXITCODE -ne 0) { throw 'API build failed' }
            $serverLog = Join-Path $sessionDir "$mode-timers.txt"
            $server = Start-Process -FilePath (Join-Path $sessionDir "$mode.exe") -ArgumentList '-port 0 -n 38' -PassThru -WindowStyle Hidden -RedirectStandardError $serverLog -RedirectStandardOutput (Join-Path $sessionDir "$mode-out.txt")
            $ownedProcesses.Add($server)
            $port = 0
            for ($i = 0; $i -lt 100; $i++) {
                $text = Get-Content -LiteralPath $serverLog -Raw -ErrorAction SilentlyContinue
                if ($text -match 'listening=127\.0\.0\.1:(\d+)') { $port = [int]$Matches[1]; break }
                if ($server.HasExited) { throw 'API exited early' }
                Start-Sleep -Milliseconds 100
            }
            if (-not $port) { throw 'No API listening address' }
            $base = "http://127.0.0.1:$port"
            GetHTTP "$base/health" (Join-Path $sessionDir 'health.txt') 'ok'
            GetHTTP "$base/work" (Join-Path $sessionDir 'work.txt') '39088169'
            GetHTTP "$base/debug/pprof/" (Join-Path $sessionDir 'index.html') 'Types of profiles available'
            GetHTTP "$base/debug/pprof/goroutine?debug=1" (Join-Path $sessionDir "$mode-goroutine.txt") 'goroutine profile'

            $cpu = Join-Path $sessionDir "$mode-cpu.pprof"
            $profileRequest = Start-Process -FilePath (Get-Command curl.exe).Source -ArgumentList "--noproxy * -f -sS --max-time 20 -o `"$cpu`" $base/debug/pprof/profile?seconds=12" -WindowStyle Hidden -PassThru -RedirectStandardError (Join-Path $sessionDir "$mode-profile-error.txt")
            $ownedProcesses.Add($profileRequest)
            # Cache the process handle before exit (Windows PowerShell 5.1).
            $null = $profileRequest.Handle
            Start-Sleep -Milliseconds 250
            $load = & (Join-Path $sessionDir 'load.exe') -url "$base/work" -count 24 -interval 500ms
            if ($LASTEXITCODE -ne 0) { throw 'load failed' }
            $load | Out-File (Join-Path $sessionDir "$mode-load.txt") -Encoding utf8
            Write-Host $load
            if (-not $profileRequest.WaitForExit(22000)) { throw 'CPU profile request timed out' }
            if ($profileRequest.ExitCode -ne 0 -or -not (Test-Path -LiteralPath $cpu)) {
                Write-Host "profile exit=$($profileRequest.ExitCode) file_exists=$(Test-Path -LiteralPath $cpu)"
                Get-Content -LiteralPath (Join-Path $sessionDir "$mode-profile-error.txt") | ForEach-Object { Write-Host $_ }
                throw 'CPU profile collection failed'
            }

            # Heap work occurs after CPU collection, identical in both variants.
            GetHTTP "$base/heap-work" (Join-Path $sessionDir 'heap-work.txt') 'retained_bytes=4194304'
            GetHTTP "$base/debug/pprof/heap?gc=1" (Join-Path $sessionDir "$mode-heap.pprof")
            Stop-Process -Id $server.Id -Force
            $server.WaitForExit()
            RunGo @('tool','pprof','-top',(Join-Path $sessionDir "$mode.exe"),$cpu) "$mode-cpu-top.txt"
            RunGo @('tool','pprof','-list=Fib',(Join-Path $sessionDir "$mode.exe"),$cpu) "$mode-cpu-list.txt" ($mode -eq 'fast')
            foreach ($sample in @('inuse_space','alloc_space')) {
                RunGo @('tool','pprof',"-sample_index=$sample",'-top',(Join-Path $sessionDir "$mode.exe"),(Join-Path $sessionDir "$mode-heap.pprof")) "$mode-heap-$sample.txt"
            }
            if ($dot) {
                & go tool pprof -svg -output (Join-Path $sessionDir "$mode-cpu.svg") (Join-Path $sessionDir "$mode.exe") $cpu
                if ($LASTEXITCODE -ne 0) { throw 'Graph rendering failed' }
                Write-Host "Real pprof graph saved: $mode-cpu.svg"
            } else { Write-Warning 'Graphviz dot unavailable: graph not generated; top/list and other measurements remain valid.' }
            Get-Content -LiteralPath $serverLog | ForEach-Object { Write-Host $_ }
        }
        RunGo @('test','-bench=.','-benchmem','-count=3','./...') 'benchmarks.txt'
        if ($OutputDirectory) {
            $outputRoot = [IO.Path]::GetFullPath($OutputDirectory)
            New-Item -ItemType Directory -Force -Path $outputRoot | Out-Null
            foreach ($name in @('slow-timers.txt','fast-timers.txt','slow-load.txt','fast-load.txt','slow-cpu-top.txt','fast-cpu-top.txt','slow-cpu-list.txt','fast-cpu-list.txt','slow-heap-inuse_space.txt','fast-heap-inuse_space.txt','slow-heap-alloc_space.txt','fast-heap-alloc_space.txt','benchmarks.txt','slow-cpu.svg','fast-cpu.svg')) {
                $file = Join-Path $sessionDir $name
                if (Test-Path -LiteralPath $file) { Copy-Item -LiteralPath $file -Destination (Join-Path $outputRoot $name) }
            }
        }
        if ($OpenUI) {
            & go build -o (Join-Path $sessionDir 'pprof.exe') cmd/pprof
            if ($LASTEXITCODE -ne 0) { throw 'pprof UI build failed' }
            $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0)
            $listener.Start(); $uiPort = $listener.LocalEndpoint.Port; $listener.Stop()
            $ui = Start-Process -FilePath (Join-Path $sessionDir 'pprof.exe') -ArgumentList "-http=127.0.0.1:$uiPort -no_browser `"$(Join-Path $sessionDir 'slow.exe')`" `"$(Join-Path $sessionDir 'slow-cpu.pprof')`"" -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $sessionDir 'ui-out.txt') -RedirectStandardError (Join-Path $sessionDir 'ui-err.txt')
            $ownedProcesses.Add($ui)
            Write-Host "Live pprof UI: http://127.0.0.1:$uiPort/ui/"
            [void](Read-Host 'Open the URL and capture the real graph; press Enter to stop the UI and delete profiles')
        }
        Write-Host 'PZ13 scenario passed. Binary profiles remain temporary and will be deleted.'
    }
    finally { Pop-Location }
}
finally {
    foreach ($process in $ownedProcesses) { if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force; $process.WaitForExit() } }
    $env:GOMAXPROCS = $oldProcs
    $env:PATH = $oldPath
    if (Test-Path -LiteralPath $sessionDir) {
        $resolved = (Resolve-Path -LiteralPath $sessionDir).Path
        $tempRoot = (Resolve-Path -LiteralPath $env:TEMP).Path
        if (-not $resolved.StartsWith($tempRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase) -or (Split-Path -Leaf $resolved) -notlike 'pz13-*') { throw 'Unsafe cleanup path' }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}

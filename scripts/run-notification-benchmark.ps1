param(
  [ValidateSet('sync', 'outbox', 'both')]
  [string]$Mode = 'both',
  [int]$VUs = 20,
  [string]$Duration = '30s',
  [string[]]$Actions = @('like', 'favorite', 'comment'),
  [string]$BaseURL = 'http://localhost:18080',
  [string]$HTTPAddr = ':18080',
  [string]$MCPAddr = ':18081',
  [string]$MySQLDSN = 'root:root@tcp(localhost:3306)/aidevclub?charset=utf8mb4&parseTime=True&loc=Local',
  [string]$RedisAddr = 'localhost:16379',
  [string]$JWTSecret = 'local-notification-benchmark-secret-2026',
  [string]$RabbitMQURL = 'amqp://notification:notification@localhost:5672/',
  [string]$OutputDirectory = 'docs/notification-benchmark-runs'
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

if ($VUs -lt 1 -or $VUs -gt 500) { throw 'VUs must be between 1 and 500.' }
if ($Actions.Count -eq 0 -or @($Actions | Where-Object { $_ -notin @('like', 'favorite', 'comment') }).Count -gt 0) {
  throw 'Actions must be selected from like, favorite, comment.'
}

$k6Command = Get-Command k6 -ErrorAction Stop
$timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$runId = [guid]::NewGuid().ToString('N').Substring(0, 8)
$outputPath = Join-Path $repoRoot (Join-Path $OutputDirectory "$timestamp-$runId")
$tempPath = Join-Path ([System.IO.Path]::GetTempPath()) "aidevclub-notification-$runId"
New-Item -ItemType Directory -Path $outputPath -Force | Out-Null
New-Item -ItemType Directory -Path $tempPath -Force | Out-Null

$envNames = @(
  'AIDEVCLUB_MYSQL_DSN', 'AIDEVCLUB_REDIS_ADDR', 'AIDEVCLUB_JWT_SECRET',
  'AIDEVCLUB_NOTIFICATION_MODE', 'AIDEVCLUB_NOTIFICATION_RABBITMQ_URL',
  'AIDEVCLUB_HTTP_ADDR', 'AIDEVCLUB_MCP_ADDR', 'AIDEVCLUB_RATELIMIT_PER_MINUTE',
  'BASE_URL', 'ACTION', 'DATA_FILE', 'VUS', 'DURATION'
)
$previousEnv = @{}
foreach ($name in $envNames) { $previousEnv[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
$serverProcess = $null

function Write-JsonFile([string]$Path, $Value) {
  $Value | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $Path -Encoding utf8
}

function Wait-Infra {
  $deadline = (Get-Date).AddMinutes(3)
  while ((Get-Date) -lt $deadline) {
    $mysqlId = (docker compose ps -q mysql).Trim()
    $redisId = (docker compose ps -q redis).Trim()
    $rabbitId = (docker compose ps -q rabbitmq).Trim()
    $mysqlReady = $false
    if ($mysqlId) {
      $null = docker compose exec -T mysql mysqladmin ping -h localhost --silent 2>$null
      $mysqlReady = $LASTEXITCODE -eq 0
    }
    $redisReady = $redisId -and ((docker compose exec -T redis redis-cli ping 2>$null) -match 'PONG')
    $rabbitReady = $false
    if ($rabbitId) {
      $health = docker inspect --format '{{.State.Health.Status}}' $rabbitId 2>$null
      $rabbitReady = $health -eq 'healthy'
    }
    if ($mysqlReady -and $redisReady -and $rabbitReady) { return }
    Start-Sleep -Seconds 2
  }
  throw 'MySQL, Redis and RabbitMQ did not become healthy within three minutes.'
}

function Wait-HttpHealth([int]$ProcessId) {
  $deadline = (Get-Date).AddMinutes(2)
  while ((Get-Date) -lt $deadline) {
    $process = Get-Process -Id $ProcessId -ErrorAction SilentlyContinue
    if (-not $process) { throw "Backend process $ProcessId exited before becoming healthy." }
    try {
      $health = Invoke-RestMethod -Uri "$BaseURL/healthz" -TimeoutSec 2
      if ($health.status -eq 'ok') { return }
    } catch { }
    Start-Sleep -Milliseconds 500
  }
  throw 'Backend did not become healthy within two minutes.'
}

function Wait-OutboxDrain([uint32]$ArticleId) {
  $sql = @"
SELECT
  (SELECT COUNT(*) FROM notification_outbox_events
   WHERE JSON_UNQUOTE(JSON_EXTRACT(payload, '$.notification.resource_id')) = '$ArticleId'
     AND status IN ('pending', 'publishing'))
  +
  (SELECT COUNT(*) FROM notification_outbox_events e
   LEFT JOIN notifications n ON n.event_id = JSON_UNQUOTE(JSON_EXTRACT(e.payload, '$.event_id'))
   WHERE JSON_UNQUOTE(JSON_EXTRACT(e.payload, '$.notification.resource_id')) = '$ArticleId'
     AND e.status = 'published' AND n.id IS NULL);
"@
  $deadline = (Get-Date).AddMinutes(2)
  while ((Get-Date) -lt $deadline) {
    $remaining = docker compose exec -T mysql mysql -uroot -proot --batch --skip-column-names aidevclub -e $sql 2>$null
    if ($LASTEXITCODE -eq 0 -and [int]($remaining | Select-Object -Last 1) -eq 0) { return }
    Start-Sleep -Seconds 1
  }
  throw "Outbox events for article $ArticleId did not drain within two minutes."
}

function Invoke-ActionBenchmark([string]$CurrentMode, [string]$Action) {
  $label = "$($CurrentMode.Substring(0, 1))-$($Action.Substring(0, 1))-$runId"
  $dataPath = Join-Path $tempPath "$label.json"
  $seedOutput = & go run ./cmd/notification-bench -users $VUs -label $label
  if ($LASTEXITCODE -ne 0) { throw "Failed to seed notification benchmark data for $CurrentMode/$Action." }
  [System.IO.File]::WriteAllText($dataPath, ($seedOutput -join "`n"), [System.Text.UTF8Encoding]::new($false))
  $testData = Get-Content -Raw -LiteralPath $dataPath | ConvertFrom-Json

  $prefix = "$CurrentMode-$Action"
  $summaryPath = Join-Path $outputPath "$prefix-k6-summary.json"
  $stdoutPath = Join-Path $outputPath "$prefix-k6-output.txt"
  $stderrPath = Join-Path $outputPath "$prefix-k6-error.txt"
  $resourcePath = Join-Path $outputPath "$prefix-resource-samples.jsonl"
  $runMetadataPath = Join-Path $outputPath "$prefix-run.json"

  $env:BASE_URL = $BaseURL
  $env:ACTION = $Action
  $env:DATA_FILE = $dataPath
  $env:VUS = "$VUs"
  $env:DURATION = $Duration

  $k6Arguments = @(
    'run', "--summary-export=$summaryPath", "--env=BASE_URL=$BaseURL",
    "--env=ACTION=$Action", "--env=DATA_FILE=$dataPath", "--env=VUS=$VUs",
    "--env=DURATION=$Duration", (Join-Path $repoRoot 'scripts/notification-benchmark.js')
  )
  $k6Process = Start-Process -FilePath $k6Command.Source -ArgumentList $k6Arguments -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath
  $samples = [System.Collections.Generic.List[object]]::new()
  while (-not $k6Process.HasExited) {
    $proc = Get-Process -Id $serverProcess.Id -ErrorAction SilentlyContinue
    $containers = @()
    $ids = @(docker compose ps -q mysql redis rabbitmq | Where-Object { $_ })
    if ($ids.Count -gt 0) {
      $containerLines = docker stats --no-stream --format '{{json .}}' $ids 2>$null
      foreach ($line in $containerLines) {
        try {
          $container = $line | ConvertFrom-Json
          $containers += [pscustomobject]@{ name = $container.Name; cpu_percent = $container.CPUPerc; memory = $container.MemUsage }
        } catch { }
      }
    }
    if ($proc) {
      $sample = [pscustomobject]@{
        sampled_at = (Get-Date).ToString('o')
        app_cpu_seconds = [math]::Round($proc.CPU, 4)
        app_working_set_bytes = $proc.WorkingSet64
        containers = $containers
      }
      $samples.Add($sample)
      $sample | ConvertTo-Json -Depth 6 -Compress | Add-Content -LiteralPath $resourcePath -Encoding utf8
    }
    Start-Sleep -Seconds 1
    $k6Process.Refresh()
  }
  $k6Process.WaitForExit()
  if ($k6Process.ExitCode -ne 0) {
    $output = if (Test-Path $stdoutPath) { Get-Content -Raw $stdoutPath } else { '' }
    $errorOutput = if (Test-Path $stderrPath) { Get-Content -Raw $stderrPath } else { '' }
    throw "k6 failed for $CurrentMode/$Action (exit $($k6Process.ExitCode)).`n$output`n$errorOutput"
  }
  if ($CurrentMode -eq 'outbox') { Wait-OutboxDrain -ArticleId ([uint32]$testData.article_id) }

  $cpuSeconds = 0.0
  $wallSeconds = 0.0
  if ($samples.Count -gt 1) {
    $cpuSeconds = [math]::Max(0, $samples[$samples.Count - 1].app_cpu_seconds - $samples[0].app_cpu_seconds)
    $firstSampleTime = [datetime]$samples[0].sampled_at
    $lastSampleTime = [datetime]$samples[$samples.Count - 1].sampled_at
    $wallSeconds = [math]::Max(1, ($lastSampleTime - $firstSampleTime).TotalSeconds)
  }
  $hostCores = [int](Get-CimInstance Win32_ComputerSystem).NumberOfLogicalProcessors
  $appCpuMachinePercent = if ($wallSeconds -gt 0 -and $hostCores -gt 0) { 100 * $cpuSeconds / ($wallSeconds * $hostCores) } else { 0 }
  $maxAppMemory = if ($samples.Count -gt 0) { ($samples | Measure-Object -Property app_working_set_bytes -Maximum).Maximum } else { 0 }

  $metadata = [ordered]@{
    mode = $CurrentMode; action = $Action; article_id = $testData.article_id
    users = $VUs; concurrency = $VUs; duration = $Duration
    auth = 'dedicated local JWT access tokens; one user per VU; same secret and middleware in each mode'
    k6_version = (& $k6Command.Source version | Select-Object -First 1)
    application_average_cpu_percent_of_machine = [math]::Round($appCpuMachinePercent, 2)
    application_peak_working_set_bytes = $maxAppMemory
    sample_count = $samples.Count
    summary_file = [System.IO.Path]::GetFileName($summaryPath)
    resource_samples_file = [System.IO.Path]::GetFileName($resourcePath)
  }
  Write-JsonFile $runMetadataPath $metadata
  Remove-Item -LiteralPath $dataPath -Force
  Write-Host "Completed $CurrentMode/$Action. k6 summary: $summaryPath"
}

try {
  $env:AIDEVCLUB_MYSQL_DSN = $MySQLDSN
  $env:AIDEVCLUB_REDIS_ADDR = $RedisAddr
  $env:AIDEVCLUB_JWT_SECRET = $JWTSecret
  $env:AIDEVCLUB_NOTIFICATION_RABBITMQ_URL = $RabbitMQURL
  $env:AIDEVCLUB_HTTP_ADDR = $HTTPAddr
  $env:AIDEVCLUB_MCP_ADDR = $MCPAddr
  $env:AIDEVCLUB_RATELIMIT_PER_MINUTE = '10000'

  docker compose up -d mysql redis rabbitmq
  if ($LASTEXITCODE -ne 0) { throw 'docker compose up failed.' }
  Wait-Infra

  $httpPort = [int]($HTTPAddr.Split(':')[-1])
  $mcpPort = [int]($MCPAddr.Split(':')[-1])
  $occupiedPorts = @(Get-NetTCPConnection -LocalPort $httpPort, $mcpPort -State Listen -ErrorAction SilentlyContinue)
  if ($occupiedPorts.Count -gt 0) {
    $occupiedPorts | Select-Object LocalPort, OwningProcess | Format-Table | Out-String | Write-Host
    throw "A benchmark server port is already in use ($httpPort or $mcpPort). Choose unused HTTPAddr/MCPAddr ports."
  }

  $computer = Get-CimInstance Win32_ComputerSystem
  $processor = Get-CimInstance Win32_Processor | Select-Object -First 1
  $os = Get-CimInstance Win32_OperatingSystem
  $dockerServerVersion = docker version --format '{{.Server.Version}}'
  $machine = [ordered]@{
    captured_at = (Get-Date).ToString('o')
    os = $os.Caption
    os_version = $os.Version
    architecture = $env:PROCESSOR_ARCHITECTURE
    cpu = $processor.Name
    logical_processors = $computer.NumberOfLogicalProcessors
    physical_memory_bytes = [int64]$computer.TotalPhysicalMemory
    docker_engine_version = $dockerServerVersion
    mysql_image = (docker inspect --format '{{.Config.Image}}' (docker compose ps -q mysql))
    redis_image = (docker inspect --format '{{.Config.Image}}' (docker compose ps -q redis))
    rabbitmq_image = (docker inspect --format '{{.Config.Image}}' (docker compose ps -q rabbitmq))
    k6_version = (& $k6Command.Source version | Select-Object -First 1)
    vus = $VUs; duration = $Duration; actions = $Actions
  }
  Write-JsonFile (Join-Path $outputPath 'machine.json') $machine

  $buildDirectory = Join-Path ([System.IO.Path]::GetTempPath()) "aidevclub-notification-bin-$runId"
  New-Item -ItemType Directory -Path $buildDirectory -Force | Out-Null
  $serverBinary = Join-Path $buildDirectory 'server.exe'
  & go build -o $serverBinary ./cmd/server
  if ($LASTEXITCODE -ne 0) { throw 'go build ./cmd/server failed.' }

  $modes = if ($Mode -eq 'both') { @('sync', 'outbox') } else { @($Mode) }
  foreach ($currentMode in $modes) {
    $env:AIDEVCLUB_NOTIFICATION_MODE = $currentMode
    $serverLog = Join-Path $outputPath "$currentMode-server-output.log"
    $serverError = Join-Path $outputPath "$currentMode-server-error.log"
    $serverProcess = Start-Process -FilePath $serverBinary -PassThru -WindowStyle Hidden -RedirectStandardOutput $serverLog -RedirectStandardError $serverError
    Wait-HttpHealth -ProcessId $serverProcess.Id
    foreach ($action in $Actions) { Invoke-ActionBenchmark -CurrentMode $currentMode -Action $action }
    Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue
    $serverProcess = $null
  }

  Write-Host "Benchmark artifacts saved under $outputPath"
} finally {
  if ($serverProcess) { Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue }
  foreach ($name in $envNames) { [Environment]::SetEnvironmentVariable($name, $previousEnv[$name], 'Process') }
}

# build-dist.ps1 — 构建便携包:dist\weixin-key-portable\
# 只含可执行文件、说明、策略库;绝不含配置、密钥、聊天、.build 或本机路径。
param([string]$Out = "")
$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repo = Resolve-Path "$scriptDir\.."
if (-not $Out) { $Out = "$repo\dist\weixin-key-portable" }
New-Item -ItemType Directory -Force $Out | Out-Null
$Out = (Resolve-Path $Out).Path

Push-Location $repo
try {
    go build -trimpath -o "$Out\weixin-key.exe" .\cmd\weixin-key
    if ($LASTEXITCODE -ne 0) { throw "build weixin-key failed" }
    go build -trimpath -o "$Out\material-probe.exe" .\cmd\material-probe-diagnostic
    if ($LASTEXITCODE -ne 0) { throw "build material-probe failed" }
    go build -trimpath -o "$Out\imgkey-probe.exe" .\cmd\imgkey-probe
    if ($LASTEXITCODE -ne 0) { throw "build imgkey-probe failed" }
} finally { Pop-Location }

Copy-Item "$repo\README.md" "$Out\"
New-Item -ItemType Directory -Force "$Out\docs" | Out-Null
Copy-Item "$repo\docs\AGENT_WORKFLOW.md" "$Out\docs\" -ErrorAction SilentlyContinue
Copy-Item "$repo\scripts\observe.ps1" "$Out\"
Copy-Item "$repo\strategies" "$Out\strategies" -Recurse
Copy-Item "$repo\skills" "$Out\skills" -Recurse

# 防呆:便携包内不允许出现配置/密钥/数据文件
$forbidden = Get-ChildItem $Out -Recurse -File | Where-Object {
    $_.Name -match 'config.*\.json$|\.db$|\.dat$|passphrase|secret' -and $_.FullName -notmatch 'strategies\\'
}
if ($forbidden) { throw "forbidden files in dist: $($forbidden.FullName -join ', ')" }

$manifest = Get-ChildItem $Out -Recurse -File | ForEach-Object {
    [ordered]@{ path = $_.FullName.Substring($Out.Length + 1); sha256 = (Get-FileHash $_.FullName -Algorithm SHA256).Hash }
}
$manifest | ConvertTo-Json -Depth 3 | Set-Content "$Out\SHA256SUMS.json" -Encoding UTF8
Write-Host "dist OK: $Out ($($manifest.Count) files)" -ForegroundColor Green

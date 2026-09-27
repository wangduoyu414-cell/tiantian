# capture-key.ps1 — 有界被动取钥 + 缓存写入 + 离线复核(SKILL.md 第 4 步)
# 全部参数显式传入;任何一个验证门不过都非零退出。
param(
    [Parameter(Mandatory)] [string]$ProbeExe,      # material-probe.exe
    [Parameter(Mandatory)] [string]$Dll,           # Weixin.dll 完整路径
    [Parameter(Mandatory)] [string]$DllSha256,
    [Parameter(Mandatory)] [int]   $MainPid,
    [Parameter(Mandatory)] [string]$Created,       # RFC3339Nano
    [Parameter(Mandatory)] [string]$AccountRoot,   # 账号根(含 db_storage)
    [Parameter(Mandatory)] [string]$Exe,           # Weixin.exe
    [Parameter(Mandatory)] [string]$ConfigPath,
    [Parameter(Mandatory)] [string]$ScratchDir,    # 已存在、空、源目录之外
    [string]$PrimaryDb = "message\message_0.db"
)
$ErrorActionPreference = "Stop"

if (-not (Test-Path $ScratchDir)) { throw "scratch dir must exist: $ScratchDir" }
$cfgHash = (Get-FileHash -LiteralPath $ConfigPath -Algorithm SHA256).Hash

Write-Host "== live bounded capture ==" -ForegroundColor Cyan
& $ProbeExe --module $Dll --sha256 $DllSha256 --pid $MainPid --created $Created `
    --root $AccountRoot --exe $Exe --primary-db $PrimaryDb `
    --snapshot-check --scratch $ScratchDir `
    --cache-verified --cache-config $ConfigPath --cache-config-sha256 $cfgHash
if ($LASTEXITCODE -ne 0) { throw "capture failed (exit $LASTEXITCODE)" }

# Gate: independent offline cache verification with the NEW hash.
$newHash = (Get-FileHash -LiteralPath $ConfigPath -Algorithm SHA256).Hash
if ($newHash -eq $cfgHash) { throw "config hash unchanged after capture; cache write did not happen" }

Write-Host "== offline verify-cache gate ==" -ForegroundColor Cyan
& $ProbeExe --verify-cache --root $AccountRoot --cache-config $ConfigPath --cache-config-sha256 $newHash
if ($LASTEXITCODE -ne 0) { throw "verify-cache failed" }
Write-Host "OK: cache verified offline" -ForegroundColor Green

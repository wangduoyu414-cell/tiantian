# ensure-imgkey.ps1 — 图片密钥:扫描 → 未驻留则提示用户 → 重扫(SKILL.md 第 5 步)
# 成功时把密钥放进调用方环境变量 WECHAT_CLI_IMGKEY_HEX(不落盘)。
param(
    [Parameter(Mandatory)] [string]$ImgkeyProbeExe, # imgkey-probe.exe
    [Parameter(Mandatory)] [int[]]  $Pids,          # 全部 Weixin.exe 进程
    [Parameter(Mandatory)] [string]$AccountRoot,    # 用于自动挑一个 _t.dat 锚点
    [int]$MaxRounds = 3
)
$ErrorActionPreference = "Stop"

$dat = Get-ChildItem (Join-Path $AccountRoot "msg\attach") -Recurse -Filter "*_t.dat" -ErrorAction SilentlyContinue |
       Select-Object -First 1 -ExpandProperty FullName
if (-not $dat) { throw "no _t.dat anchor found under $AccountRoot\msg\attach (account may have no images)" }

$pidArgs = @()
foreach ($p in $Pids) { $pidArgs += @("--pid", "$p") }

for ($round = 1; $round -le $MaxRounds; $round++) {
    $out = & $ImgkeyProbeExe @pidArgs --dat $dat
    if ($LASTEXITCODE -eq 0) {
        $j = $out | ConvertFrom-Json
        # 只暴露给当前会话;调用方在同一进程内读到
        [Environment]::SetEnvironmentVariable("WECHAT_CLI_IMGKEY_HEX", $j.key_hex, "Process")
        Write-Host "OK: image key resident, captured into session env (pid $($j.pid))" -ForegroundColor Green
        return
    }
    if ($round -lt $MaxRounds) {
        Write-Host "图片密钥尚未加载到内存。请在微信里随便点开 2-3 张聊天图片(看完即可),然后按回车继续..." -ForegroundColor Yellow
        Read-Host | Out-Null
    }
}
throw "image key still not resident after $MaxRounds rounds"

# export.ps1 — 导出 + 人话验收(SKILL.md 第 6、7 步)
param(
    [Parameter(Mandatory)] [string]$CliExe,        # weixin-key.exe
    [Parameter(Mandatory)] [string]$ConfigPath,
    [Parameter(Mandatory)] [string]$AccountRoot,
    [Parameter(Mandatory)] [string]$Account,
    [Parameter(Mandatory)] [string]$OutDir
)
$ErrorActionPreference = "Stop"

$env:WECHAT_CLI_CONFIG = $ConfigPath
$json = & $CliExe export --out $OutDir --db-root $AccountRoot --account $Account --pretty
$code = $LASTEXITCODE
# exit 0 = complete; exit 1 = partial(仍可能有完整可用结果,看对账)
$rep = $json | ConvertFrom-Json

$att = @{ copied = 0; decrypted = 0; missing = 0; "not-extracted" = 0 }
$jsonl = Join-Path $OutDir "data\messages.jsonl"
if (Test-Path $jsonl) {
    foreach ($line in [System.IO.File]::ReadLines($jsonl)) {
        foreach ($m in [regex]::Matches($line, '"status":"([a-z-]+)"')) {
            $k = $m.Groups[1].Value
            if ($att.ContainsKey($k)) { $att[$k]++ }
        }
    }
}

Write-Host ""
Write-Host "===== 导出对账 =====" -ForegroundColor Cyan
Write-Host ("消息总数: {0}   会话数: {1}" -f $rep.total_messages, $rep.total_conversations)
Write-Host ("图片已解密: {0}   语音/视频/文件已复制: {1}" -f $att.decrypted, $att.copied)
Write-Host ("源文件已被微信清理(无法恢复): {0}   表情未提取: {1}" -f $att.missing, $att."not-extracted")
Write-Host ("完整性: {0}   输出: {1}" -f $rep.status, $rep.out_dir)
if ($rep.status -ne "complete") {
    Write-Host "注意: status=partial —— 上面 missing/not-extracted 即为不完整部分,其余内容完整可用。" -ForegroundColor Yellow
}
exit $code

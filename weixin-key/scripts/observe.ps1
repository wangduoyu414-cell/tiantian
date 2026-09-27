# observe.ps1 — 新机微信环境只读观测
# 收集 material-probe / imgkey-probe / export 所需的全部现场参数。
# 不读聊天内容、不读密钥、不写任何文件。输出一个 JSON,供 agent 消费。
#
# 用法: powershell -NoProfile -File observe.ps1 [-StrategiesDir <path>]
param(
    [string]$StrategiesDir = "$PSScriptRoot\..\strategies"
)
$ErrorActionPreference = "SilentlyContinue"

$result = [ordered]@{
    schema_version = 1
    observed_at    = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ss.fffffffzzz")
    install        = $null
    processes      = @()
    accounts       = @()
    recommended    = $null
    strategy       = $null
    warnings       = @()
}

# --- 安装信息(注册表) ---
$uninstall = @(
    "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*"
)
foreach ($p in $uninstall) {
    Get-ItemProperty $p | Where-Object {
        ($_.InstallLocation -and $_.InstallLocation -match 'Weixin') -or
        ($_.DisplayName -and $_.DisplayName -match 'Weixin|WeChat') -or
        ($_.DisplayName -and $_.DisplayName -match "^\x5fae\x4fe1$")
    } | ForEach-Object {
        $loc = $_.InstallLocation.Trim('"')
        if ($loc -and (Test-Path "$loc\Weixin.exe")) {
            $result.install = [ordered]@{
                version = $_.DisplayVersion
                root    = $loc
                exe     = "$loc\Weixin.exe"
            }
        }
    }
}
if (-not $result.install) {
    # Fallback: common install roots
    foreach ($c in @("C:\Program Files\Tencent\Weixin", "D:\软件\weixin", "D:\Program Files\Tencent\Weixin")) {
        if (Test-Path "$c\Weixin.exe") {
            $ver = (Get-ChildItem $c -Directory | Where-Object { $_.Name -match '^\d+\.\d+' } | Sort-Object Name -Descending | Select-Object -First 1).Name
            $result.install = [ordered]@{ version = $ver; root = $c; exe = "$c\Weixin.exe" }
            break
        }
    }
}
if (-not $result.install) { $result.warnings += "weixin-not-in-registry" }

# --- 运行中的进程(主进程 = 无 Weixin 父进程的那个) ---
$all = Get-CimInstance Win32_Process | Where-Object { $_.Name -eq "Weixin.exe" }
$procList = @()
foreach ($p in $all) {
    $procList += [ordered]@{
        pid         = $p.ProcessId
        parent_pid  = $p.ParentProcessId
        created     = $p.CreationDate.ToString("yyyy-MM-ddTHH:mm:ss.fffffffzzz")
        exe         = $p.ExecutablePath
    }
}
$childPids = @($all | ForEach-Object { $_.ProcessId })
foreach ($p in $procList) {
    $p["is_main"] = -not ($childPids -contains $p.parent_pid)
}
$result.processes = $procList
if ($procList.Count -eq 0) { $result.warnings += "weixin-not-running" }

# --- Weixin.dll 定位与哈希 ---
$dll = $null
if ($result.install) {
    $verDir = Join-Path $result.install.root $result.install.version
    $candidate = Join-Path $verDir "Weixin.dll"
    if (Test-Path $candidate) { $dll = $candidate }
}
if (-not $dll -and $result.install) {
    $dll = Get-ChildItem (Join-Path $result.install.root "*\Weixin.dll") | Select-Object -First 1 -ExpandProperty FullName
}
if ($dll) {
    $result.install.dll = $dll
    $result.install.dll_sha256 = (Get-FileHash -LiteralPath $dll -Algorithm SHA256).Hash
} else {
    $result.warnings += "weixin-dll-not-found"
}

# --- 版本策略匹配 ---
if (Test-Path $StrategiesDir) {
    $stratFiles = Get-ChildItem $StrategiesDir -Filter "weixin-*.json"
    foreach ($f in $stratFiles) {
        $s = Get-Content $f.FullName -Raw | ConvertFrom-Json
        if ($s.module_sha256 -and $result.install.dll_sha256 -and
            ($s.module_sha256.ToUpper() -eq $result.install.dll_sha256.ToUpper())) {
            $result.strategy = [ordered]@{
                status = "known-version"
                file   = $f.Name
                notes  = $s.notes
            }
        }
    }
    if (-not $result.strategy) {
        $result.strategy = [ordered]@{
            status = "unknown-version"
            action = "run material-probe --inspect first; capture is bounded and safe to attempt, but expect possible no-match"
        }
    }
}

# --- 账号目录发现(自定义缓存路径 + 默认 Documents) ---
$roots = @()
foreach ($drive in (Get-PSDrive -PSProvider FileSystem).Root) {
    $roots += Get-ChildItem $drive -Directory -Filter "xwechat_files" -Recurse -Depth 3 -ErrorAction SilentlyContinue | Select-Object -ExpandProperty FullName
}
$accounts = @()
foreach ($r in ($roots | Sort-Object -Unique)) {
    Get-ChildItem $r -Directory | Where-Object { $_.Name -match '_' -and $_.Name -notin @('all_users','Backup') -and (Test-Path (Join-Path $_.FullName "db_storage")) } | ForEach-Object {
        $msgDir = Join-Path $_.FullName "db_storage\message"
        $lastWrite = (Get-ChildItem $msgDir -Filter "*.db-wal" -ErrorAction SilentlyContinue |
                      Sort-Object LastWriteTime -Descending | Select-Object -First 1).LastWriteTime
        if (-not $lastWrite) { $lastWrite = $_.LastWriteTime }
        $accounts += [ordered]@{
            account            = $_.Name
            root               = $_.FullName
            last_message_write = $lastWrite.ToString("yyyy-MM-ddTHH:mm:sszzz")
        }
    }
}
$result.accounts = @($accounts | Sort-Object { $_.last_message_write } -Descending)

# --- 推荐目标:最近有写入、且(若有主进程)目录被进程持有 ---
if ($result.accounts.Count -gt 0) {
    $result.recommended = $result.accounts[0]
    if ($result.accounts.Count -gt 1) {
        $result.warnings += "multiple-accounts-present-confirm-with-user"
    }
} else {
    $result.warnings += "no-account-dir-found"
}

$result | ConvertTo-Json -Depth 6

param(
    [Parameter(Mandatory = $true)]
    [string]$WeixinExe,

    [string]$CaptureDll = (Join-Path $PSScriptRoot 'wxkey_capture.dll'),

    [string]$WeixinKey = (Join-Path (Split-Path -Parent $PSScriptRoot) 'weixin-key.exe'),

    [ValidateRange(30, 300)]
    [int]$TimeoutSeconds = 120,

    [ValidateRange(5, 90)]
    [int]$HookReadyTimeoutSeconds = 45,

    [ValidateRange(0, 60)]
    [int]$UIClickTimeoutSeconds = 15,

    [switch]$KeepCaptureFile
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not ('SuspendedInjector' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;

public static class SuspendedInjector
{
    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    public struct STARTUPINFO
    {
        public uint cb;
        public string lpReserved;
        public string lpDesktop;
        public string lpTitle;
        public uint dwX;
        public uint dwY;
        public uint dwXSize;
        public uint dwYSize;
        public uint dwXCountChars;
        public uint dwYCountChars;
        public uint dwFillAttribute;
        public uint dwFlags;
        public short wShowWindow;
        public short cbReserved2;
        public IntPtr lpReserved2;
        public IntPtr hStdInput;
        public IntPtr hStdOutput;
        public IntPtr hStdError;
    }

    [StructLayout(LayoutKind.Sequential)]
    public struct PROCESS_INFORMATION
    {
        public IntPtr hProcess;
        public IntPtr hThread;
        public uint dwProcessId;
        public uint dwThreadId;
    }

    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern bool CreateProcessW(
        string applicationName, StringBuilder commandLine,
        IntPtr processAttributes, IntPtr threadAttributes,
        bool inheritHandles, uint creationFlags, IntPtr environment,
        string currentDirectory, ref STARTUPINFO startupInfo,
        out PROCESS_INFORMATION processInformation);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern IntPtr VirtualAllocEx(
        IntPtr process, IntPtr address, UIntPtr size, uint allocationType, uint protect);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool VirtualFreeEx(
        IntPtr process, IntPtr address, UIntPtr size, uint freeType);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool WriteProcessMemory(
        IntPtr process, IntPtr address, byte[] buffer, UIntPtr size, out UIntPtr written);

    [DllImport("kernel32.dll", CharSet = CharSet.Ansi, SetLastError = true)]
    private static extern IntPtr GetModuleHandleA(string moduleName);

    [DllImport("kernel32.dll", CharSet = CharSet.Ansi, SetLastError = true)]
    private static extern IntPtr GetProcAddress(IntPtr module, string procName);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern IntPtr CreateRemoteThread(
        IntPtr process, IntPtr threadAttributes, UIntPtr stackSize,
        IntPtr startAddress, IntPtr parameter, uint creationFlags, out uint threadId);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern uint WaitForSingleObject(IntPtr handle, uint milliseconds);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool GetExitCodeThread(IntPtr thread, out uint exitCode);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern uint ResumeThread(IntPtr thread);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool TerminateProcess(IntPtr process, uint exitCode);

    [DllImport("kernel32.dll")]
    private static extern bool CloseHandle(IntPtr handle);

    private static void Win32(bool ok, string operation)
    {
        if (!ok) throw new Win32Exception(Marshal.GetLastWin32Error(), operation);
    }

    public static uint LaunchAndInject(string exe, string dll)
    {
        const uint CREATE_SUSPENDED = 0x00000004;
        const uint MEM_COMMIT = 0x1000;
        const uint MEM_RESERVE = 0x2000;
        const uint MEM_RELEASE = 0x8000;
        const uint PAGE_READWRITE = 0x04;
        const uint WAIT_OBJECT_0 = 0x00000000;

        STARTUPINFO startup = new STARTUPINFO();
        startup.cb = (uint)Marshal.SizeOf(typeof(STARTUPINFO));
        PROCESS_INFORMATION process;
        StringBuilder command = new StringBuilder("\"" + exe + "\" --scene=desktop");
        Win32(CreateProcessW(exe, command, IntPtr.Zero, IntPtr.Zero, false,
                            CREATE_SUSPENDED, IntPtr.Zero,
                            System.IO.Path.GetDirectoryName(exe), ref startup, out process),
              "CreateProcessW");

        bool resumed = false;
        IntPtr remote = IntPtr.Zero;
        try
        {
            byte[] path = Encoding.Unicode.GetBytes(dll + "\0");
            remote = VirtualAllocEx(process.hProcess, IntPtr.Zero,
                                    (UIntPtr)path.Length,
                                    MEM_COMMIT | MEM_RESERVE, PAGE_READWRITE);
            Win32(remote != IntPtr.Zero, "VirtualAllocEx");
            UIntPtr written;
            Win32(WriteProcessMemory(process.hProcess, remote, path,
                                     (UIntPtr)path.Length, out written) &&
                  written.ToUInt64() == (ulong)path.Length,
                  "WriteProcessMemory");

            IntPtr kernel32 = GetModuleHandleA("kernel32.dll");
            Win32(kernel32 != IntPtr.Zero, "GetModuleHandleA(kernel32.dll)");
            IntPtr loadLibrary = GetProcAddress(kernel32, "LoadLibraryW");
            Win32(loadLibrary != IntPtr.Zero, "GetProcAddress(LoadLibraryW)");
            uint remoteThreadId;
            IntPtr remoteThread = CreateRemoteThread(process.hProcess, IntPtr.Zero,
                                                     UIntPtr.Zero, loadLibrary, remote,
                                                     0, out remoteThreadId);
            Win32(remoteThread != IntPtr.Zero, "CreateRemoteThread");
            try
            {
                Win32(WaitForSingleObject(remoteThread, 10000) == WAIT_OBJECT_0,
                      "WaitForSingleObject(LoadLibraryW)");
                uint moduleHandle;
                Win32(GetExitCodeThread(remoteThread, out moduleHandle) && moduleHandle != 0,
                      "LoadLibraryW(remote)");
            }
            finally
            {
                CloseHandle(remoteThread);
            }

            if (ResumeThread(process.hThread) == 0xffffffff)
                throw new Win32Exception(Marshal.GetLastWin32Error(), "ResumeThread");
            resumed = true;
            return process.dwProcessId;
        }
        finally
        {
            if (remote != IntPtr.Zero)
                VirtualFreeEx(process.hProcess, remote, UIntPtr.Zero, MEM_RELEASE);
            if (!resumed)
                TerminateProcess(process.hProcess, 1);
            CloseHandle(process.hThread);
            CloseHandle(process.hProcess);
        }
    }
}
'@
}

function Resolve-RequiredFile {
    param([string]$Path, [string]$Label)
    $resolved = (Resolve-Path -LiteralPath $Path -ErrorAction Stop).Path
    $item = Get-Item -LiteralPath $resolved -Force
    if (-not $item.PSIsContainer -and $item.Length -gt 0) {
        return $resolved
    }
    throw "$Label must be a non-empty regular file"
}

function Stop-WeChatProcesses {
    $names = @('Weixin', 'WeChat', 'WeChatAppEx')
    $deadline = [DateTime]::UtcNow.AddSeconds(20)
    $emptySamples = 0
    while ([DateTime]::UtcNow -lt $deadline) {
        $running = @(Get-Process -Name $names -ErrorAction SilentlyContinue)
        if ($running.Count -eq 0) {
            $emptySamples++
            if ($emptySamples -ge 5) {
                return
            }
        }
        else {
            $emptySamples = 0
            $running | Stop-Process -Force -ErrorAction SilentlyContinue
        }
        Start-Sleep -Milliseconds 200
    }
    $remaining = @(Get-Process -Name $names -ErrorAction SilentlyContinue)
    $ids = ($remaining | ForEach-Object { $_.Id }) -join ','
    throw "WeChat processes did not exit before injection (remaining PIDs: $ids)"
}

function New-PrivateCaptureWorkspace {
    $directory = Join-Path ([IO.Path]::GetTempPath()) ('wechat-cli-capture-' + [Guid]::NewGuid().ToString('N'))
    [IO.Directory]::CreateDirectory($directory) | Out-Null

    $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User
    $inheritance = [Security.AccessControl.InheritanceFlags]::ContainerInherit -bor
        [Security.AccessControl.InheritanceFlags]::ObjectInherit
    $directorySecurity = [Security.AccessControl.DirectorySecurity]::new()
    $directorySecurity.SetAccessRuleProtection($true, $false)
    $directorySecurity.AddAccessRule([Security.AccessControl.FileSystemAccessRule]::new(
        $sid,
        [Security.AccessControl.FileSystemRights]::FullControl,
        $inheritance,
        [Security.AccessControl.PropagationFlags]::None,
        [Security.AccessControl.AccessControlType]::Allow))
    [IO.Directory]::SetAccessControl($directory, $directorySecurity)

    $capture = Join-Path $directory 'candidates.txt'
    $stream = [IO.FileStream]::new(
        $capture,
        [IO.FileMode]::CreateNew,
        [IO.FileAccess]::ReadWrite,
        [IO.FileShare]::Read)
    $stream.Dispose()

    $fileSecurity = [Security.AccessControl.FileSecurity]::new()
    $fileSecurity.SetAccessRuleProtection($true, $false)
    $fileSecurity.AddAccessRule([Security.AccessControl.FileSystemAccessRule]::new(
        $sid,
        [Security.AccessControl.FileSystemRights]::FullControl,
        [Security.AccessControl.AccessControlType]::Allow))
    [IO.File]::SetAccessControl($capture, $fileSecurity)

    return [pscustomobject]@{
        Directory = $directory
        Capture = $capture
        Stdout = (Join-Path $directory 'cli.stdout')
        Stderr = (Join-Path $directory 'cli.stderr')
    }
}

function Get-CaptureCandidateCount {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return 0
    }
    $info = Get-Item -LiteralPath $Path -Force
    if ($info.Length -gt 1MB) {
        throw 'capture file exceeded the 1 MiB safety limit'
    }

    $keys = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    $stream = $null
    $reader = $null
    try {
        $stream = [IO.FileStream]::new(
            $Path,
            [IO.FileMode]::Open,
            [IO.FileAccess]::Read,
            [IO.FileShare]::ReadWrite -bor [IO.FileShare]::Delete)
        $reader = [IO.StreamReader]::new($stream)
        while (-not $reader.EndOfStream) {
            $line = $reader.ReadLine()
            if ($null -ne $line) {
                $candidate = $line.Trim()
                if ($candidate -cmatch '^[0-9a-fA-F]{64}$') {
                    [void]$keys.Add($candidate)
                }
            }
        }
    }
    catch [IO.IOException] {
        return 0
    }
    finally {
        if ($null -ne $reader) { $reader.Dispose() }
        elseif ($null -ne $stream) { $stream.Dispose() }
    }
    if ($keys.Count -gt 64) {
        throw 'capture file exceeded the 64-candidate safety limit'
    }
    return $keys.Count
}

function Invoke-EnterWeChat {
    param([int]$TimeoutSeconds)
    if ($TimeoutSeconds -le 0) {
        return $false
    }
    Add-Type -AssemblyName UIAutomationClient
    Add-Type -AssemblyName UIAutomationTypes
    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $allowedPids = @(Get-Process -Name Weixin, WeChatAppEx -ErrorAction SilentlyContinue |
            Select-Object -ExpandProperty Id -Unique)
        if ($allowedPids.Count -gt 0) {
            [System.Windows.Automation.Condition[]]$pidConditions = @(
                foreach ($processId in $allowedPids) {
                    [System.Windows.Automation.PropertyCondition]::new(
                        [System.Windows.Automation.AutomationElement]::ProcessIdProperty,
                        [int]$processId)
                }
            )
            $processCondition = if ($pidConditions.Count -eq 1) {
                $pidConditions[0]
            }
            else {
                [System.Windows.Automation.OrCondition]::new($pidConditions)
            }
            [System.Windows.Automation.Condition[]]$conditions = @(
                $processCondition,
                [System.Windows.Automation.PropertyCondition]::new(
                    [System.Windows.Automation.AutomationElement]::NameProperty,
                    '进入微信')
            )
            $condition = [System.Windows.Automation.AndCondition]::new($conditions)
            $element = [System.Windows.Automation.AutomationElement]::RootElement.FindFirst(
                [System.Windows.Automation.TreeScope]::Descendants,
                $condition)
            if ($null -ne $element) {
                $pattern = $null
                if ($element.TryGetCurrentPattern(
                        [System.Windows.Automation.InvokePattern]::Pattern,
                        [ref]$pattern)) {
                    ([System.Windows.Automation.InvokePattern]$pattern).Invoke()
                    return $true
                }
                if ($element.TryGetCurrentPattern(
                        [System.Windows.Automation.LegacyIAccessiblePattern]::Pattern,
                        [ref]$pattern)) {
                    ([System.Windows.Automation.LegacyIAccessiblePattern]$pattern).DoDefaultAction()
                    return $true
                }
            }
        }
        Start-Sleep -Milliseconds 250
    }
    return $false
}

function Invoke-CaptureValidation {
    param(
        [string]$CliPath,
        [string]$CapturePath,
        [uint32]$WeChatPID,
        [string]$StdoutPath,
        [string]$StderrPath
    )
    $env:WECHAT_CLI_CAPTURE_KEY_FILE = $CapturePath
    $env:WECHAT_CLI_CAPTURE_PID = [string]$WeChatPID
    Remove-Item -LiteralPath $StdoutPath -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $StderrPath -Force -ErrorAction SilentlyContinue
    $process = Start-Process -FilePath $CliPath `
        -ArgumentList @('setup', '--pretty') `
        -NoNewWindow -Wait -PassThru `
        -RedirectStandardOutput $StdoutPath `
        -RedirectStandardError $StderrPath
    return $process.ExitCode
}

function Remove-PrivateCaptureWorkspace {
    param([pscustomobject]$Workspace)

    Remove-Item -LiteralPath $Workspace.Stdout -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $Workspace.Stderr -Force -ErrorAction SilentlyContinue
    for ($attempt = 0; $attempt -lt 20; $attempt++) {
        Remove-Item -LiteralPath $Workspace.Capture -Force -ErrorAction SilentlyContinue
        if (-not (Test-Path -LiteralPath $Workspace.Capture)) {
            break
        }
        Start-Sleep -Milliseconds 100
    }
    Remove-Item -LiteralPath $Workspace.Directory -Force -ErrorAction SilentlyContinue
    return -not (Test-Path -LiteralPath $Workspace.Capture)
}

function Get-WechatCliConfigPath {
    if (-not [string]::IsNullOrWhiteSpace($env:WECHAT_CLI_CONFIG)) {
        return [IO.Path]::GetFullPath($env:WECHAT_CLI_CONFIG)
    }
    if (-not [string]::IsNullOrWhiteSpace($env:WX_MCP_CONFIG)) {
        return [IO.Path]::GetFullPath($env:WX_MCP_CONFIG)
    }
    return Join-Path ([Environment]::GetFolderPath('UserProfile')) '.config\wxcli\config.json'
}

$WeixinExe = Resolve-RequiredFile $WeixinExe 'Weixin executable'
$CaptureDll = Resolve-RequiredFile $CaptureDll 'capture DLL'
$WeixinKey = Resolve-RequiredFile $WeixinKey 'weixin-key executable'
$configPath = Get-WechatCliConfigPath
$configExistedBefore = Test-Path -LiteralPath $configPath -PathType Leaf
$workspace = $null
$readyEvent = $null
$launchedPid = [uint32]0
$candidateCount = 0
$lastCliExitCode = $null
$verified = $false
$uiAction = 'not_attempted'
$failure = $null
$exitCode = 2
$cleanupOK = $true
$environmentNames = @(
    'WECHAT_CLI_KEY_CAPTURE_FILE',
    'WECHAT_CLI_KEY_CAPTURE_READY_EVENT',
    'WECHAT_CLI_CAPTURE_KEY_FILE',
    'WECHAT_CLI_CAPTURE_PID'
)
$oldEnvironment = @{}
foreach ($name in $environmentNames) {
    $oldEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}

try {
    $workspace = New-PrivateCaptureWorkspace
    $eventName = 'Local\wechat-cli-key-capture-' + [Guid]::NewGuid().ToString('N')
    $readyEvent = [Threading.EventWaitHandle]::new(
        $false,
        [Threading.EventResetMode]::ManualReset,
        $eventName)

    $env:WECHAT_CLI_KEY_CAPTURE_FILE = $workspace.Capture
    $env:WECHAT_CLI_KEY_CAPTURE_READY_EVENT = $eventName

    Stop-WeChatProcesses
    $launchedPid = [SuspendedInjector]::LaunchAndInject($WeixinExe, $CaptureDll)

    if (-not $readyEvent.WaitOne([TimeSpan]::FromSeconds($HookReadyTimeoutSeconds))) {
        throw 'capture DLL did not report a ready hook before the timeout'
    }

    if (Invoke-EnterWeChat -TimeoutSeconds $UIClickTimeoutSeconds) {
        $uiAction = 'invoked_enter_wechat'
    }
    else {
        $uiAction = 'manual_click_requested'
        Write-Warning 'The Enter WeChat control was not found. Click it manually while key capture continues.'
    }

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    $lastAttemptCount = 0
    while ([DateTime]::UtcNow -lt $deadline) {
        $candidateCount = Get-CaptureCandidateCount -Path $workspace.Capture
        if ($candidateCount -gt $lastAttemptCount) {
            $lastCliExitCode = Invoke-CaptureValidation `
                -CliPath $WeixinKey `
                -CapturePath $workspace.Capture `
                -WeChatPID $launchedPid `
                -StdoutPath $workspace.Stdout `
                -StderrPath $workspace.Stderr
            $lastAttemptCount = $candidateCount
            if ($lastCliExitCode -eq 0) {
                $verified = $true
                $exitCode = 0
                break
            }
        }
        Start-Sleep -Milliseconds 250
    }

    if (-not $verified) {
        if ($candidateCount -eq 0) {
            $failure = 'capture timeout before any candidate was recorded'
        }
        else {
            $failure = 'captured candidates did not validate before the timeout'
        }
    }
}
catch {
    $failure = $_.Exception.Message
}
finally {
    foreach ($name in $environmentNames) {
        [Environment]::SetEnvironmentVariable($name, $oldEnvironment[$name], 'Process')
    }
    if ($null -ne $readyEvent) {
        $readyEvent.Dispose()
    }
    if ($null -ne $workspace) {
        if ($KeepCaptureFile) {
            Remove-Item -LiteralPath $workspace.Stdout -Force -ErrorAction SilentlyContinue
            Remove-Item -LiteralPath $workspace.Stderr -Force -ErrorAction SilentlyContinue
        }
        else {
            $cleanupOK = Remove-PrivateCaptureWorkspace -Workspace $workspace
        }
    }
}

if (-not $cleanupOK) {
    $verified = $false
    $exitCode = 2
    $failure = 'temporary capture file cleanup failed; the private capture directory requires manual removal'
}

$summary = [ordered]@{
    Ok = $verified
    PID = $launchedPid
    CandidateCount = $candidateCount
    Verified = $verified
    CLIExitCode = $lastCliExitCode
    UIAction = $uiAction
    ConfigPath = $configPath
    ConfigExistedBefore = $configExistedBefore
    ConfigPresentAfter = (Test-Path -LiteralPath $configPath -PathType Leaf)
    CaptureFileRetained = ($null -ne $workspace -and (Test-Path -LiteralPath $workspace.Capture -PathType Leaf))
}
if ($KeepCaptureFile -and $null -ne $workspace -and (Test-Path -LiteralPath $workspace.Capture -PathType Leaf)) {
    $summary.RetainedCapturePath = $workspace.Capture
}
if (-not [string]::IsNullOrWhiteSpace($failure)) {
    $summary.Error = $failure
}
$summary | ConvertTo-Json -Compress
exit $exitCode

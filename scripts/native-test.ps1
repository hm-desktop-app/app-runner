param([string]$BinaryPath = '')
$ErrorActionPreference = 'Stop'
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using System.Text;
using System.Security.Principal;
public static class NativeTestWindow {
    private delegate bool EnumProc(IntPtr hwnd, IntPtr value);
    [DllImport("user32.dll")] private static extern bool EnumWindows(EnumProc callback, IntPtr value);
    [DllImport("user32.dll")] private static extern bool EnumChildWindows(IntPtr parent, EnumProc callback, IntPtr value);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint pid);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] private static extern int GetWindowText(IntPtr hwnd, StringBuilder title, int length);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr hwnd, StringBuilder name, int length);
    [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr hwnd, uint message, IntPtr wparam, IntPtr lparam);
    [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr hwnd);
    [DllImport("kernel32.dll")] private static extern IntPtr OpenProcess(uint access, bool inherit, uint id);
    [DllImport("advapi32.dll")] private static extern bool OpenProcessToken(IntPtr process, uint access, out IntPtr token);
    [DllImport("advapi32.dll")] private static extern bool GetTokenInformation(IntPtr token, int info, IntPtr buffer, int length, out int returned);
    [DllImport("kernel32.dll")] private static extern bool CloseHandle(IntPtr handle);
    public static string Integrity(uint pid) {
        var process = OpenProcess(0x1000, false, pid);
        IntPtr token;
        try {
            if (!OpenProcessToken(process, 8, out token)) throw new Exception("OpenProcessToken failed");
            try {
                int length;
                GetTokenInformation(token, 25, IntPtr.Zero, 0, out length);
                var buffer = Marshal.AllocHGlobal(length);
                try {
                    if (!GetTokenInformation(token, 25, buffer, length, out length)) throw new Exception("GetTokenInformation failed");
                    return new SecurityIdentifier(Marshal.ReadIntPtr(buffer)).Value;
                } finally { Marshal.FreeHGlobal(buffer); }
            } finally { CloseHandle(token); }
        } finally { CloseHandle(process); }
    }
    public static IntPtr Find(uint processId) {
        IntPtr found = IntPtr.Zero;
        EnumWindows((hwnd, _) => {
            uint pid;
            GetWindowThreadProcessId(hwnd, out pid);
            var title = new StringBuilder(256);
            GetWindowText(hwnd, title, title.Capacity);
            if (pid == processId && title.ToString().StartsWith("App Runner")) { found = hwnd; return false; }
            return true;
        }, IntPtr.Zero);
        return found;
    }
    public static bool HasWebView(IntPtr parent) {
        bool found = false;
        EnumChildWindows(parent, (hwnd, _) => {
            var name = new StringBuilder(256);
            GetClassName(hwnd, name, name.Capacity);
            if (name.ToString().StartsWith("Chrome_")) found = true;
            return true;
        }, IntPtr.Zero);
        return found;
    }
}
'@
$taskProject = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$taskBinary = if ($BinaryPath) { [IO.Path]::GetFullPath($BinaryPath) } else { Join-Path $taskProject 'bin\app-runner.exe' }
$taskFixture = Join-Path $taskProject 'selfcheck\native-library'
New-Item -ItemType Directory -Force (Join-Path $taskFixture 'sample\content') | Out-Null
Set-Content -LiteralPath (Join-Path $taskFixture 'sample\app.toml') -Value "title = 'Native Test'`ntype = 'GUI'`nentry = 'app.exe'" -Encoding utf8
Set-Content -LiteralPath (Join-Path $taskFixture 'sample\content\app.exe') -Value 'test fixture only' -Encoding utf8
$taskFirst = $null
$taskSecond = $null
$taskRestart = $null
$taskStdout = Join-Path $taskProject 'selfcheck\native-stdout.log'
$taskStderr = Join-Path $taskProject 'selfcheck\native-stderr.log'
$taskWindow = [IntPtr]::Zero
$taskOldLocal = $env:LOCALAPPDATA
$taskOldRoaming = $env:APPDATA
$taskProfile = Join-Path $taskProject ('selfcheck\native-profile-' + [Guid]::NewGuid().ToString('N'))
function Wait-NativeWindow($taskProcess) {
    for ($taskAttempt = 0; $taskAttempt -lt 60; $taskAttempt++) {
        Start-Sleep -Milliseconds 200
        $taskProcess.Refresh()
        if ($taskProcess.HasExited) { throw "Native application exited during startup ($($taskProcess.ExitCode)): $(Get-Content -LiteralPath $taskStderr -Raw)" }
        $taskHandle = [NativeTestWindow]::Find($taskProcess.Id)
        if ($taskHandle -ne [IntPtr]::Zero -and [NativeTestWindow]::HasWebView($taskHandle) -and (Select-String -LiteralPath $taskStderr -Pattern 'App Runner: native window ready' -Quiet)) { return $taskHandle }
    }
    throw "Native WebView2 window did not initialise. $(Get-Content -LiteralPath $taskStderr -Raw)"
}
try {
    $env:LOCALAPPDATA = Join-Path $taskProfile 'Local'
    $env:APPDATA = Join-Path $taskProfile 'Roaming'
    New-Item -ItemType Directory -Force $env:LOCALAPPDATA,$env:APPDATA | Out-Null
    $taskLock = Join-Path $env:LOCALAPPDATA 'app-runner\instance.lock'
    $taskArgs = @('--app-dir', ('"' + $taskFixture + '"'))
    $taskFirst = Start-Process -FilePath $taskBinary -ArgumentList $taskArgs -WorkingDirectory $taskProject -WindowStyle Hidden -RedirectStandardOutput $taskStdout -RedirectStandardError $taskStderr -PassThru
    $taskWindow = Wait-NativeWindow $taskFirst
    if (-not (Test-Path -LiteralPath $taskLock -PathType Leaf)) { throw 'Instance lock file was not created' }
    $taskIntegrity = [NativeTestWindow]::Integrity($taskFirst.Id)
    if ($taskIntegrity -ne 'S-1-16-8192') { throw "Expected normal Medium integrity, got $taskIntegrity" }
    if (-not [NativeTestWindow]::PostMessage($taskWindow, 0x10, [IntPtr]::Zero, [IntPtr]::Zero)) { throw 'Could not request window close' }
    Start-Sleep -Milliseconds 500
    $taskFirst.Refresh()
    if ($taskFirst.HasExited -or [NativeTestWindow]::IsWindowVisible($taskWindow)) { throw 'Closing the window should hide to tray' }
    $taskSecond = Start-Process -FilePath $taskBinary -ArgumentList $taskArgs -WorkingDirectory $taskFixture -WindowStyle Hidden -PassThru
    if (-not $taskSecond.WaitForExit(10000)) { throw 'Second instance did not exit' }
    if ($taskSecond.ExitCode -ne 0) { throw "Second instance exit code: $($taskSecond.ExitCode)" }
    Start-Sleep -Milliseconds 500
    $taskFirst.Refresh()
    if ($taskFirst.HasExited -or -not [NativeTestWindow]::IsWindowVisible($taskWindow)) { throw 'Second instance did not restore the first window' }
    Stop-Process -Id $taskFirst.Id -Force
    if (-not $taskFirst.WaitForExit(10000)) { throw 'First instance did not terminate' }
    if (-not (Test-Path -LiteralPath $taskLock -PathType Leaf)) { throw 'Lock file should survive forced termination' }
    $taskRestart = Start-Process -FilePath $taskBinary -ArgumentList $taskArgs -WorkingDirectory $taskProject -WindowStyle Hidden -RedirectStandardOutput $taskStdout -RedirectStandardError $taskStderr -PassThru
    $null = Wait-NativeWindow $taskRestart
    $taskPrincipal = [Security.Principal.WindowsPrincipal]::new([Security.Principal.WindowsIdentity]::GetCurrent())
    $taskReport = @{ passed = $true; integrity = $taskIntegrity; elevated = $taskPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator); checks = @('native WebView2 window', 'close hides to tray', 'file lock and second launch activation from another working directory', 'restart after forced termination with lock file still present'); processId = $taskFirst.Id } | ConvertTo-Json
    Set-Content -LiteralPath (Join-Path $taskProject 'selfcheck\native-test.json') -Value $taskReport -Encoding utf8
    Write-Output $taskReport
} finally {
    $env:LOCALAPPDATA = $taskOldLocal
    $env:APPDATA = $taskOldRoaming
    if ($null -ne $taskRestart -and -not $taskRestart.HasExited) { Stop-Process -Id $taskRestart.Id -Force }
    if ($null -ne $taskSecond -and -not $taskSecond.HasExited) { Stop-Process -Id $taskSecond.Id -Force }
    if ($null -ne $taskFirst -and -not $taskFirst.HasExited) { Stop-Process -Id $taskFirst.Id -Force }
}

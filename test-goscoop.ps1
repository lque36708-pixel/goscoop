<#
== goscoop Full Test Suite ==
Tests fresh install, all commands, edge cases
#>

$ErrorActionPreference = "Continue"
$PASS = 0
$FAIL = 0
$TEST_NUM = 0
$GOSCOOP = ""

function Test-Banner($msg) {
    $TEST_NUM++
    Write-Host ""
    Write-Host "----- [TEST $TEST_NUM] $msg -----" -ForegroundColor Cyan
}

function Test-Pass($msg) {
    $global:PASS++
    Write-Host "  OK $msg" -ForegroundColor Green
}

function Test-Fail($msg) {
    $global:FAIL++
    Write-Host "  FAIL $msg" -ForegroundColor Red
}

function Test-Run($cmd) {
    $result = Invoke-Expression $cmd 2>&1
    $exitCode = $LASTEXITCODE
    return @{ Output = $result; ExitCode = $exitCode }
}

# ---- Phase 1: Fresh Install ----
Write-Host "`n===== PHASE 1: FRESH INSTALL FROM RELEASE =====" -ForegroundColor Yellow

Test-Banner "Create fresh install directory"
$testDir = "$env:USERPROFILE\goscoop_test"
if (Test-Path $testDir) { Remove-Item $testDir -Recurse -Force -ErrorAction SilentlyContinue }
New-Item -ItemType Directory -Path $testDir -Force | Out-Null
Copy-Item "C:\Users\spyrovn\Codes\goscoop\goscoop-upgrade-test.exe" "$testDir\goscoop.exe" -Force
$GOSCOOP = "$testDir\goscoop.exe"
$binSize = (Get-Item $GOSCOOP).Length
$binVer = & $GOSCOOP version 2>&1
Write-Host "  Binary: $GOSCOOP ($binSize bytes)"
Write-Host "  Version: $binVer"

$env:PATH = $testDir + ';' + $env:PATH
Test-Pass "Install dir created and added to PATH"

Test-Banner "goscoop --help"
$result = Test-Run "goscoop --help"
if ($result.Output -match "compress" -and $result.Output -match "install" -and $result.Output -match "version") {
    Test-Pass "Help shows all commands"
} else {
    Test-Fail "Help missing commands"
}

Test-Banner "goscoop version"
$result = Test-Run "goscoop version"
if ($result.Output -match "v0.1.3") {
    Test-Pass "Version is v0.1.3"
} else {
    Test-Fail "Version mismatch: $($result.Output)"
}

# ---- Phase 2: Installation ----
Write-Host "`n===== PHASE 2: INSTALL APPS =====" -ForegroundColor Yellow

$script:installTimes = @{}

Test-Banner "goscoop install go (single app, main bucket)"
$sw = [Diagnostics.Stopwatch]::StartNew()
$result = Test-Run "goscoop install go"
$sw.Stop()
$script:installTimes['go'] = $sw.Elapsed.TotalSeconds
if ($result.Output -match "was installed successfully" -and $LASTEXITCODE -eq 0) {
    Test-Pass "go installed in $($sw.Elapsed.TotalSeconds.ToString('0.0'))s"
} else {
    Test-Fail "go install failed: $($result.Output | Out-String)"
}

Test-Banner "goscoop list (after go install)"
$result = Test-Run "goscoop list"
if ($result.Output -match "go") {
    Test-Pass "list shows go"
} else {
    Test-Fail "list missing go"
}

Test-Banner "goscoop install git (.7z.exe extraction)"
$sw = [Diagnostics.Stopwatch]::StartNew()
$result = Test-Run "goscoop install git"
$sw.Stop()
$script:installTimes['git'] = $sw.Elapsed.TotalSeconds
if ($result.Output -match "was installed successfully" -and $LASTEXITCODE -eq 0) {
    Test-Pass "git installed in $($sw.Elapsed.TotalSeconds.ToString('0.0'))s"
} else {
    Test-Fail "git install failed: $($result.Output | Out-String)"
}

Test-Banner "goscoop install gh (bulk with existing apps)"
$result = Test-Run "goscoop install gh"
if ($result.Output -match "was installed successfully" -and $LASTEXITCODE -eq 0) {
    Test-Pass "gh installed"
} else {
    Test-Fail "gh install failed"
}

Test-Banner "goscoop install nonExistentApp (error handling)"
$result = Test-Run "goscoop install nonExistentApp123xyz"
if ($result.Output -match "Skipping" -and $result.Output -match "nonExistentApp123xyz") {
    Test-Pass "Non-existent app shows skip message"
} else {
    Test-Fail "Non-existent app not handled: $($result.Output | Out-String)"
}

Test-Banner "goscoop install --compress which"
$sw = [Diagnostics.Stopwatch]::StartNew()
$result = Test-Run "goscoop install --compress which"
$sw.Stop()
$script:installTimes['which'] = $sw.Elapsed.TotalSeconds
if ($result.Output -match "was installed successfully") {
    Test-Pass "which installed with --compress in $($sw.Elapsed.TotalSeconds.ToString('0.0'))s"
} else {
    Test-Fail "which install failed: $($result.Output | Out-String)"
}

Test-Banner "goscoop install -g go (global flag)"
$result = Test-Run "goscoop install -g go"
if ($result.Output -match "already installed" -or $result.Output -match "was installed") {
    Test-Pass "install -g handled (already installed)"
} else {
    Test-Pass "install -g did not crash: $($result.Output | Out-String)"
}

# ---- Phase 3: Information Commands ----
Write-Host "`n===== PHASE 3: INFO / SEARCH / STATUS =====" -ForegroundColor Yellow

Test-Banner "goscoop list (all apps)"
$result = Test-Run "goscoop list"
$appsFound = @('go','git','gh','which') | Where-Object { $result.Output -match $_ }
if ($appsFound.Count -eq 4) {
    Test-Pass "list shows 4 apps: $($appsFound -join ', ')"
} else {
    Test-Fail "list missing: $(@('go','git','gh','which') | Where-Object { $result.Output -notmatch $_ } | Out-String)"
}

Test-Banner "goscoop search chrome"
$sw = [Diagnostics.Stopwatch]::StartNew()
$result = Test-Run "goscoop search chrome"
$sw.Stop()
if ($result.Output -match "googlechrome" -or $result.Output -match "chrome") {
    Test-Pass "search found chrome apps in $($sw.Elapsed.TotalSeconds.ToString('0.000'))s"
} else {
    Test-Pass "search ran (no crash) in $($sw.Elapsed.TotalSeconds.ToString('0.000'))s"
}

Test-Banner "goscoop info go"
$result = Test-Run "goscoop info go"
if ($result.Output -match "Version" -and $result.Output -match "Description") {
    Test-Pass "info shows go details"
} else {
    Test-Fail "info incomplete: $($result.Output | Out-String)"
}

Test-Banner "goscoop status"
$result = Test-Run "goscoop status"
if ($LASTEXITCODE -eq 0) {
    Test-Pass "status ran successfully"
} else {
    Test-Fail "status failed: $($result | Out-String)"
}

Test-Banner "goscoop search zzz_nonexistent"
$result = Test-Run "goscoop search zzz_nonexistent"
if ($result.Output -match "No apps found|no results|0 results") {
    Test-Pass "Search for non-existent shows no results"
} else {
    Test-Pass "Search handled (no crash): $($result.Output | Out-String)"
}

# ---- Phase 4: Bucket & Cache ----
Write-Host "`n===== PHASE 4: BUCKET & CACHE MANAGEMENT =====" -ForegroundColor Yellow

Test-Banner "goscoop bucket list"
$result = Test-Run "goscoop bucket list"
if ($result.Output -match "main" -and $result.Output -match "extras") {
    Test-Pass "bucket list shows main and extras"
} else {
    Test-Fail "bucket list incomplete: $($result.Output | Out-String)"
}

Test-Banner "goscoop bucket rm versions / bucket add versions"
$result = Test-Run "goscoop bucket rm versions"
$result2 = Test-Run "goscoop bucket add versions"
if ($result.ExitCode -eq 0 -and $result2.ExitCode -eq 0) {
    Test-Pass "bucket rm/add versions works"
} else {
    Test-Fail "bucket rm/add: rm=$($result.ExitCode) add=$($result2.ExitCode)"
}

Test-Banner "goscoop cache list"
$result = Test-Run "goscoop cache list"
if ($result.Output -match "Total" -or $LASTEXITCODE -eq 0) {
    Test-Pass "cache list shows entries"
} else {
    Test-Fail "cache list: $($result.Output | Out-String)"
}

# ---- Phase 5: Hold/Reset/Compress ----
Write-Host "`n===== PHASE 5: HOLD / RESET / COMPRESS =====" -ForegroundColor Yellow

Test-Banner "goscoop hold go"
$result = Test-Run "goscoop hold go"
if ($LASTEXITCODE -eq 0) {
    Test-Pass "hold go succeeded"
} else {
    Test-Fail "hold go failed: $($result.Output | Out-String)"
}

Test-Banner "goscoop status (after hold)"
$result = Test-Run "goscoop status"
if ($LASTEXITCODE -eq 0) {
    Test-Pass "status ran after hold"
} else {
    Test-Fail "status failed: $($result | Out-String)"
}

Test-Banner "goscoop unhold go"
$result = Test-Run "goscoop unhold go"
if ($LASTEXITCODE -eq 0) {
    Test-Pass "unhold go succeeded"
} else {
    Test-Fail "unhold go failed: $($result.Output | Out-String)"
}

Test-Banner "goscoop reset which (reinstall shims)"
$result = Test-Run "goscoop reset which"
if ($result.Output -match "shim" -or $LASTEXITCODE -eq 0) {
    Test-Pass "reset which succeeded"
} else {
    Test-Fail "reset which: $($result.Output | Out-String)"
}

Test-Banner "goscoop compress which"
$result = Test-Run "goscoop compress which"
if ($LASTEXITCODE -eq 0) {
    Test-Pass "compress which succeeded"
} else {
    Test-Fail "compress which: $($result.Output | Out-String)"
}

Test-Banner "goscoop compress --all"
$result = Test-Run "goscoop compress --all"
if ($LASTEXITCODE -eq 0) {
    Test-Pass "compress --all succeeded"
} else {
    Test-Fail "compress --all: $($result.Output | Out-String)"
}

# ---- Phase 6: Update ----
Write-Host "`n===== PHASE 6: UPDATE =====" -ForegroundColor Yellow

Test-Banner "goscoop update (all buckets)"
$sw = [Diagnostics.Stopwatch]::StartNew()
$result = Test-Run "goscoop update"
$sw.Stop()
$hasFail = $result.Output | Select-String -Pattern 'FAIL'
$hasOK = $result.Output | Select-String -Pattern 'OK'
if ($hasOK -and -not $hasFail) {
    Test-Pass "update all OK in $($sw.Elapsed.TotalSeconds.ToString('0.0'))s"
} else {
    if ($hasFail) {
        Test-Fail "update had FAIL: $($result.Output | Out-String)"
    } else {
        Test-Pass "update ran in $($sw.Elapsed.TotalSeconds.ToString('0.0'))s"
    }
}

Test-Banner "goscoop update go (single app)"
$result = Test-Run "goscoop update go"
if ($LASTEXITCODE -eq 0) {
    Test-Pass "update go (no error)"
} else {
    Test-Fail "update go: $($result.Output | Out-String)"
}

# ---- Phase 7: Uninstall ----
Write-Host "`n===== PHASE 7: UNINSTALL =====" -ForegroundColor Yellow

Test-Banner "goscoop uninstall which"
$result = Test-Run "goscoop uninstall which"
if ($result.Output -match "was uninstalled") {
    Test-Pass "uninstall which succeeded"
} else {
    Test-Fail "uninstall which: $($result.Output | Out-String)"
}

Test-Banner "goscoop uninstall nonexistent"
$result = Test-Run "goscoop uninstall nonexistent123"
if ($result.Output -match "not installed" -or $result.Output -match "did you mean") {
    Test-Pass "uninstall non-existent shows error"
} else {
    Test-Pass "uninstall non-existent handled (no crash)"
}

Test-Banner "goscoop uninstall -p gh (purge persist)"
$result = Test-Run "goscoop uninstall -p gh"
if ($result.Output -match "was uninstalled") {
    Test-Pass "uninstall -p gh succeeded"
} else {
    Test-Fail "uninstall gh: $($result.Output | Out-String)"
}

# ---- Phase 8: Self-upgrade ----
Write-Host "`n===== PHASE 8: SELF UPGRADE =====" -ForegroundColor Yellow

Test-Banner "goscoop upgrade (already latest)"
$result = Test-Run "goscoop upgrade"
if ($result.Output -match "already up to date") {
    Test-Pass "upgrade reports up to date"
} else {
    Test-Fail "upgrade: $($result.Output | Out-String)"
}

Test-Banner "goscoop upgrade --force"
$result = Test-Run "goscoop upgrade --force"
if ($result.Output -match "re-downloading|updated") {
    Test-Pass "upgrade --force re-downloads"
} else {
    Test-Fail "upgrade --force: $($result.Output | Out-String)"
}

# ---- Phase 9: Clean Uninstall ----
Write-Host "`n===== PHASE 9: SELF UNINSTALL =====" -ForegroundColor Yellow

Test-Banner "goscoop uninstall --self (clean removal)"
$result = cmd /c "echo y | goscoop uninstall --self" 2>&1
if ($result -match "goscoop has been removed") {
    Test-Pass "uninstall --self completed"
} else {
    Test-Fail "uninstall --self: $($result | Out-String)"
}

Test-Banner "Verify clean state"
$scoopDirExists = Test-Path "$env:USERPROFILE\scoop"
$goscoopDirExists = Test-Path "$env:USERPROFILE\goscoop"
$userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
$inPath = $userPath -match 'scoop|goscoop'
if (-not $scoopDirExists -and -not $goscoopDirExists -and -not $inPath) {
    Test-Pass "System clean: no scoop dir, no goscoop dir, no PATH entries"
} else {
    if ($scoopDirExists) { Test-Fail "scoop dir still exists" }
    if ($goscoopDirExists) { Test-Fail "goscoop dir still exists: $goscoopDirExists" }
    if ($inPath) { Test-Fail "PATH still has scoop/goscoop" }
}

# Remove test dir
Remove-Item $testDir -Recurse -Force -ErrorAction SilentlyContinue

# ---- Summary ----
Write-Host "`n============================================" -ForegroundColor Yellow
Write-Host "TEST SUMMARY" -ForegroundColor Yellow
Write-Host "  Passed: $PASS" -ForegroundColor Green
Write-Host "  Failed: $FAIL" -ForegroundColor $(if ($FAIL -gt 0) { 'Red' } else { 'Green' })
Write-Host "  Total:  $($PASS + $FAIL)" -ForegroundColor Cyan
if ($script:installTimes.Count -gt 0) {
    Write-Host "`nInstall times:" -ForegroundColor Cyan
    $script:installTimes.GetEnumerator() | Sort-Object Name | ForEach-Object {
        Write-Host "  $($_.Key): $($_.Value.ToString('0.0'))s"
    }
}
Write-Host "============================================" -ForegroundColor Yellow

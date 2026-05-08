$ErrorActionPreference = "Stop"

$baseUrl = "http://localhost:8081"
$apiUrl = "$baseUrl/api"

# Configure the accounts you want to use (keep this short to avoid auth rate limits)
$credentials = @(
    @{ key = "admin"; email = "23bsm007@iiitdmj.ac.in"; password = "user123" },
    @{ key = "student_owner"; email = "23bsm001@iiitdmj.ac.in"; password = "user123" },
    @{ key = "student_non_owner"; email = "23bsm002@iiitdmj.ac.in"; password = "user123" },
    @{ key = "student_target"; email = "23bsm003@iiitdmj.ac.in"; password = "user123" }
)

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Path,
        [string]$Token = $null,
        $Body = $null
    )

    $headers = @{}
    if ($Token) {
        $headers["Authorization"] = "Bearer $Token"
    }

    try {
        if ($Body -ne $null) {
            $json = $Body | ConvertTo-Json -Depth 6
            $resp = Invoke-RestMethod -Method $Method -Uri ($apiUrl + $Path) -Headers $headers -ContentType "application/json" -Body $json -ErrorAction Stop
        } else {
            $resp = Invoke-RestMethod -Method $Method -Uri ($apiUrl + $Path) -Headers $headers -ErrorAction Stop
        }
        return @{ ok = $true; data = $resp }
    } catch {
        $errMsg = $_.ErrorDetails.Message
        if (-not $errMsg) {
            $errMsg = $_.Exception.Message
        }
        return @{ ok = $false; error = $errMsg }
    }
}

function Login-User {
    param([string]$Email, [string]$Password)
    $body = @{ email = $Email; password = $Password }
    $resp = Invoke-Api -Method POST -Path "/auth/login" -Body $body
    if ($resp.ok -and $resp.data.success) {
        return @{ ok = $true; token = $resp.data.data.access_token; user = $resp.data.data.user }
    }
    return @{ ok = $false; error = "Invalid credentials" }
}

function Test-Result {
    param([string]$Name, [bool]$Ok, [string]$Details = "")
    $status = if ($Ok) { "PASS" } else { "FAIL" }
    Write-Host "[$status] $Name"
    if ($Details) { Write-Host "       $Details" }
}

Write-Host "=== CMS Comprehensive Test Run ==="

# 1) Health check
try {
    $health = Invoke-RestMethod -Method GET -Uri "$baseUrl/health"
    Test-Result "Health check" $true $health.mode
} catch {
    Test-Result "Health check" $false $_.Exception.Message
}

# 2) Logins (keep this list short to avoid rate limits)
$tokens = @{}
$users = @{}

Write-Host "\n=== Login Tests ==="
foreach ($u in $credentials) {
    $login = Login-User -Email $u.email -Password $u.password
    if ($login.ok) {
        $tokens[$u.key] = $login.token
        $users[$u.key] = $login.user
        Test-Result "Login $($u.email)" $true
    } else {
        Test-Result "Login $($u.email)" $false $login.error
    }
}

# 3) Find SIH achievement ID
$sihId = $null
$adminToken = $tokens["admin"]

if ($adminToken) {
    $ach = Invoke-Api -Method GET -Path "/tables/achievements" -Token $adminToken
    if ($ach.ok -and $ach.data.data) {
        $match = $ach.data.data | Where-Object { $_.title -match "SIH" } | Select-Object -First 1
        if ($match) { $sihId = $match.id }
    }
}

if (-not $sihId) {
    foreach ($key in $tokens.Keys) {
        if ($key -like "s_*") {
            $ws = Invoke-Api -Method GET -Path "/student/me" -Token $tokens[$key]
            if ($ws.ok -and $ws.data.data.linked_records.achievement_members) {
                $sih = $ws.data.data.linked_records.achievement_members | Where-Object { $_.achievement_title -match "SIH" } | Select-Object -First 1
                if ($sih) { $sihId = $sih.achievement_id; break }
            }
        }
    }
}

Test-Result "Find SIH achievement" ($sihId -ne $null) $sihId

# 4) Student workspaces
Write-Host "\n=== Student Workspace Checks ==="
$studentInfo = @{}
foreach ($key in $tokens.Keys) {
    if ($key -like "student_*") {
        $ws = Invoke-Api -Method GET -Path "/student/me" -Token $tokens[$key]
        if ($ws.ok) {
            $studentId = $ws.data.data.student.id
            $hasSih = $false
            if ($sihId -and $ws.data.data.linked_records.achievement_members) {
                $hasSih = ($ws.data.data.linked_records.achievement_members | Where-Object { $_.achievement_id -eq $sihId }).Count -gt 0
            }
            $studentInfo[$key] = @{ student_id = $studentId; has_sih = $hasSih; name = $ws.data.data.student.name }
            Test-Result "Workspace $($users[$key].email)" $true ("student_id=$studentId; SIH=$hasSih")
        } else {
            Test-Result "Workspace $($users[$key].email)" $false $ws.error
        }
    }
}

# 5) Contributor tests
Write-Host "\n=== Contributor Tests (SIH) ==="
$ownerKey = $studentInfo.Keys | Where-Object { $studentInfo[$_].has_sih } | Select-Object -First 1
$nonOwnerKey = $studentInfo.Keys | Where-Object { -not $studentInfo[$_].has_sih } | Select-Object -First 1

if ($sihId -and $ownerKey) {
    $ownerToken = $tokens[$ownerKey]
    $ownerName = $studentInfo[$ownerKey].name

    $targetKey = $nonOwnerKey
    if (-not $targetKey) { $targetKey = $ownerKey }

    $targetStudentId = $studentInfo[$targetKey].student_id
    $targetName = $studentInfo[$targetKey].name

    $list = Invoke-Api -Method GET -Path ("/student/contributors?type=achievement&id=" + $sihId) -Token $ownerToken
    Test-Result "Get SIH contributors" $list.ok

    $add = Invoke-Api -Method POST -Path "/student/achievements/contributors" -Token $ownerToken -Body @{ achievement_id = $sihId; student_id = $targetStudentId }
    Test-Result "Owner add contributor ($ownerName -> $targetName)" $add.ok $add.error

    if ($tokens[$targetKey]) {
        $wsTarget = Invoke-Api -Method GET -Path "/student/me" -Token $tokens[$targetKey]
        $hasSihTarget = $false
        if ($wsTarget.ok -and $wsTarget.data.data.linked_records.achievement_members) {
            $hasSihTarget = ($wsTarget.data.data.linked_records.achievement_members | Where-Object { $_.achievement_id -eq $sihId }).Count -gt 0
        }
        Test-Result "Target sees SIH after add" $hasSihTarget
    }

    if ($nonOwnerKey) {
        $nonOwnerToken = $tokens[$nonOwnerKey]
        $nonOwnerName = $studentInfo[$nonOwnerKey].name
        $nonOwnerAdd = Invoke-Api -Method POST -Path "/student/achievements/contributors" -Token $nonOwnerToken -Body @{ achievement_id = $sihId; student_id = $targetStudentId }
        Test-Result "Non-owner add blocked ($nonOwnerName)" (-not $nonOwnerAdd.ok) $nonOwnerAdd.error
    }

    $remove = Invoke-Api -Method DELETE -Path "/student/achievements/contributors" -Token $ownerToken -Body @{ achievement_id = $sihId; student_id = $targetStudentId }
    Test-Result "Owner remove contributor" $remove.ok $remove.error

    if ($tokens[$targetKey]) {
        $wsTarget2 = Invoke-Api -Method GET -Path "/student/me" -Token $tokens[$targetKey]
        $hasSihTarget2 = $false
        if ($wsTarget2.ok -and $wsTarget2.data.data.linked_records.achievement_members) {
            $hasSihTarget2 = ($wsTarget2.data.data.linked_records.achievement_members | Where-Object { $_.achievement_id -eq $sihId }).Count -gt 0
        }
        Test-Result "Target sees SIH after remove (should be false)" (-not $hasSihTarget2)
    }

    $badStudent = Invoke-Api -Method POST -Path "/student/achievements/contributors" -Token $ownerToken -Body @{ achievement_id = $sihId; student_id = "00000000-0000-0000-0000-000000000000" }
    Test-Result "Add invalid student rejected" (-not $badStudent.ok) $badStudent.error

    $badAchievement = Invoke-Api -Method POST -Path "/student/achievements/contributors" -Token $ownerToken -Body @{ achievement_id = "00000000-0000-0000-0000-000000000000"; student_id = $targetStudentId }
    Test-Result "Add invalid achievement rejected" (-not $badAchievement.ok) $badAchievement.error

    $missingParams = Invoke-Api -Method POST -Path "/student/achievements/contributors" -Token $ownerToken -Body @{ student_id = $targetStudentId }
    Test-Result "Add missing params rejected" (-not $missingParams.ok) $missingParams.error
} else {
    Test-Result "Contributor tests" $false "Missing SIH ID or no member logged in"
}

# 6) Student search
Write-Host "\n=== Search Test ==="
$anyStudentToken = $tokens.Values | Select-Object -First 1
if ($anyStudentToken) {
    $search = Invoke-Api -Method GET -Path "/student/search?q=Jain" -Token $anyStudentToken
    Test-Result "Search students" $search.ok
}

# 7) Admin endpoints
Write-Host "\n=== Admin Endpoint Tests ==="
if ($adminToken) {
    $pending = Invoke-Api -Method GET -Path "/admin/change-requests?status=pending" -Token $adminToken
    Test-Result "Admin list pending requests" $pending.ok

    if ($anyStudentToken) {
        $studentAdmin = Invoke-Api -Method GET -Path "/admin/change-requests?status=pending" -Token $anyStudentToken
        Test-Result "Student blocked from admin" (-not $studentAdmin.ok) $studentAdmin.error
    }
} else {
    Test-Result "Admin tests" $false "No admin token"
}

Write-Host "\n=== Test Run Complete ==="

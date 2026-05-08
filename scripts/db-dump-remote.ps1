param(
    [string]$OutputPath
)

$ErrorActionPreference = "Stop"

function Import-EnvFile {
    param([Parameter(Mandatory = $true)][string]$Path)

    if (-not (Test-Path $Path)) {
        return
    }

    foreach ($line in Get-Content $Path) {
        if ($line -match '^\s*#' -or $line -notmatch '=') {
            continue
        }

        $parts = $line -split '=', 2
        $name = $parts[0].Trim()
        if ([string]::IsNullOrWhiteSpace($name) -or [Environment]::GetEnvironmentVariable($name, "Process")) {
            continue
        }

        $value = $parts[1]
        if ($value.Length -ge 2 -and (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'")))) {
            $value = $value.Substring(1, $value.Length - 2)
        }

        [Environment]::SetEnvironmentVariable($name, $value, "Process")
    }
}

Import-EnvFile ".env.remote"
Import-EnvFile ".env"

if (-not $env:REMOTE_DATABASE_URL) {
    throw "Set REMOTE_DATABASE_URL before running this script."
}

$env:PGCONNECT_TIMEOUT = if ($env:PGCONNECT_TIMEOUT) { $env:PGCONNECT_TIMEOUT } else { "15" }

$pgClientImage = if ($env:PG_CLIENT_IMAGE) {
    $env:PG_CLIENT_IMAGE
} else {
    "docker.io/library/postgres:17-alpine"
}

if (-not $OutputPath) {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $OutputPath = "backups/prod-$stamp.dump"
}

$backupFile = [System.IO.Path]::GetFullPath($OutputPath)
$backupDir = Split-Path -Parent $backupFile
$backupName = Split-Path -Leaf $backupFile
New-Item -ItemType Directory -Force -Path $backupDir | Out-Null

podman run --rm `
    -e REMOTE_DATABASE_URL `
    -e PGCONNECT_TIMEOUT `
    -v "${backupDir}:/backups" `
    $pgClientImage `
    sh -euc "pg_dump `"`$REMOTE_DATABASE_URL`" --format=custom --no-owner --no-privileges --file=/backups/$backupName"

if ($LASTEXITCODE -ne 0) {
    throw "pg_dump failed with exit code $LASTEXITCODE"
}

$writtenFile = Join-Path $backupDir $backupName
if (-not (Test-Path $writtenFile) -or (Get-Item $writtenFile).Length -eq 0) {
    throw "pg_dump did not create a non-empty dump file."
}

Write-Output "Dump written to $backupFile"

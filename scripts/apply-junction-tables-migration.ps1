# Script to apply junction tables migration to local Supabase
# This creates the achievement_members, project_members, and student_tags tables

$ErrorActionPreference = "Stop"

# Database connection settings
$DbHost = "127.0.0.1"
$DbPort = "54322"
$DbUser = "supabase_admin"
$DbName = "postgres"

# Function to execute SQL via psql
function Invoke-Psql {
    param([string]$Sql)

    # Try using podman exec first, fallback to docker exec
    $podmanCmd = "podman exec --interactive postgres psql -U $DbUser -d $DbName -c `"$Sql`""
    $dockerCmd = "docker exec --interactive postgres psql -U $DbUser -d $DbName -c `"$Sql`""

    try {
        Write-Host "Trying podman..."
        $result = Invoke-Expression $podmanCmd 2>&1
        if ($LASTEXITCODE -eq 0) {
            return $result
        }
    } catch {
        Write-Host "Podman failed, trying docker..."
    }

    try {
        $result = Invoke-Expression $dockerCmd 2>&1
        return $result
    } catch {
        throw "Both podman and docker commands failed. Make sure your local Supabase is running."
    }
}

Write-Host "Applying junction tables migration..." -ForegroundColor Cyan

# Read the migration file
$migrationPath = Join-Path $PSScriptRoot "..\supabase\migrations\20260507200000_junction_tables.sql"
$sqlContent = Get-Content $migrationPath -Raw

# Execute the migration
try {
    $result = Invoke-Psql -Sql $sqlContent
    Write-Host "Migration applied successfully!" -ForegroundColor Green
    Write-Host $result
} catch {
    Write-Host "Migration failed: $_" -ForegroundColor Red
    exit 1
}

# Apply the is_admin function migration
Write-Host "`nApplying is_admin() function migration..." -ForegroundColor Cyan
$functionPath = Join-Path $PSScriptRoot "..\supabase\migrations\20260507205000_is_admin_function.sql"
$sqlContent = Get-Content $functionPath -Raw

try {
    $result = Invoke-Psql -Sql $sqlContent
    Write-Host "Function migration applied successfully!" -ForegroundColor Green
    Write-Host $result
} catch {
    Write-Host "Function migration failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host "`nAll migrations applied successfully!" -ForegroundColor Green
Write-Host "Please restart your backend server." -ForegroundColor Yellow

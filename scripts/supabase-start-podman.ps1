param(
    [switch]$DebugLogs,
    [switch]$Full
)

$ErrorActionPreference = "Stop"
$env:SUPABASE_TELEMETRY_DISABLED = "1"

if (-not (Get-Command podman -ErrorAction SilentlyContinue)) {
    throw "Podman is not available on PATH."
}

$machine = podman machine list --format "{{.Name}} {{.Running}}" |
    Select-String -Pattern "True" |
    Select-Object -First 1

if (-not $machine) {
    throw "No running Podman machine found. Start Podman Desktop or run: podman machine start"
}

$supabaseArgs = @("supabase", "start", "--dns-resolver", "https")
if (-not $Full) {
    # Minimal first boot for this CMS: DB, Auth, REST, and Kong.
    # This avoids pulling optional images until the local stack is stable.
    $supabaseArgs += @(
        "--exclude",
        "realtime,storage-api,imgproxy,mailpit,postgres-meta,studio,edge-runtime,logflare,vector,supavisor"
    )
}
if ($DebugLogs) {
    $supabaseArgs += "--debug"
}

npx @supabaseArgs

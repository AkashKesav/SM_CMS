param(
    [Parameter(Mandatory = $true)]
    [string]$DumpPath,
    [string]$LocalDatabaseUrl = "postgresql://supabase_admin:postgres@127.0.0.1:54322/postgres",
    [switch]$Clean,
    [switch]$PreserveOwners
)

$ErrorActionPreference = "Stop"

$dumpFile = [System.IO.Path]::GetFullPath($DumpPath)
if (-not (Test-Path $dumpFile)) {
    throw "Dump file not found: $dumpFile"
}

$dumpDir = Split-Path -Parent $dumpFile
$dumpName = Split-Path -Leaf $dumpFile

$env:PGCONNECT_TIMEOUT = if ($env:PGCONNECT_TIMEOUT) { $env:PGCONNECT_TIMEOUT } else { "15" }

$pgClientImage = if ($env:PG_CLIENT_IMAGE) {
    $env:PG_CLIENT_IMAGE
} else {
    "docker.io/library/postgres:17-alpine"
}

function Invoke-LocalSql {
    param([Parameter(Mandatory = $true)][string]$Sql)

    $Sql | podman run --rm -i --network host `
        -e PGCONNECT_TIMEOUT `
        $pgClientImage `
        sh -euc "psql '$LocalDatabaseUrl' -v ON_ERROR_STOP=1"

    if ($LASTEXITCODE -ne 0) {
        throw "psql failed with exit code $LASTEXITCODE"
    }
}

if ($Clean) {
    Invoke-LocalSql @'
DROP PUBLICATION IF EXISTS supabase_realtime;
DROP EXTENSION IF EXISTS pg_graphql CASCADE;
DROP EXTENSION IF EXISTS pg_net CASCADE;
DROP EXTENSION IF EXISTS pg_stat_statements CASCADE;
DROP EXTENSION IF EXISTS pg_trgm CASCADE;
DROP EXTENSION IF EXISTS pgcrypto CASCADE;
DROP EXTENSION IF EXISTS supabase_vault CASCADE;
DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;

DO $$
DECLARE
    trigger_record record;
BEGIN
    FOR trigger_record IN SELECT evtname FROM pg_event_trigger LOOP
        EXECUTE format('DROP EVENT TRIGGER IF EXISTS %I', trigger_record.evtname);
    END LOOP;
END $$;

DROP SCHEMA IF EXISTS auth CASCADE;
DROP SCHEMA IF EXISTS extensions CASCADE;
DROP SCHEMA IF EXISTS graphql CASCADE;
DROP SCHEMA IF EXISTS graphql_public CASCADE;
DROP SCHEMA IF EXISTS pgbouncer CASCADE;
DROP SCHEMA IF EXISTS realtime CASCADE;
DROP SCHEMA IF EXISTS storage CASCADE;
DROP SCHEMA IF EXISTS vault CASCADE;
DROP SCHEMA IF EXISTS public CASCADE;

CREATE SCHEMA public AUTHORIZATION pg_database_owner;
GRANT ALL ON SCHEMA public TO postgres;
GRANT USAGE ON SCHEMA public TO anon, authenticated, service_role;
GRANT ALL ON SCHEMA public TO supabase_admin;
'@
}

$restoreArgs = "--no-privileges"
if (-not $PreserveOwners) {
    $restoreArgs = "--no-owner $restoreArgs"
}

podman run --rm --network host `
    -e PGCONNECT_TIMEOUT `
    -v "${dumpDir}:/backups" `
    $pgClientImage `
    sh -euc "pg_restore --dbname='$LocalDatabaseUrl' $restoreArgs /backups/$dumpName"

if ($LASTEXITCODE -ne 0) {
    throw "pg_restore failed with exit code $LASTEXITCODE"
}

Invoke-LocalSql @'
ALTER ROLE supabase_auth_admin SET search_path = auth;
ALTER ROLE supabase_storage_admin SET search_path = storage;
ALTER ROLE postgres SET search_path = "$user", public, extensions;

GRANT USAGE, CREATE ON SCHEMA auth TO supabase_auth_admin;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA auth TO supabase_auth_admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA auth TO supabase_auth_admin;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO supabase_auth_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL PRIVILEGES ON TABLES TO supabase_auth_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL PRIVILEGES ON SEQUENCES TO supabase_auth_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT EXECUTE ON FUNCTIONS TO supabase_auth_admin;
GRANT USAGE ON SCHEMA auth TO authenticator, anon, authenticated, service_role;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO anon, authenticated, service_role;

GRANT USAGE ON SCHEMA public TO anon, authenticated, service_role;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO anon, authenticated;
GRANT INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO authenticated;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO service_role;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO anon, authenticated;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO service_role;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO anon, authenticated, service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO anon, authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT INSERT, UPDATE, DELETE ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO anon, authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO anon, authenticated, service_role;

GRANT USAGE, CREATE ON SCHEMA storage TO supabase_storage_admin;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA storage TO supabase_storage_admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA storage TO supabase_storage_admin;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO supabase_storage_admin;
GRANT USAGE ON SCHEMA storage TO anon, authenticated, service_role;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA storage TO authenticated, service_role;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO authenticated, service_role;

GRANT USAGE, CREATE ON SCHEMA realtime TO supabase_realtime_admin;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA realtime TO supabase_realtime_admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA realtime TO supabase_realtime_admin;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA realtime TO supabase_realtime_admin;
GRANT USAGE ON SCHEMA realtime TO authenticated, service_role;

GRANT USAGE ON SCHEMA pgbouncer TO pgbouncer;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA pgbouncer TO pgbouncer;
GRANT USAGE ON SCHEMA graphql_public TO anon, authenticated, service_role;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA graphql_public TO anon, authenticated, service_role;

DO $$
DECLARE
    table_record record;
BEGIN
    FOR table_record IN
        SELECT n.nspname, c.relname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'auth'
          AND c.relkind IN ('r', 'p')
          AND c.relrowsecurity
    LOOP
        EXECUTE format('ALTER TABLE %I.%I DISABLE ROW LEVEL SECURITY', table_record.nspname, table_record.relname);
    END LOOP;
END $$;
'@

Write-Output "Restored $dumpFile into local Supabase database."

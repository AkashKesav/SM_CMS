//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Database connection string from environment or default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://supabase_admin:postgres@127.0.0.1:54322/postgres"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to ping database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected to database successfully!")

	// Read and execute the migration SQL
	migrationSQL := `
-- Create junction tables for student-owned relationships

BEGIN;

-- Create is_admin() function if it doesn't exist
CREATE OR REPLACE FUNCTION public.is_admin()
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
AS $$
    SELECT EXISTS (
        SELECT 1 FROM public.profiles
        WHERE id = auth.uid()
        AND role = 'admin'
    );
$$;

-- Create achievement_members junction table
DROP TABLE IF EXISTS public.achievement_members CASCADE;
CREATE TABLE public.achievement_members (
    student_id uuid NOT NULL REFERENCES public.students(id) ON DELETE CASCADE,
    achievement_id uuid NOT NULL REFERENCES public.achievements(id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT now(),
    PRIMARY KEY (student_id, achievement_id)
);

-- Create project_members junction table
DROP TABLE IF EXISTS public.project_members CASCADE;
CREATE TABLE public.project_members (
    student_id uuid NOT NULL REFERENCES public.students(id) ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES public.projects(id) ON DELETE CASCADE,
    role text,
    created_at timestamptz DEFAULT now(),
    PRIMARY KEY (student_id, project_id)
);

-- Create student_tags junction table
DROP TABLE IF EXISTS public.student_tags CASCADE;
CREATE TABLE public.student_tags (
    student_id uuid NOT NULL REFERENCES public.students(id) ON DELETE CASCADE,
    tag_id uuid NOT NULL REFERENCES public.tags(id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT now(),
    PRIMARY KEY (student_id, tag_id)
);

-- Migrate existing single-student data from achievements.student_id to achievement_members
INSERT INTO public.achievement_members (student_id, achievement_id)
SELECT DISTINCT student_id, id
FROM public.achievements
WHERE student_id IS NOT NULL
ON CONFLICT (student_id, achievement_id) DO NOTHING;

COMMIT;
`

	// Split by statements and execute
	// Note: We'll execute the whole block at once since it's wrapped in BEGIN/COMMIT
	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migration applied successfully!")
	fmt.Println("Created tables: achievement_members, project_members, student_tags")
	fmt.Println("Created function: is_admin()")

	// Verify the tables exist
	var tableCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('achievement_members', 'project_members', 'student_tags')
	`).Scan(&tableCount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification failed: %v\n", err)
	} else {
		fmt.Printf("Verified: %d/3 junction tables exist\n", tableCount)
	}

	// Check how many achievement_members records exist
	var memberCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM public.achievement_members").Scan(&memberCount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to count achievement_members: %v\n", err)
	} else {
		fmt.Printf("Achievement members: %d records\n", memberCount)
	}
}

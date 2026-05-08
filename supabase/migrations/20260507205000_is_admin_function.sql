-- Migration: Create is_admin() helper function for RLS policies

BEGIN;

-- Drop existing function if it exists
DROP FUNCTION IF EXISTS public.is_admin() CASCADE;

-- Create the is_admin() function
-- This checks if the current authenticated user has admin role
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

-- Grant execute to authenticated users
GRANT EXECUTE ON FUNCTION public.is_admin() TO authenticated;

COMMIT;

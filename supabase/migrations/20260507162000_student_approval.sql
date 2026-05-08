BEGIN;

DO $$
BEGIN
  IF to_regclass('public.profiles') IS NULL OR to_regclass('public.students') IS NULL THEN
    RAISE NOTICE 'Skipping student approval migration because app tables are not restored yet.';
    RETURN;
  END IF;

  EXECUTE $sql$
    ALTER TABLE public.profiles
      DROP CONSTRAINT IF EXISTS profiles_role_check
  $sql$;

  EXECUTE $sql$
    ALTER TABLE public.profiles
      ADD CONSTRAINT profiles_role_check CHECK (role IN ('admin', 'student'))
  $sql$;

  EXECUTE $sql$
    ALTER TABLE public.profiles
      ADD COLUMN IF NOT EXISTS student_id uuid REFERENCES public.students(id) ON DELETE SET NULL
  $sql$;

  EXECUTE $sql$
    CREATE INDEX IF NOT EXISTS idx_profiles_student_id
      ON public.profiles(student_id)
  $sql$;

  EXECUTE $sql$
    CREATE TABLE IF NOT EXISTS public.student_change_requests (
      id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
      requester_user_id uuid NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
      student_id uuid REFERENCES public.students(id) ON DELETE SET NULL,
      request_type text NOT NULL CHECK (request_type IN ('claim_student', 'create', 'update', 'delete', 'link', 'unlink')),
      target_table text NOT NULL CHECK (
        target_table IN (
          'students',
          'achievements',
          'student_tags',
          'project_members',
          'hof_competition_members',
          'hof_coordinators',
          'hof_internships'
        )
      ),
      target_key jsonb,
      current_data jsonb,
      proposed_data jsonb NOT NULL DEFAULT '{}'::jsonb,
      status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
      admin_note text,
      reviewed_by uuid REFERENCES auth.users(id) ON DELETE SET NULL,
      reviewed_at timestamptz,
      created_at timestamptz NOT NULL DEFAULT now(),
      updated_at timestamptz NOT NULL DEFAULT now()
    )
  $sql$;

  EXECUTE $sql$
    CREATE INDEX IF NOT EXISTS idx_student_change_requests_requester
      ON public.student_change_requests(requester_user_id)
  $sql$;

  EXECUTE $sql$
    CREATE INDEX IF NOT EXISTS idx_student_change_requests_student
      ON public.student_change_requests(student_id)
  $sql$;

  EXECUTE $sql$
    CREATE INDEX IF NOT EXISTS idx_student_change_requests_status
      ON public.student_change_requests(status, created_at DESC)
  $sql$;

  EXECUTE $sql$
    ALTER TABLE public.student_change_requests ENABLE ROW LEVEL SECURITY
  $sql$;

  EXECUTE $sql$
    DROP POLICY IF EXISTS student_read_own_change_requests ON public.student_change_requests
  $sql$;

  EXECUTE $sql$
    CREATE POLICY student_read_own_change_requests
      ON public.student_change_requests
      FOR SELECT
      USING (requester_user_id = auth.uid())
  $sql$;

  EXECUTE $sql$
    DROP POLICY IF EXISTS student_create_own_change_requests ON public.student_change_requests
  $sql$;

  EXECUTE $sql$
    CREATE POLICY student_create_own_change_requests
      ON public.student_change_requests
      FOR INSERT
      WITH CHECK (requester_user_id = auth.uid())
  $sql$;

  EXECUTE $sql$
    DROP POLICY IF EXISTS admin_manage_student_change_requests ON public.student_change_requests
  $sql$;

  EXECUTE $sql$
    CREATE POLICY admin_manage_student_change_requests
      ON public.student_change_requests
      USING (public.is_admin())
      WITH CHECK (public.is_admin())
  $sql$;

  EXECUTE $sql$
    DROP POLICY IF EXISTS admin_manage_profiles ON public.profiles
  $sql$;

  EXECUTE $sql$
    CREATE POLICY admin_manage_profiles
      ON public.profiles
      USING (public.is_admin())
      WITH CHECK (public.is_admin())
  $sql$;

  EXECUTE $sql$
    DROP POLICY IF EXISTS user_can_read_own_profile ON public.profiles
  $sql$;

  EXECUTE $sql$
    CREATE POLICY user_can_read_own_profile
      ON public.profiles
      FOR SELECT
      USING (id = auth.uid())
  $sql$;

  EXECUTE $sql$
    DROP POLICY IF EXISTS user_can_read_linked_student ON public.students
  $sql$;

  EXECUTE $sql$
    CREATE POLICY user_can_read_linked_student
      ON public.students
      FOR SELECT
      USING (
        id IN (
          SELECT student_id
          FROM public.profiles
          WHERE profiles.id = auth.uid()
        )
      )
  $sql$;
END $$;

COMMIT;

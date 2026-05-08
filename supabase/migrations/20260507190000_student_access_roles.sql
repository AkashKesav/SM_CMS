BEGIN;

DO $$
BEGIN
  IF to_regclass('public.profiles') IS NULL OR to_regclass('public.students') IS NULL THEN
    RAISE NOTICE 'Skipping student access role migration because app tables are not restored yet.';
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

  UPDATE public.profiles p
  SET role = CASE
    WHEN lower(s.roll_no) IN ('23bsm006', '23bsm007') THEN 'admin'
    ELSE 'student'
  END
  FROM public.students s
  WHERE p.student_id = s.id;

  UPDATE public.profiles
  SET role = 'student'
  WHERE student_id IS NULL;
END $$;

COMMIT;

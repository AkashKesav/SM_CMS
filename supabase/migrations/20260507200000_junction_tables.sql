-- Migration: Create junction tables for student-owned relationships
-- These tables allow multiple students to be linked to achievements, projects, etc.

BEGIN;

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

-- Migrate existing single-student data from projects.student_id to project_members (if exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'projects' AND column_name = 'student_id') THEN
        INSERT INTO public.project_members (student_id, project_id)
        SELECT DISTINCT student_id, id
        FROM public.projects
        WHERE student_id IS NOT NULL
        ON CONFLICT (student_id, project_id) DO NOTHING;
    END IF;
END $$;

-- Enable Row Level Security
ALTER TABLE public.achievement_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.project_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.student_tags ENABLE ROW LEVEL SECURITY;

-- RLS Policies: Students can read their own links
CREATE POLICY students_read_own_achievement_members ON public.achievement_members
    FOR SELECT USING (student_id IN (SELECT student_id FROM public.profiles WHERE id = auth.uid()));

CREATE POLICY students_read_own_project_members ON public.project_members
    FOR SELECT USING (student_id IN (SELECT student_id FROM public.profiles WHERE id = auth.uid()));

CREATE POLICY students_read_own_student_tags ON public.student_tags
    FOR SELECT USING (student_id IN (SELECT student_id FROM public.profiles WHERE id = auth.uid()));

-- RLS Policies: Admins can manage everything
CREATE POLICY admins_manage_achievement_members ON public.achievement_members
    USING (public.is_admin()) WITH CHECK (public.is_admin());

CREATE POLICY admins_manage_project_members ON public.project_members
    USING (public.is_admin()) WITH CHECK (public.is_admin());

CREATE POLICY admins_manage_student_tags ON public.student_tags
    USING (public.is_admin()) WITH CHECK (public.is_admin());

COMMIT;

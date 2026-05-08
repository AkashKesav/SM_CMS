-- 1. Create the junction table
CREATE TABLE IF NOT EXISTS public.achievement_members (
    achievement_id UUID REFERENCES public.achievements(id) ON DELETE CASCADE,
    student_id UUID REFERENCES public.students(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (achievement_id, student_id)
);

-- 2. Migrate existing data
INSERT INTO public.achievement_members (achievement_id, student_id)
SELECT id, student_id FROM public.achievements WHERE student_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- 3. RLS
ALTER TABLE public.achievement_members ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS admin_manage_members ON public.achievement_members;
CREATE POLICY admin_manage_members ON public.achievement_members FOR ALL TO authenticated USING (true) WITH CHECK (true);

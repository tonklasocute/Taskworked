-- Reverses 0003_organization_constraints.up.sql.

ALTER TABLE public.projects DROP CONSTRAINT IF EXISTS fk_projects_organization;
ALTER TABLE public.departments DROP CONSTRAINT IF EXISTS fk_departments_organization;
ALTER TABLE public.users DROP CONSTRAINT IF EXISTS fk_users_organization;

ALTER TABLE public.projects ALTER COLUMN organization_id DROP NOT NULL;
ALTER TABLE public.departments ALTER COLUMN organization_id DROP NOT NULL;
ALTER TABLE public.users ALTER COLUMN organization_id DROP NOT NULL;

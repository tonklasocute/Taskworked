-- P1.1: introduces the organizations table and backfills every existing
-- user/department/project into one default organization.
--
-- Deliberately stops at nullable columns + indexes, no NOT NULL / foreign
-- keys / uniqueness-scope changes yet — those are a separate follow-up
-- migration pending a product decision on whether users.email should
-- remain globally unique or become unique-per-organization. See
-- docs/superpowers/specs/2026-08-10-p1-organization-architecture-audit.md
-- §9 for the full reasoning. No authorization behavior changes with this
-- migration — organization_id exists but nothing reads it yet.

CREATE TABLE public.organizations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    description text,
    logo text,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.organizations ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);
CREATE UNIQUE INDEX idx_organizations_slug ON public.organizations USING btree (slug);

ALTER TABLE public.users ADD COLUMN organization_id uuid;
ALTER TABLE public.departments ADD COLUMN organization_id uuid;
ALTER TABLE public.projects ADD COLUMN organization_id uuid;

CREATE INDEX idx_users_organization_id ON public.users USING btree (organization_id);
CREATE INDEX idx_departments_organization_id ON public.departments USING btree (organization_id);
CREATE INDEX idx_projects_organization_id ON public.projects USING btree (organization_id);

-- Backfill: exactly one default organization, at a fixed well-known ID so
-- this migration is deterministic and the ID can be referenced as a
-- constant in Go code/tests if ever needed. Rename it (a normal UPDATE
-- against the running database, not a migration) once the real deployment's
-- company name is known — this migration can't know that.
INSERT INTO public.organizations (id, name, slug, status)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Organization', 'default', 'active');

UPDATE public.users SET organization_id = '00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;
UPDATE public.departments SET organization_id = '00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;
UPDATE public.projects SET organization_id = '00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;

-- Consistency check: fail the migration loudly (rolling back the whole
-- transaction) if the backfill somehow missed a row, rather than silently
-- leaving NULLs for a later, harder-to-diagnose bug.
DO $$
DECLARE
    missing_users integer;
    missing_departments integer;
    missing_projects integer;
BEGIN
    SELECT count(*) INTO missing_users FROM public.users WHERE organization_id IS NULL;
    SELECT count(*) INTO missing_departments FROM public.departments WHERE organization_id IS NULL;
    SELECT count(*) INTO missing_projects FROM public.projects WHERE organization_id IS NULL;

    IF missing_users > 0 OR missing_departments > 0 OR missing_projects > 0 THEN
        RAISE EXCEPTION 'organization backfill incomplete: % users, % departments, % projects still have NULL organization_id',
            missing_users, missing_departments, missing_projects;
    END IF;
END $$;

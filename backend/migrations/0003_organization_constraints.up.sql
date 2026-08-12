-- P1.2: tightens the nullable organization_id columns migration 0002
-- introduced into a real, enforced foreign key relationship, now that
-- every row has been confirmed backfilled (migration 0002's own
-- consistency check already guarantees this — see that file) and every
-- code path that creates a new user/department/project now stamps
-- organization_id at creation time (see the P1.2 tenant isolation audit).
--
-- Deliberately does NOT touch users.email's uniqueness scope
-- (still globally unique, not per-organization) — that's a separate,
-- still-open product decision (single account per email across all
-- organizations, vs. per-organization accounts), independent of whether
-- organization_id itself is required and referentially valid. See
-- docs/superpowers/specs/2026-08-10-p1-organization-architecture-audit.md §9.

ALTER TABLE public.users ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE public.departments ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE public.projects ALTER COLUMN organization_id SET NOT NULL;

ALTER TABLE public.users
    ADD CONSTRAINT fk_users_organization
    FOREIGN KEY (organization_id) REFERENCES public.organizations(id);

ALTER TABLE public.departments
    ADD CONSTRAINT fk_departments_organization
    FOREIGN KEY (organization_id) REFERENCES public.organizations(id);

ALTER TABLE public.projects
    ADD CONSTRAINT fk_projects_organization
    FOREIGN KEY (organization_id) REFERENCES public.organizations(id);

-- Reverses 0002_organizations.up.sql. Dropping the columns discards the
-- backfilled organization_id values — acceptable for a rollback of this
-- migration specifically, since nothing else depends on that data yet (no
-- authorization behavior reads organization_id as of this migration).

ALTER TABLE public.projects DROP COLUMN IF EXISTS organization_id;
ALTER TABLE public.departments DROP COLUMN IF EXISTS organization_id;
ALTER TABLE public.users DROP COLUMN IF EXISTS organization_id;
DROP TABLE IF EXISTS public.organizations;

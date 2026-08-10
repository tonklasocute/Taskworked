-- Reverses 0001_baseline.up.sql. No FK constraints exist yet at this point
-- in the migration history, so drop order doesn't matter.

DROP TABLE IF EXISTS public.watchers;
DROP TABLE IF EXISTS public.users;
DROP TABLE IF EXISTS public.tasks;
DROP TABLE IF EXISTS public.tags;
DROP TABLE IF EXISTS public.projects;
DROP TABLE IF EXISTS public.preferences;
DROP TABLE IF EXISTS public.notifications;
DROP TABLE IF EXISTS public.mission_progresses;
DROP TABLE IF EXISTS public.milestones;
DROP TABLE IF EXISTS public.members;
DROP TABLE IF EXISTS public.goals;
DROP TABLE IF EXISTS public.dependencies;
DROP TABLE IF EXISTS public.departments;
DROP TABLE IF EXISTS public.comments;
DROP TABLE IF EXISTS public.checklist_items;
DROP TABLE IF EXISTS public.characters;
DROP TABLE IF EXISTS public.badges;
DROP TABLE IF EXISTS public.attachments;

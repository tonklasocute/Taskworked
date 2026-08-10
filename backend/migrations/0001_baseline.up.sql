-- Baseline migration: recreates the exact schema this project has been
-- running via GORM AutoMigrate up to this point (18 tables, no FK
-- constraints, no organization concept yet). Generated from
-- `pg_dump --schema-only` against a database whose schema was produced by
-- bootstrap.Migrate's AutoMigrate call, not hand-transcribed from the Go
-- model tags, so this is a faithful snapshot of what's actually been
-- running in production/dev up to now.
--
-- Everything below this point is a genuinely new database: from here on,
-- schema changes ship as new numbered migration files, never AutoMigrate.

CREATE TABLE public.attachments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    uploader_id uuid NOT NULL,
    file_name text NOT NULL,
    content_type text NOT NULL,
    size_bytes bigint NOT NULL,
    object_key text NOT NULL,
    created_at timestamp with time zone
);

CREATE TABLE public.badges (
    user_id uuid NOT NULL,
    code character varying(30) NOT NULL,
    awarded_at timestamp with time zone
);

CREATE TABLE public.characters (
    user_id uuid NOT NULL,
    exp bigint DEFAULT 0 NOT NULL,
    level bigint DEFAULT 1 NOT NULL,
    total_completed bigint DEFAULT 0 NOT NULL,
    current_streak bigint DEFAULT 0 NOT NULL,
    longest_streak bigint DEFAULT 0 NOT NULL,
    last_completion_date timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE TABLE public.checklist_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    text text NOT NULL,
    done boolean DEFAULT false NOT NULL,
    "position" bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone
);

CREATE TABLE public.comments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    author_id uuid NOT NULL,
    body text NOT NULL,
    created_at timestamp with time zone
);

CREATE TABLE public.departments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone
);

CREATE TABLE public.dependencies (
    task_id uuid NOT NULL,
    depends_on_id uuid NOT NULL,
    created_at timestamp with time zone
);

CREATE TABLE public.goals (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    status character varying(20) DEFAULT 'not_started'::character varying NOT NULL,
    due_date timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE TABLE public.members (
    project_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role character varying(20) DEFAULT 'member'::character varying NOT NULL,
    created_at timestamp with time zone
);

CREATE TABLE public.milestones (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    goal_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    status character varying(20) DEFAULT 'not_started'::character varying NOT NULL,
    due_date timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE TABLE public.mission_progresses (
    user_id uuid NOT NULL,
    type character varying(10) NOT NULL,
    period_key text NOT NULL,
    count bigint DEFAULT 0 NOT NULL,
    rewarded boolean DEFAULT false NOT NULL,
    updated_at timestamp with time zone
);

CREATE TABLE public.notifications (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    type character varying(30) NOT NULL,
    title text NOT NULL,
    body text,
    link text,
    read boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone
);

CREATE TABLE public.preferences (
    user_id uuid NOT NULL,
    email_enabled boolean DEFAULT true NOT NULL,
    line_enabled boolean DEFAULT false NOT NULL,
    line_notify_token text
);

CREATE TABLE public.projects (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    description text,
    owner_id uuid NOT NULL,
    status character varying(20) DEFAULT 'planning'::character varying NOT NULL,
    due_date timestamp with time zone,
    color text,
    icon text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE TABLE public.tags (
    task_id uuid NOT NULL,
    tag text NOT NULL
);

CREATE TABLE public.tasks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    parent_task_id uuid,
    title text NOT NULL,
    description text,
    priority character varying(20) DEFAULT 'medium'::character varying NOT NULL,
    status character varying(20) DEFAULT 'backlog'::character varying NOT NULL,
    start_date timestamp with time zone,
    due_date timestamp with time zone,
    estimate_hours numeric,
    assignee_id uuid,
    reporter_id uuid NOT NULL,
    milestone_id uuid,
    completed_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    role character varying(20) DEFAULT 'employee'::character varying NOT NULL,
    avatar_url text,
    department_id uuid,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE TABLE public.watchers (
    task_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp with time zone
);

ALTER TABLE ONLY public.attachments ADD CONSTRAINT attachments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.badges ADD CONSTRAINT badges_pkey PRIMARY KEY (user_id, code);
ALTER TABLE ONLY public.characters ADD CONSTRAINT characters_pkey PRIMARY KEY (user_id);
ALTER TABLE ONLY public.checklist_items ADD CONSTRAINT checklist_items_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.comments ADD CONSTRAINT comments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.departments ADD CONSTRAINT departments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.dependencies ADD CONSTRAINT dependencies_pkey PRIMARY KEY (task_id, depends_on_id);
ALTER TABLE ONLY public.goals ADD CONSTRAINT goals_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.members ADD CONSTRAINT members_pkey PRIMARY KEY (project_id, user_id);
ALTER TABLE ONLY public.milestones ADD CONSTRAINT milestones_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.mission_progresses ADD CONSTRAINT mission_progresses_pkey PRIMARY KEY (user_id, type, period_key);
ALTER TABLE ONLY public.notifications ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.preferences ADD CONSTRAINT preferences_pkey PRIMARY KEY (user_id);
ALTER TABLE ONLY public.projects ADD CONSTRAINT projects_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tags ADD CONSTRAINT tags_pkey PRIMARY KEY (task_id, tag);
ALTER TABLE ONLY public.tasks ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.watchers ADD CONSTRAINT watchers_pkey PRIMARY KEY (task_id, user_id);

CREATE INDEX idx_attachments_task_id ON public.attachments USING btree (task_id);
CREATE INDEX idx_checklist_items_task_id ON public.checklist_items USING btree (task_id);
CREATE INDEX idx_comments_task_id ON public.comments USING btree (task_id);
CREATE UNIQUE INDEX idx_departments_name ON public.departments USING btree (name);
CREATE INDEX idx_goals_project_id ON public.goals USING btree (project_id);
CREATE INDEX idx_milestones_goal_id ON public.milestones USING btree (goal_id);
CREATE INDEX idx_notifications_user_id ON public.notifications USING btree (user_id);
CREATE INDEX idx_projects_owner_id ON public.projects USING btree (owner_id);
CREATE INDEX idx_tasks_assignee_id ON public.tasks USING btree (assignee_id);
CREATE INDEX idx_tasks_milestone_id ON public.tasks USING btree (milestone_id);
CREATE INDEX idx_tasks_parent_task_id ON public.tasks USING btree (parent_task_id);
CREATE INDEX idx_tasks_project_id ON public.tasks USING btree (project_id);
CREATE INDEX idx_users_department_id ON public.users USING btree (department_id);
CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);

CREATE EXTENSION IF NOT EXISTS pg_trgm;

--------------------------------------------------------------------------------
--- USERS ----------------------------------------------------------------------
--------------------------------------------------------------------------------

CREATE TYPE user_role AS ENUM (
    'examiner',
    'examinee',
    'admin'
);

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(100) UNIQUE NOT NULL CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    first_name      VARCHAR(50) NOT NULL CHECK (first_name <> ''),
    last_name       VARCHAR(50) NOT NULL CHECK (last_name <> ''),
    password_hash   TEXT NOT NULL CHECK (password_hash <> ''),
    role            user_role NOT NULL,

    UNIQUE (id, role)
);

CREATE TABLE examiners (
    id      UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    role    user_role NOT NULL DEFAULT 'examiner' CHECK (role = 'examiner'),
    FOREIGN KEY (id, role) REFERENCES users(id, role)
);

CREATE TABLE examinees (
    id      UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    role    user_role NOT NULL DEFAULT 'examinee' CHECK (role = 'examinee'),
    FOREIGN KEY (id, role) REFERENCES users(id, role)
);

--------------------------------------------------------------------------------



--------------------------------------------------------------------------------
--- SCRIPTS --------------------------------------------------------------------
--------------------------------------------------------------------------------

CREATE TABLE scripts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title               VARCHAR(200) NOT NULL CHECK (title <> ''),
    heading             VARCHAR(200) NOT NULL CHECK (heading <> ''),
    description         VARCHAR(800) NOT NULL,
    locked              BOOLEAN DEFAULT FALSE,

    default_mark    INTEGER NOT NULL DEFAULT 1 CHECK (default_mark >= 1),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_modified_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    creator_id UUID NOT NULL REFERENCES examiners(id)
);

-- Indexes for quick searching scripts

CREATE INDEX idx_scripts_creator_id ON scripts(creator_id);
CREATE INDEX idx_scripts_title_trgm ON scripts USING gin(title gin_trgm_ops);
CREATE INDEX idx_scripts_created_at ON scripts(created_at DESC, id DESC);
CREATE INDEX idx_scripts_last_modified_at ON scripts(last_modified_at DESC, id DESC);

-- For updating its last_modified_at
-- when fields change (e.g. title, heading, description)

CREATE TRIGGER trg_scripts_update_set_last_modified_at
BEFORE UPDATE ON scripts
FOR EACH ROW
EXECUTE FUNCTION set_last_modified_at();

-----------------------------------------------
--- QUESTIONS ---------------------------------
-----------------------------------------------

CREATE TYPE question_type AS ENUM (
    'choice',
    'text'
);

CREATE TABLE questions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    text        TEXT NOT NULL,
    image_url   TEXT,
    type        question_type NOT NULL,

    mark    INTEGER DEFAULT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    script_id UUID NOT NULL REFERENCES scripts(id) ON DELETE CASCADE,
    UNIQUE (id, type)
);

CREATE INDEX idx_questions_created_at ON questions(created_at ASC);

-- Answer keys just represent a list of values
-- for a question that can be marked as the answer.

CREATE TABLE answer_keys (
    value       TEXT NOT NULL CHECK (value <> ''),
    question_id UUID REFERENCES questions(id) ON DELETE CASCADE,
    PRIMARY KEY (question_id, value)
);

-------------------------
--- Choice Questions ----
-------------------------

-- To ensure mutual exclusivity of subtypes, we create
-- a "constant" unique type to the subquestion that doesn't need
-- to be filled, and combine that with the primary key to
-- reference the parent table (id & type)

CREATE TABLE choice_questions (
    id                  UUID PRIMARY KEY REFERENCES questions(id) ON DELETE CASCADE,
    is_multiple_choice  BOOLEAN NOT NULL DEFAULT TRUE,

    type question_type NOT NULL DEFAULT 'choice' CHECK (type = 'choice'),
    FOREIGN KEY (id, type) REFERENCES questions(id, type)
);

CREATE TABLE options (
    value       TEXT NOT NULL CHECK (value <> ''),
    question_id UUID NOT NULL REFERENCES choice_questions(id) ON DELETE CASCADE,
    image_url   TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (question_id, value)
);

CREATE INDEX idx_options_created_at ON options(created_at ASC);

----------------------
--- Text Questions ---
----------------------

-- To ensure mutual exclusivity of subtypes, we create
-- a "constant" unique type to the subquestion that doesn't need
-- to be filled, and combine that with the primary key to
-- reference the parent table (id & type)

CREATE TABLE text_questions (
    id              UUID PRIMARY KEY REFERENCES questions(id) ON DELETE CASCADE,
    is_short_text   BOOLEAN NOT NULL DEFAULT TRUE,

    type question_type NOT NULL DEFAULT 'text' CHECK (type = 'text'),
    FOREIGN KEY (id, type) REFERENCES questions(id, type)
);

--------------------------------------------------------------------------------

-- For Updating scripts.last_modified_at whenever
-- a sub entity is changed

CREATE TRIGGER trg_questions_update_scripts_on_change
AFTER INSERT OR UPDATE OR DELETE ON questions
FOR EACH ROW
EXECUTE FUNCTION update_scripts_last_modified_at();

CREATE TRIGGER trg_answer_keys_update_scripts_on_change
AFTER INSERT OR UPDATE OR DELETE ON answer_keys
FOR EACH ROW
EXECUTE FUNCTION update_scripts_last_modified_at();

CREATE TRIGGER trg_choice_questions_update_scripts_on_change
AFTER INSERT OR UPDATE OR DELETE ON choice_questions
FOR EACH ROW
EXECUTE FUNCTION update_scripts_last_modified_at();

CREATE TRIGGER trg_text_questions_update_scripts_on_change
AFTER INSERT OR UPDATE OR DELETE ON text_questions
FOR EACH ROW
EXECUTE FUNCTION update_scripts_last_modified_at();

CREATE TRIGGER trg_options_update_scripts_on_change
AFTER INSERT OR UPDATE OR DELETE ON options
FOR EACH ROW
EXECUTE FUNCTION update_scripts_last_modified_at();

--------------------------------------------------------------------------------



--------------------------------------------------------------------------------
--- SESSIONS -------------------------------------------------------------------
--------------------------------------------------------------------------------

CREATE TYPE session_status AS ENUM (
    'closed',
    'open',
    'started',
    'ended'
);

-- CREATE TABLE session_schedules (
--     id              UUID PRIMARY KEY REFERENCES sessions(id),
--     start_time      TIMESTAMPTZ,
--     duration_mins   INTEGER CHECK (duration_mins > 0),

--     start_task_id   TEXT,
--     end_task_id     TEXT
-- );

CREATE TABLE sessions (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title   VARCHAR(200) NOT NULL,
    status  session_status NOT NULL DEFAULT 'closed',

    started_at          TIMESTAMPTZ DEFAULT NULL,
    ended_at            TIMESTAMPTZ DEFAULT NULL,
    join_code           VARCHAR(10) NOT NULL UNIQUE,
    allow_any_examinee  BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    creator_id  UUID REFERENCES examiners(id) NOT NULL,
    script_id   UUID REFERENCES scripts(id) NOT NULL
);

CREATE INDEX idx_sessions_created_at ON sessions(created_at DESC);

--------------------------------------------------------------------------------

CREATE TABLE proctor_event_types (
    name TEXT PRIMARY KEY NOT NULL
);

INSERT INTO proctor_event_types
    (name)
VALUES
    ('APP_STATE_CHANGE'),
    ('EXTENDED_DISCONNECT'),
    ('REPEATED_DISCONNECT')
;

CREATE TABLE proctor_events (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type    TEXT NOT NULL REFERENCES proctor_event_types(name),

    occurred_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    session_id  UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    examinee_id UUID NOT NULL REFERENCES examinees(id) ON DELETE CASCADE
);

--------------------------------------------------------------------------------


--------------------------------------------------------------------------------
--- SUBMISSIONS ----------------------------------------------------------------
--------------------------------------------------------------------------------

CREATE TYPE submission_status AS ENUM (
    'enrolled',
    'joined',
    'left',
    'editable',
    'disconnected',
    'flagged',
    'submitted',
    'unreviewed',
    'marked'
);

CREATE TABLE submissions (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status  submission_status NOT NULL DEFAULT 'enrolled',

    submitted_at    TIMESTAMPTZ,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    examinee_id UUID REFERENCES examinees(id) ON DELETE CASCADE NOT NULL,
    session_id  UUID REFERENCES sessions(id) ON DELETE CASCADE NOT NULL,
    UNIQUE (examinee_id, session_id)
);

-- A submission question is an insurance that any answer added is
-- actually valid i.e. the script attached to the session for the
-- submission actually has the question being answered.
--
-- The foreign key constraint ensures that once the questions have
-- been preset for a submission, answers attached will always be for
-- that question.
--
--------------------------------------------------------------------
--
-- SubmissionQuestion --------------| AnswerValue
-- -> created_at: TZ                | AnswerValue
-- -> mark: INTEGER                 | AnswerValue
--                                  | AnswerValue
--                                  | AnswerValue

CREATE TABLE submission_questions (
    submission_id   UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    question_id     UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,

    mark        INTEGER DEFAULT NULL CHECK (mark >= 0),
    feedback    TEXT NOT NULL DEFAULT '',

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (submission_id, question_id)
);

CREATE INDEX idx_sq_created_at ON submission_questions(created_at ASC);

CREATE TABLE answer_values (
    value           TEXT NOT NULL,
    submission_id   UUID NOT NULL,
    question_id     UUID NOT NULL,

    PRIMARY KEY (submission_id, question_id, value),
    FOREIGN KEY (submission_id, question_id)
        REFERENCES submission_questions(submission_id, question_id) ON DELETE CASCADE
);

--------------------------------------------------------------------------------


-- Creates a new script
-----------------------
-- name: CreateScript :one

INSERT INTO scripts (
    title,
    heading,
    description,
    creator_id
) VALUES ($1, $2, $3, $4) RETURNING *;


-- Updates the script if possible
---------------------------------
-- name: UpdateScriptFields :exec

UPDATE scripts SET
    title       = $2,
    heading     = $3,
    description = $4
WHERE scripts.id = $1
    AND scripts.locked = false;


-- Deletes the script if possible
---------------------------------
-- name: DeleteScript :exec

DELETE FROM scripts
WHERE scripts.id = $1
    AND scripts.locked = false;

-----------------------------------------------------------------
--- Finding Scripts ---------------------------------------------
-----------------------------------------------------------------

-- Gets the scripts belonging to the examiner
-- with cursor pagination
---------------------------------------------
-- name: FindScriptsForExaminer :many

SELECT * FROM scripts
WHERE scripts.creator_id = @examiner_id::UUID
    AND (scripts.last_modified_at, scripts.id) < (@cursor_ts::TIMESTAMPTZ, @cursor_id::UUID)
ORDER BY scripts.last_modified_at DESC, scripts.id DESC
LIMIT @page_size;


-- Finds the number of scripts belonging to the
-- examiner
-- --------------------------------------------
-- name: FindScriptCountForExaminer :one

SELECT COUNT(*) FROM scripts
WHERE scripts.creator_id = $1;

-----------------------------------------------------------------
-----------------------------------------------------------------

-- Gets the scripts belonging to the examiner
-- with cursor pagination and a search query
---------------------------------------------
-- name: SearchScriptsForExaminer :many

SELECT * FROM scripts
WHERE scripts.creator_id = @examiner_id::UUID
    AND scripts.title ILIKE '%' || @search::TEXT || '%'
    AND (scripts.last_modified_at, scripts.id) < (@cursor_ts::TIMESTAMPTZ, @cursor_id::UUID)
ORDER BY scripts.last_modified_at DESC, scripts.id DESC
LIMIT @page_size;


-- Finds the number of scripts belonging to the
-- examiner with the search query
-- --------------------------------------------
-- name: SearchScriptCountForExaminer :one

SELECT COUNT(*) FROM scripts
WHERE
    scripts.creator_id = @examiner_id::UUID
    AND scripts.title ILIKE '%' || @search::TEXT || '%';

-----------------------------------------------------------------
-----------------------------------------------------------------

-- Finds a script by its ID
---------------------------
-- name: FindScriptByID :one

SELECT * FROM scripts
WHERE scripts.id = $1;

-----------------------------------------------------------------
--- Answer Key --------------------------------------------------
-----------------------------------------------------------------

-- Fully replace all answer key values for the
-- question with the given ID
----------------------------------------------
-- name: ReplaceAnswerKeyForQuestion :exec

WITH deleted AS (
    DELETE FROM answer_keys
    WHERE answer_keys.question_id = $1
) INSERT INTO answer_keys (
    question_id,
    value
) SELECT $1, unnest(@answer_key::text[]) FROM
    questions INNER JOIN scripts
    ON questions.script_id = scripts.id
WHERE questions.id = $1
    AND scripts.id = questions.script_id
    AND scripts.locked = false;


-- Finds the answer key for a question as a list
------------------------------------------------
-- name: FindAnswerKeyForQuestion :many

SELECT answer_keys.value FROM answer_keys
WHERE answer_keys.question_id = $1;

-----------------------------------------------------------------
--- Creating Questions ------------------------------------------
-----------------------------------------------------------------

-- Creates a new choice question
--------------------------------
-- name: CreateChoiceQuestion :one

WITH inserted AS (
    INSERT INTO questions (
        script_id,
        type,
        text,
        image_url
    ) SELECT @script_id::uuid, 'choice', $1, $2 FROM scripts
    WHERE scripts.id = @script_id::uuid
        AND scripts.locked = false
    RETURNING *
) INSERT INTO choice_questions (
    id,
    is_multiple_choice
) SELECT inserted.id, $3 FROM
    inserted INNER JOIN scripts ON inserted.script_id = scripts.id
WHERE scripts.id = @script_id::uuid
AND scripts.locked = false
RETURNING id;


-- Creates a new text question
------------------------------
-- name: CreateTextQuestion :one

WITH inserted AS (
    INSERT INTO questions (
        script_id,
        type,
        text,
        image_url
    ) SELECT @script_id::uuid, 'text', $1, $2 FROM scripts
    WHERE scripts.id = @script_id::uuid
        AND scripts.locked = false
    RETURNING *
) INSERT INTO text_questions (
    id,
    is_short_text
) SELECT inserted.id, $3 FROM
    inserted INNER JOIN scripts ON inserted.script_id = scripts.id
WHERE scripts.id = @script_id::uuid
AND scripts.locked = false
RETURNING id;


-----------------------------------------------------------------
--- Manipulating Questions --------------------------------------
-----------------------------------------------------------------

-- Updates the parent question fields if possible.
--------------------------------------------------
-- name: UpdateQuestionFields :exec

UPDATE questions SET
    text        = $2,
    image_url   = $3
FROM scripts
WHERE questions.id = $1
    AND scripts.id = questions.script_id
    AND scripts.locked = false;


-- Deletes the question if possible.
-- cascades to child components (e.g. subquestions & options)
-------------------------------------------------------------
-- name: DeleteQuestion :exec

DELETE FROM questions
USING scripts
WHERE questions.id = $1
    AND scripts.id = questions.script_id
    AND scripts.locked = false;

-----------------------------------------------------------------
-----------------------------------------------------------------

-- Insert text-question values unless there is conflict,
-- in which case update the existing subquestion.
--
-- Inserting a new subquestion in this case will automatically
-- delete all other subquestions & change the type of the
-- parent question to maintain FK constraints.
---------------------------------------------------------------
-- name: UpsertTextQuestion :exec

WITH deleted AS (
    SELECT delete_all_subquestions($1)
), updated_type AS (
    UPDATE questions SET
        type = 'text'
    WHERE questions.id = $1
    RETURNING questions.id
) INSERT INTO text_questions (
    id,
    is_short_text
) SELECT $1, $2 FROM
    updated_type
    INNER JOIN questions ON questions.id = updated_type.id
    INNER JOIN scripts ON questions.script_id = scripts.id
    CROSS JOIN deleted
WHERE questions.id = $1
    AND scripts.locked = false
ON CONFLICT (id) DO UPDATE SET
    is_short_text = EXCLUDED.is_short_text;


-- Insert text-question values unless there is conflict,
-- in which case update the existing subquestion.
--
-- Inserting a new subquestion in this case will automatically
-- delete all other subquestions & change the type of the
-- parent question to maintain FK constraints.
---------------------------------------------------------------
-- name: UpsertChoiceQuestion :exec

WITH deleted AS (
    SELECT delete_all_subquestions($1)
), updated_type AS (
    UPDATE questions SET
        type = 'choice'
    WHERE questions.id = $1
    RETURNING questions.id
) INSERT INTO choice_questions (
    id,
    is_multiple_choice
) SELECT $1, $2 FROM
    updated_type
    INNER JOIN questions ON questions.id = updated_type.id
    INNER JOIN scripts ON questions.script_id = scripts.id
    CROSS JOIN deleted
WHERE questions.id = $1
    AND scripts.locked = false
ON CONFLICT (id) DO UPDATE SET
    is_multiple_choice = EXCLUDED.is_multiple_choice;

-----------------------------------------------------------------
--- Querying Questions ------------------------------------------
-----------------------------------------------------------------

-- Gets all the questions for a script ordered
-- by time of creation
----------------------------------------------
-- name: FindQuestionsForScript :many

SELECT questions.* FROM
    scripts INNER JOIN questions on scripts.id = questions.script_id
WHERE scripts.id = $1
ORDER BY questions.created_at ASC;


-- Gets all the questions for a script as shuffled
--------------------------------------------------
-- name: FindQuestionsForScriptShuffled :many

SELECT questions.* FROM
    scripts INNER JOIN questions ON scripts.id = questions.script_id
WHERE scripts.id = $1
ORDER BY random();


-- Gets a single question
-------------------------
-- name: FindQuestionByID :one

SELECT * FROM questions
WHERE questions.id = $1;

-----------------------------------------------------------------
-----------------------------------------------------------------

-- name: FindChoiceQuestionByID :one
SELECT * FROM choice_questions
WHERE choice_questions.id = $1;

-- name: FindTextQuestionByID :one
SELECT * FROM text_questions WHERE text_questions.id = $1;

-----------------------------------------------------------------
--- Options -----------------------------------------------------
-----------------------------------------------------------------

-- Fully replace all options with arrays of new values.
-- Arrays passed are "zipped" together using unnest.
-- Will not work with other subquestion types thanks to the
-- foreign key constraints.
-----------------------------------------------------------
-- name: ReplaceOptionsForQuestion :exec

WITH deleted AS (
    DELETE FROM options
    WHERE options.question_id = $1
) INSERT INTO options (
    question_id,
    value,
    image_url
) SELECT $1, unnest(@values::text[]), nullif(unnest(@image_urls::text[]), '') FROM
    questions INNER JOIN scripts
    ON questions.script_id = scripts.id
WHERE questions.id = $1
    AND scripts.id = questions.script_id
    AND scripts.locked = false;

-----------------------------------------------------------------
--- Querying Options --------------------------------------------
-----------------------------------------------------------------

-- Finds options for a question order by creation.
-- Used for obtaining options for a question for editing
--------------------------------------------------------
-- name: FindOptionsForQuestion :many
SELECT * FROM options
    WHERE options.question_id = $1
    ORDER BY options.created_at ASC;


-- Finds options for a question unordered. Used for
-- obtaining options for a question during an exam session
----------------------------------------------------------
-- name: FindOptionsForQuestionShuffled :many
SELECT * FROM options
    WHERE options.question_id = $1
    ORDER BY random();

-----------------------------------------------------------------
-----------------------------------------------------------------

-- Creates a new choice question
--
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
)
INSERT INTO choice_questions (
    id,
    is_multiple_choice
) SELECT inserted.id, $3 FROM
    inserted INNER JOIN scripts ON inserted.script_id = scripts.id
WHERE scripts.id = @script_id::uuid
AND scripts.locked = false
RETURNING id;


-- Creates a new text question
--
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
)
INSERT INTO text_questions (
    id,
    is_short_text
) SELECT inserted.id, $3 FROM
    inserted INNER JOIN scripts ON inserted.script_id = scripts.id
WHERE scripts.id = @script_id::uuid
AND scripts.locked = false
RETURNING id;




-- Fully replace all answer key values for the
-- question with the given ID
--
----------------------------------------------
-- name: ReplaceAnswerKeyForQuestion :exec

WITH deleted AS (
    DELETE FROM answer_keys
    WHERE answer_keys.question_id = $1
    RETURNING 1
)
INSERT INTO answer_keys (
    question_id,
    value
) SELECT $1, unnest(@answer_key::text[]) FROM
    questions INNER JOIN scripts ON questions.script_id = scripts.id
    CROSS JOIN (SELECT count(*) FROM deleted) as _dep
WHERE questions.id = $1
    AND scripts.id = questions.script_id
    AND scripts.locked = false;


-- Finds the answer key for a question as a list
--
------------------------------------------------
-- name: FindAnswerKeyForQuestion :many

SELECT answer_keys.value FROM answer_keys
WHERE answer_keys.question_id = $1;




-- Updates the parent question fields if possible.
--
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
--
-------------------------------------------------------------
-- name: DeleteQuestion :exec

DELETE FROM questions
USING scripts
WHERE questions.id = $1
    AND scripts.id = questions.script_id
    AND scripts.locked = false;



-- Insert text-question values unless there is conflict,
-- in which case update the existing subquestion.
--
-- Inserting a new subquestion in this case will automatically
-- delete all other subquestions & change the type of the
-- parent question to maintain FK constraints.
--
---------------------------------------------------------------
-- name: UpsertTextQuestion :exec

WITH
deleted AS (
    SELECT delete_all_subquestions($1)
),
updated_type AS (
    UPDATE questions SET
        type = 'text'
    WHERE questions.id = $1
    RETURNING questions.id
)
INSERT INTO text_questions (
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
--
---------------------------------------------------------------
-- name: UpsertChoiceQuestion :exec

WITH
deleted AS (
    SELECT delete_all_subquestions($1)
),
updated_type AS (
    UPDATE questions SET
        type = 'choice'
    WHERE questions.id = $1
    RETURNING questions.id
)
INSERT INTO choice_questions (
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




-- Gets all the questions for a script ordered
-- by time of creation
--
----------------------------------------------
-- name: FindQuestionsForScript :many

SELECT questions.* FROM
    scripts INNER JOIN questions on scripts.id = questions.script_id
WHERE scripts.id = $1
ORDER BY questions.created_at ASC;


-- Returns all question fields assigned for an examinee.
-- Combine with FindOptionsForQuestionShuffled
--
--------------------------------------------------------
-- name: FindQuestionsForSubmission :many

SELECT questions.* FROM
    questions INNER JOIN submission_questions sq ON questions.id = sq.question_id
WHERE sq.submission_id = $1
ORDER BY sq.created_at DESC;


-- Gets a single question
--
-------------------------
-- name: FindQuestionByID :one

SELECT * FROM questions
WHERE questions.id = $1;



-- name: FindChoiceQuestionByID :one
SELECT * FROM choice_questions
WHERE choice_questions.id = $1;

-- name: FindTextQuestionByID :one
SELECT * FROM text_questions WHERE text_questions.id = $1;



-- Fully replace all options with arrays of new values.
-- Arrays passed are "zipped" together using unnest.
-- Will not work with other subquestion types thanks to the
-- foreign key constraints.
--
-----------------------------------------------------------
-- name: ReplaceOptionsForQuestion :exec

WITH deleted AS (
    DELETE FROM options
    WHERE options.question_id = $1
)
INSERT INTO options (
    question_id,
    value,
    image_url
) SELECT $1, unnest(@values::text[]), nullif(unnest(@image_urls::text[]), '') FROM
    questions INNER JOIN scripts
    ON questions.script_id = scripts.id
WHERE questions.id = $1
    AND scripts.id = questions.script_id
    AND scripts.locked = false;




-- Finds options for a question order by creation.
-- Used for obtaining options for a question for editing
--
--------------------------------------------------------
-- name: FindOptionsForQuestion :many
SELECT * FROM options
    WHERE options.question_id = $1
    ORDER BY options.created_at ASC;


-- Finds options for a question unordered. Used for
-- obtaining options for a question during an exam session
--
----------------------------------------------------------
-- name: FindOptionsForQuestionShuffled :many
SELECT * FROM options
    WHERE options.question_id = $1
    ORDER BY random();

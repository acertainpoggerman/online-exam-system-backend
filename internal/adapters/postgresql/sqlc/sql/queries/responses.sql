
-- Creates a response given a session ID
-- and an examinee ID. Will default to ENROLLED
-- and enrolled_at set to the current time of execution
--
-- name: CreateResponse :one

INSERT INTO responses (
    session_id, examinee_id
) VALUES ($1, $2) RETURNING *;


-- Finds responses for an examinee using the examinee's ID
--
-- name: ExamineeFindResponses :many

SELECT * FROM responses r
WHERE
    r.examinee_id = $1
AND r.status IN ('submitted', 'unreviewed', 'marked')
ORDER BY r.joined_at DESC;


-- Finds the marks for a response (must have MARKED status)
-- given session ID and examinee ID. Will return the mark's total
-- and maximum which represents: (total / maximum)
--
-- name: FindResponseMarks :one

SELECT
    sum(qr.mark) as total,
    sum(get_question_mark(qr.question_id)) as maximum
FROM
    question_responses qr JOIN responses r ON qr.response_id = r.id
WHERE
    r.session_id = $1
AND r.examinee_id = $2
AND r.status = 'marked'
LIMIT 1;


-- Find the response with matching session ID and examinee ID.
--
-- name: FindSingleResponse :one

SELECT * FROM responses r
WHERE
    r.session_id = $1
AND r.examinee_id = $2
LIMIT 1;


-- Finds the responses for a session given the
-- session ID. No extra information is added
--
-- name: FindSessionResponses :many

SELECT
    sqlc.embed(responses)
FROM
    responses JOIN users ON responses.examinee_id = users.id
WHERE responses.session_id = $1;


-- Finds the responses for a session given the
-- session ID, including each response's examinee information
--
-- name: FindSessionResponsesWithUser :many

SELECT
    sqlc.embed(responses),
    sqlc.embed(users)
FROM
    responses JOIN users ON responses.examinee_id = users.id
WHERE responses.session_id = $1;


-- Sets the questions for the submission of a user. Intended to
-- be used after an examiner starts the exam session before
-- responses are set as EDITABLE for examinees to answer
--
-- name: SetQuestionResponsesForSession :exec

WITH selected AS (
    SELECT
        responses.id as response_id,
        questions.id as question_id
    FROM
        scripts
            JOIN questions ON scripts.id = questions.script_id
            JOIN sessions ON sessions.script_id = scripts.id
            JOIN responses ON responses.session_id = sessions.id
    WHERE sessions.id = @session_id::uuid
    ORDER BY random()
)
INSERT INTO question_responses (
    response_id,
    question_id
) SELECT
    selected.response_id,
    selected.question_id
FROM selected;


-- Replaces the values (i.e. the answers) for
-- a question response.
--
-- name: ReplaceQuestionResponseValues :exec

WITH deleted AS (
    DELETE FROM response_values USING responses
    WHERE
        responses.id = response_values.response_id
    AND response.session_id = @session_id::uuid
    AND response.examinee_id = @examinee_id::uuid
    AND response_values.question_id = @question_id::uuid
    RETURNING 1
)
INSERT INTO response_values (
    response_id,
    question_id,
    value
) SELECT
    responses.id,
    @question_id::uuid,
    unnest(@value::text[])
FROM
    responses
        JOIN sessions ON responses.session_id = sessions.id
        CROSS JOIN (SELECT count(*) FROM deleted) AS dep -- Ensures CTE deletion goes through
WHERE
    responses.session_id = @session_id::uuid
AND responses.examinee_id = @examinee_id::uuid
AND responses.status = 'editable'
AND sessions.status = 'started';


-- Gets all the question responses for a response, along
-- with the response values (i.e. the response's answers) for that question.
--
-- name: FindQuestionResponses :many

SELECT
    sqlc.embed(qr),
    coalesce(
        array_agg(response_values.value) FILTER (WHERE response_values.value IS NOT NULL), '{}'
    )::text[] as value
FROM
    responses r
        JOIN question_responses qr ON qr.response_id = r.id
        LEFT JOIN response_values
            ON qr.question_id = response_values.question_id
            AND qr.response_id = response_values.response_id
WHERE
    r.session_id = @session_id::uuid
AND r.examinee_id = @examinee_id::uuid

GROUP BY qr.response_id, qr.question_id, qr.created_at
ORDER BY qr.created_at ASC;




-- Automatically gives a mark to the question response. Will
-- only give the mark as 0, or the maximum mark based
-- on the application layer's is_correct value.
--
-- If this fails for any of the question responses, then the whole
-- thing should fail, meaning that post marking of these
-- questions, all questions that can be automatically marked have been
-- marked.
--
-- name: AutoMarkQuestionResponse :one

WITH key_count AS (
    SELECT count(*) as value FROM answer_keys
    WHERE answer_keys.question_id = @question_id::uuid
)
UPDATE question_responses qr SET
    mark = CASE
        WHEN @is_correct::boolean = true THEN get_question_mark(@question_id::uuid)
        ELSE 0
    END
FROM responses r, key_count kc
WHERE
        r.id = qr.response_id
        AND r.session_id = @session_id::uuid
        AND r.examinee_id = @examinee_id::uuid
    AND qr.question_id = @question_id::uuid

    -- [X] Mark must be NULL prior
    -- [X] Submission must have a SUBMITTED status
    -- [X] Answer Key count is greater than 0

    AND qr.mark IS NULL
    AND r.status = 'submitted'
    AND kc.value > 0

RETURNING qr.*;


-- Sets the mark manually for an individual question response.
-- Should clamp the mark to (0, get_question_mark(question_id))
--
-- name: ExaminerMarkQuestionResponse :exec

WITH key_count AS (
    SELECT count(*) as value FROM answer_keys
    WHERE answer_keys.question_id = @question_id::uuid
)
UPDATE question_responses qr SET
    mark        = greatest(0, least(@mark::integer, get_question_mark(@question_id::uuid))),
    feedback    = @feedback::text
FROM responses r, key_count kc
WHERE
        r.id = qr.response_id
        AND r.session_id = @session_id::uuid
        AND r.examinee_id = @examinee_id::uuid
    AND qr.question_id = @question_id::uuid

    -- [X] Mark must be NULL prior
    -- [X] Submission must have an UNREVIEWED status
    -- [X] Answer Key count is equal to 0

    AND qr.mark IS NULL
    AND r.status = 'unreviewed'
    AND kc.value = 0

RETURNING qr.*;

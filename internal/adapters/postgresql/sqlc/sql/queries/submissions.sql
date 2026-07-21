-- name: CreateSubmission :one
INSERT INTO submissions (
    session_id, examinee_id
) VALUES ($1, $2) RETURNING *;

-- name: FindSubmissionsForSession :many
SELECT * FROM submissions WHERE submissions.session_id = $1;


-- name: ExamineeFindSubmitted :many
SELECT * FROM submissions s
WHERE s.examinee_id = $1
    AND s.status IN ('submitted', 'unreviewed', 'marked')
ORDER BY s.joined_at DESC;


-- name: SubmissionMarks :one
SELECT sum(sq.mark) as lhs, sum(get_question_mark(sq.question_id)) as rhs FROM
    submission_questions sq JOIN submissions s ON sq.submission_id = s.id
WHERE
        s.session_id = $1
    AND s.examinee_id = $2
    AND s.status = 'marked'
LIMIT 1;



-- Find submission with matching session ID and examinee ID.
--
-- name: FindSubmissionByID :one

SELECT * FROM submissions s
WHERE s.session_id = @session_id::uuid
    AND s.examinee_id = @examinee_id::uuid
LIMIT 1;


--
-- name: FindSubmssionsForSessionWithUser :many

SELECT sqlc.embed(submissions), sqlc.embed(users) FROM
    submissions INNER JOIN users ON submissions.examinee_id = users.id
WHERE submissions.session_id = $1;


-- Sets the questions for the submission of a user. Intended to
-- be used after an examiner starts the exam session before
-- submissions are set as EDITABLE for examinees to answer
--
-- name: SetQuestionsForSubmissions :exec

WITH selected AS (
    SELECT submissions.id as submission_id, questions.id as question_id FROM
        scripts INNER JOIN questions ON scripts.id = questions.script_id
        INNER JOIN sessions ON sessions.script_id = scripts.id
        INNER JOIN submissions ON submissions.session_id = sessions.id
    WHERE sessions.id = @session_id::uuid
    ORDER BY random()
)
INSERT INTO submission_questions (
    submission_id,
    question_id
) SELECT selected.submission_id, selected.question_id FROM selected;


-- Replaces the submission answer for a given question
--
-- name: ReplaceAnswerForQuestion :exec

WITH deleted AS (
    DELETE FROM answer_values
    WHERE answer_values.submission_id = $1
        AND answer_values.question_id = $2
    RETURNING 1
)
INSERT INTO answer_values (
    submission_id,
    question_id,
    value
) SELECT $1, $2, unnest(@value::text[]) FROM
    submissions INNER JOIN sessions ON submissions.session_id = sessions.id
    CROSS JOIN (SELECT count(*) FROM deleted) AS _dep
WHERE submissions.id = $1
    AND submissions.status = 'editable'
    AND sessions.status = 'started';


-- Gets the answers for a submission
--
-- name: FindSubmissionAnswers :many

SELECT
    sqlc.embed(sq),
    coalesce(array_agg(answer_values.value) FILTER (WHERE answer_values.value IS NOT NULL), '{}')::text[] as value
FROM
    submissions s INNER JOIN submission_questions sq ON sq.submission_id = s.id
    LEFT JOIN answer_values
        ON sq.question_id = answer_values.question_id
        AND sq.submission_id = answer_values.submission_id
WHERE s.session_id = @session_id
    AND s.examinee_id = @examinee_id
GROUP BY sq.submission_id, sq.question_id, sq.created_at
ORDER BY sq.created_at ASC;




-- Automatically gives a mark to the question. Will
-- only give the mark as 0, or the maximum mark based
-- on the application layer's is_correct value.
--
-- If this fails for any of the questions, then the whole
-- thing should fail, meaning that post marking of these
-- questions,
--
-- name: AutoMarkQuestion :one

WITH key_count AS (
    SELECT count(*) as value FROM answer_keys
    WHERE answer_keys.question_id = @question_id::uuid
)
UPDATE submission_questions sq SET
    mark = CASE
        WHEN @is_correct::boolean = true THEN get_question_mark(@question_id::uuid)
        ELSE 0
    END
FROM submissions s, key_count kc
WHERE
        s.id = sq.submission_id
    AND s.session_id = @session_id::uuid
    AND s.examinee_id = @examinee_id::uuid
    AND sq.question_id = @question_id::uuid

    -- [X] Mark must be NULL prior
    -- [X] Submission must have a SUBMITTED status
    -- [X] Answer Key count is greater than 0

    AND sq.mark IS NULL
    AND s.status = 'submitted'
    AND kc.value > 0

RETURNING sq.*;


-- Sets the mark manually for an individual question.
-- Should clamp the mark to (0, get_question_mark(question_id))
--
-- name: ExaminerMarkQuestion :exec

WITH key_count AS (
    SELECT count(*) as value FROM answer_keys
    WHERE answer_keys.question_id = @question_id::uuid
)
UPDATE submission_questions sq SET
    mark        = greatest(0, least(@mark::integer, get_question_mark(@question_id::uuid))),
    feedback    = @feedback::text
FROM submissions s, key_count kc
WHERE
        s.id = sq.submission_id
    AND s.session_id = @session_id::uuid
    AND s.examinee_id = @examinee_id::uuid
    AND sq.question_id = @question_id::uuid

    -- [X] Mark must be NULL prior
    -- [X] Submission must have an UNREVIEWED status
    -- [X] Answer Key count is equal to 0

    AND sq.mark IS NULL
    AND s.status = 'unreviewed'
    AND kc.value = 0

RETURNING sq.*;




-- name: GetQuestionMark :one

SELECT get_question_mark(@question_id::uuid)::integer;

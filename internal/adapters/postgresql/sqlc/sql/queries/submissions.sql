-- name: CreateSubmission :one
INSERT INTO submissions (
    session_id, examinee_id
) VALUES ($1, $2) RETURNING *;

-- name: FindSubmissionsForSession :many
SELECT * FROM submissions WHERE submissions.session_id = $1;

-- name: FindSubmissionByID :one
SELECT * FROM submissions WHERE submissions.id = $1;

-- name: UpdateSubmissionGrade :exec
UPDATE submissions SET grade = $2
    WHERE submissions.id = $1;


-- Set submissions as editable. Intended to be used
-- when the session has been started and the questions
-- for each user's submission has been set
--
------------------------------------------------------
-- name: SetSubmissionsEditableForSession :exec

UPDATE submissions SET
    status = 'editable'
WHERE submissions.session_id = $1;


-- Submits submission for a given session. Intended to be
-- used by the server to auto lock examinee submissions
-- when the session ends. Once submitted, the submission
-- can not be edited again.
--
---------------------------------------------------------
-- name: SubmitAllSubmissionsForSession :exec

UPDATE submissions SET
    status          = 'submitted',
    submitted_at    = now()
WHERE submissions.session_id = $1
    AND submissions.status != 'submitted';


-- Submits a single submission for an examinee. Intended to
-- be used by the examinee to submit their answers when done
--
------------------------------------------------------------
-- name: SubmitSubmission :one

UPDATE submissions SET
    status          = 'submitted',
    submitted_at    = now()
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
RETURNING *;


-- Sets the questions for the submission of a user. Intended to
-- be used after an examiner starts the exam session before
-- submissions are set as 'editable' for examinees to answer
--
----------------------------------------------------------------
-- name: SetQuestionsForSubmissions :exec

WITH selected AS (
    SELECT submissions.id as submission_id, questions.id as question_id FROM
        scripts INNER JOIN questions ON scripts.id = questions.script_id
        INNER JOIN sessions ON sessions.script_id = scripts.id
        INNER JOIN submissions ON submissions.session_id = sessions.id
    WHERE sessions.id = @session_id::uuid
    ORDER BY random()
) INSERT INTO submission_questions (
    submission_id,
    question_id
) SELECT selected.submission_id, selected.question_id FROM selected;


-- Replaces the submission answer for a given question
--
------------------------------------------------------
-- name: ReplaceSubAnswerForQuestion :exec

WITH deleted AS (
    DELETE FROM answer_values
    WHERE answer_values.submission_id = $1
        AND answer_values.question_id = $2
    RETURNING 1
) INSERT INTO answer_values (
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
------------------------------------
-- name: FindSubmissionAnswers :many

SELECT
    sq.question_id,
    coalesce(array_agg(answer_values.value) FILTER (WHERE answer_values.value IS NOT NULL), '{}')::text[] as value
FROM
    submissions INNER JOIN submission_questions sq ON sq.submission_id = submissions.id
    LEFT JOIN answer_values
        ON sq.question_id = answer_values.question_id
        AND sq.submission_id = answer_values.submission_id
WHERE submissions.id = $1
GROUP BY sq.question_id, sq.created_at
ORDER BY sq.created_at ASC;

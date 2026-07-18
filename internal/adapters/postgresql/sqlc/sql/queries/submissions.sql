-- name: CreateSubmission :one
INSERT INTO submissions (
    session_id, examinee_id
) VALUES ($1, $2) RETURNING *;

-- name: FindSubmissionsForSession :many
SELECT * FROM submissions WHERE submissions.session_id = $1;

-- name: FindSubmissionByID :one
SELECT * FROM submissions WHERE submissions.id = $1;


-- name: FindSubmssionsForSessionWithUser :many
SELECT sqlc.embed(submissions), sqlc.embed(users) FROM
    submissions INNER JOIN users ON submissions.examinee_id = users.id
WHERE submissions.session_id = $1;


-- Updates the mark for an answer to a question in
-- a submission. Can only update a submitted submission
--
-- name: SetSubmissionQuestionMark :one

UPDATE submission_questions SET
    mark = @mark::integer
FROM submissions
WHERE submission_questions.submission_id = $1
    AND submission_questions.question_id = $2
    AND submissions.id = submission_questions.submission_id
    AND submissions.status = 'submitted'
RETURNING submission_questions.*;


-- Calculates the mark for a submission based on
-- the individual marks for each question's answer.
--
-- name: CalculateSubmisionMark :one

WITH total AS (
    SELECT sum(sq.mark) AS mark FROM submission_questions sq
    WHERE sq.submission_id = @submission_id::uuid
)
UPDATE submissions SET
    mark    = total.mark,
    status  = 'marked'
FROM total
WHERE submissions.id = @submission_id::uuid
    AND submissions.status = 'submitted'
RETURNING total.mark;


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
-- name: ReplaceSubAnswerForQuestion :exec

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

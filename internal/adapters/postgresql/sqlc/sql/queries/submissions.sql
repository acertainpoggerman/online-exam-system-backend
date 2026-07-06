-- name: CreateSubmission :one
INSERT INTO submissions (
    session_id, examinee_id
) VALUES ($1, $2) RETURNING *;

-- name: FindSubmissionsForExaminee :many
SELECT * FROM submissions WHERE submissions.examinee_id = $1;

-- name: FindSubmissionsForSession :many
SELECT * FROM submissions WHERE submissions.session_id = $1;

-- name: FindSubmissionByID :one
SELECT * FROM submissions WHERE submissions.id = $1;

-- name: FindSubmissionByExamineeAndSession :one
SELECT * FROM submissions
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
LIMIT 1;

-- name: UpdateSubmissionGrade :exec
UPDATE submissions SET grade = $2
    WHERE submissions.id = $1;


-- name: SetSubmissionsEditableForSession :exec
UPDATE submissions SET
    status = 'editable'
WHERE submissions.session_id = $1;

-- name: SubmitSubmissionsForSession :exec
UPDATE submissions SET
    status          = 'submitted',
    submitted_at    = now()
WHERE submissions.session_id = $1;

-- name: SubmitSubmission :one
UPDATE submissions SET
    status          = 'submitted',
    submitted_at    = now()
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
RETURNING *;


-- name: FindActiveSubmissionForScript :one
SELECT submissions.* FROM
    submissions INNER JOIN sessions ON submissions.session_id = sessions.id
    INNER JOIN scripts ON sessions.script_id = scripts.id
WHERE scripts.id = $1
    AND submissions.examinee_id = $2
    AND submissions.status = 'editable'
    AND sessions.status = 'started';


-- name: CreateAnswerValue :exec
INSERT INTO answer_values (
    submission_id, question_id, value
) VALUES ($1, $2, $3) RETURNING value;

-- name: CreateAnswerValues :copyfrom
INSERT INTO answer_values (
    submission_id, question_id, value
) VALUES ($1, $2, $3);

-- name: FindAnswersForSubmission :many
SELECT answers.* FROM answers
    WHERE answers.submission_id = $1;

-- name: FindAnswerValuesForQuestion :many
SELECT answer_values.value FROM answer_values
    WHERE answer_values.submission_id = $1 AND question_id = $2;

-- name: DeleteAnswerValuesByIDs :execrows
DELETE FROM answer_values
    WHERE answer_values.submission_id = $1 AND answer_values.question_id = $2;

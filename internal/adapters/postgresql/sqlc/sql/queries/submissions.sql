-- name: CreateSubmission :one
INSERT INTO submissions (
    session_id, examinee_id
) VALUES ($1, $2) RETURNING id;

-- name: FindSubmissionsForExaminee :many
SELECT * FROM submissions WHERE submissions.examinee_id = $1;

-- name: FindSubmissionsForSession :many
SELECT * FROM submissions WHERE submissions.session_id = $1;

-- name: FindSubmissionByID :one
SELECT * FROM submissions WHERE submissions.id = $1;

-- name: UpdateSubmissionGrade :exec
UPDATE submissions SET grade = $2
    WHERE submissions.id = $1;


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

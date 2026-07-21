-- name: LogProctorEvent :one
INSERT INTO proctor_events (
    session_id,
    examinee_id,
    type,
    occurred_at
) VALUES ($1, $2, $3, $4)
RETURNING id;


-- name: FindProctorLogsForExaminee :many
SELECT * FROM proctor_events p
WHERE p.session_id = $1
    AND p.examinee_id = $2
ORDER BY received_at DESC;

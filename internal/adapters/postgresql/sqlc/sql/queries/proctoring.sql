-- name: LogProctorEvent :one
INSERT INTO proctor_events (
    session_id,
    examinee_id,
    type,
    occurred_at
) VALUES ($1, $2, $3, $4)
RETURNING id;

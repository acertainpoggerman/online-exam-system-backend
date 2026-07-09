
-- name: CreateSession :one
INSERT INTO sessions (
    creator_id,
    script_id,
    title,
    join_code
) VALUES ($1, $2, $3, $4) RETURNING id;



-- name: FindSessionsForExaminer :many
SELECT * FROM sessions
WHERE sessions.creator_id = $1;

-- name: FindSessionByID :one
SELECT * FROM sessions
WHERE sessions.id = $1 LIMIT 1;

-- name: FindSessionByJoinCode :one
SELECT * FROM sessions
WHERE sessions.join_code ILIKE $1
    AND sessions.status = 'open'
LIMIT 1;


-- Updates the top level fields of the sessions
--
-- name: UpdateSessionFields :one

UPDATE sessions SET
    title       = $2,
    script_id   = $3
WHERE sessions.id = $1
    AND sessions.status IN ('open', 'closed')
RETURNING *;

-- Deletes a session only if it is CLOSED
--
-- name: DeleteSessionByID :exec

DELETE FROM sessions
WHERE sessions.status = 'closed'
    AND sessions.id = $1;


-- Returns the submission ID if it's found
-- and is still in an active state (JOINED or EDITABLE)
--
-- name: FindActiveSubmissionInSession :one

SELECT submissions.id FROM submissions
WHERE submissions.session_id = $1
    AND submissions.examinee_id = $2
    AND submissions.status IN ('joined', 'editable');


-- Puts the session in OPEN mode. While in OPEN mode
-- examinees can join the session. Only sessions in
-- CLOSED mode can be opened.
--
-- name: OpenSession :one

UPDATE sessions SET
    status = 'open'
WHERE sessions.id = $1
    AND sessions.status = 'closed'
RETURNING *;


-- Puts the session in CLOSED mode (default mode). While in
-- CLOSED mode, examinees cannot join the session. Only
-- sessions in OPEN mode can be closed.
--
-- name: CloseSession :one

UPDATE sessions SET
    status = 'closed'
WHERE sessions.id = $1
    AND sessions.status = 'open'
RETURNING *;


-- Sets the session to STARTED mode. Only sessions in OPEN
-- mode can be started.
--
-- name: StartSession :one

WITH updated_session AS (
    UPDATE sessions SET
        status      = 'started',
        started_at  = now()
    WHERE sessions.id = $1
        AND sessions.status = 'open'
    RETURNING *
)
UPDATE scripts SET locked = true
FROM updated_session
WHERE scripts.id = updated_session.script_id
RETURNING updated_session.*;


-- Sets the session to ENDED mode. Only sessions in STARTED
-- mode can be started.
--
-- name: EndSession :one

UPDATE sessions SET
    status      = 'ended',
    ended_at    = now()
WHERE sessions.id = $1
    AND sessions.status = 'started'
RETURNING *;

--------------------------------------------------------------------------------
--- Changing Session Status ----------------------------------------------------
--------------------------------------------------------------------------------

-- The state machine below for switching session statuses

-------------------------------------------------------------------------
-- State machine :: (Start) : CLOSED <-> OPEN -> STARTED -> ENDED : (End)
-------------------------------------------------------------------------

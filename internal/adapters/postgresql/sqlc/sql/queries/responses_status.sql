
-- name: FindResponseStatus :one

SELECT responses.status FROM responses
WHERE responses.session_id = $1
AND responses.examinee_id = $2;


-- Returns the response ID if the examinee may open a websocket connection.
--
-- OPEN:     enrolled or left (waiting room entry / reconnect)
-- STARTED:  disconnected only (grace reconnect; flagged requires appeal first)
--
-- name: FindConnectableResponse :one

SELECT responses.id FROM
    responses INNER JOIN sessions on responses.session_id = sessions.id
    WHERE responses.session_id = $1
    AND responses.examinee_id = $2
    AND (
        (sessions.status = 'open' AND responses.status IN ('enrolled', 'left'))
        OR (sessions.status = 'started' AND responses.status = 'disconnected')
    );


-- Set the response status to a connected state based on current
-- response and session state
--
-- response: (ENROLLED, LEFT) session: OPEN       -> JOINED
-- response: (DISCONNECTED)   sesssion: STARTED   -> EDITABLE
--
-- name: SetResponseStatusOnConnect :one

UPDATE responses SET
    status = CASE
        WHEN sessions.status = 'open' THEN 'joined'::response_status
        WHEN sessions.status = 'started' THEN 'editable'::response_status
    END
FROM sessions
WHERE responses.session_id = sessions.id
    AND responses.session_id = $1
    AND responses.examinee_id = $2
    AND (
        (sessions.status = 'open' AND responses.status IN ('enrolled', 'left'))
        OR (sessions.status = 'started' AND responses.status = 'disconnected')
    )
RETURNING responses.*;


-- Set the response status to a disconnected state based
-- on current response and session state
--
-- response: (JOINED)     session: OPEN       -> LEFT
-- response: (EDITABLE)   sesssion: STARTED   -> DISCONNECTED
--
-- name: SetResponseStatusOnDisconnect :one

UPDATE responses SET
    status = CASE
        WHEN sessions.status = 'open' THEN 'left'::response_status
        WHEN sessions.status = 'started' THEN 'disconnected'::response_status
    END
FROM sessions
WHERE responses.session_id = sessions.id
    AND responses.session_id = $1
    AND responses.examinee_id = $2
    AND (
        (sessions.status = 'open' AND responses.status = 'joined')
        OR (sessions.status = 'started' AND responses.status = 'editable')
    )
RETURNING responses.*;


-- On session start: promote waiting-room examinees and lock out no-shows.
-- Use after the session is STARTED and response questions are assigned.
--
-- ENROLLED -> LEFT: enrolled but never connected during OPEN (see SetResponseJoined).
-- JOINED -> EDITABLE: was in the waiting room when the exam started.
--
-- Rows already LEFT (disconnected during OPEN) are unchanged and require
-- examiner readmit (SetResponseDisconnectedFromLeft) to participate.
--
-- name: SetResponseStatusesForStartedSession :execrows

WITH enrolled_to_left AS (
    UPDATE responses SET
        status = 'left'
    WHERE responses.session_id = $1
    AND responses.status = 'enrolled'
    RETURNING id
)
UPDATE responses SET
    status = 'editable'
WHERE responses.session_id = $1
AND responses.status = 'joined';


-- On session end: auto-submit all in-progress responses for the session.
-- Updates LEFT, EDITABLE, DISCONNECTED, and FLAGGED to SUBMITTED.
--
-- ENROLLED and JOINED are excluded (should not exist after session start;
-- see SetResponseStatusesForStartedSession). SUBMITTED and MARKED are terminal.
--
-- name: SubmitAllResponsesForSession :execrows

UPDATE responses SET
    status          = 'submitted',
    submitted_at    = now()
WHERE responses.session_id = $1
AND responses.status NOT IN ('submitted', 'marked', 'joined', 'enrolled');


-- Voluntary final submit by the examinee (EDITABLE -> SUBMITTED).
--
-- name: SetResponseSubmitted :one

UPDATE responses SET
    status          = 'submitted',
    submitted_at    = now()
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status = 'editable'
RETURNING *;


-- Connect during OPEN: first entry (ENROLLED -> JOINED)
-- or reconnect (LEFT -> JOINED).
--
-- name: SetResponseJoined :one

UPDATE responses SET
    status = 'joined'
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status IN ('enrolled', 'left')
RETURNING *;


-- Disconnect during OPEN (JOINED -> LEFT). Ensures session start does not
-- promote disconnected examinees to EDITABLE.
--
-- name: SetResponseLeftFromJoined :one

UPDATE responses SET
    status = 'left'
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status = 'joined'
RETURNING *;


-- Examiner readmit after missing the start window (LEFT -> DISCONNECTED).
-- Examinee reconnects with SetResponseEditableFromDisconnect.
--
-- name: SetResponseDisconnectedFromLeft :one

UPDATE responses SET
    status = 'disconnected'
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status = 'left'
RETURNING *;


-- Reconnect during STARTED after grace disconnect (DISCONNECTED -> EDITABLE).
--
-- name: SetResponseEditableFromDisconnect :one

UPDATE responses SET
    status = 'editable'
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status = 'disconnected'
RETURNING *;


-- Examiner grants appeal and client is connected in session
-- hub (FLAGGED -> EDITABLE).
--
-- name: SetResponseEditableFromFlagged :one

UPDATE responses SET
    status = 'editable'
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status = 'flagged'
RETURNING *;


-- Examiner grants appeal and client is disconnected in
-- session hub (FLAGGED -> DISCONNECTED).
--
-- name: SetResponseDisconnectedFromFlagged :one

UPDATE responses SET
    status = 'disconnected'
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status = 'flagged'
RETURNING *;



-- Disconnect during STARTED while still eligible to answer (EDITABLE -> DISCONNECTED).
--
-- name: SetResponseDisconnected :one

UPDATE responses SET
    status = 'disconnected'
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status IN ('editable')
RETURNING *;


-- Flag response when grace period has expired
-- (DISCONNECTED -> FLAGGED).
--
-- name: SetResponseFlaggedFromDisconnect :one

UPDATE responses SET
    status = 'flagged'
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status = 'disconnected'
RETURNING *;


-- Flag response when if server flags user as cheating
-- (EDITABLE -> FLAGGED).
--
-- name: SetResponseFlaggedFromEditable :one

UPDATE responses SET
    status = 'flagged'
WHERE responses.examinee_id = $1
AND responses.session_id = $2
AND responses.status = 'editable'
RETURNING *;






-- Set the status of the response post automatic marking.
-- If the response still has questions yet to be marked, THEN
-- set it as UNREVIEWED.
--
-- Should only be used after marking every individual question
-- that could be marked using AutoMarkQuestion().
--
-- (SUBMITTED -> UNREVIEWED)    : len(questions.mark == NULL) >= 1
-- (SUBMITTED -> MARKED)        : len(questions.mark == NULL) == 0
--
-- The examiner should handle manually marking questions for responses
-- post automatic marking that still have unresolved questions.
--
-- name: SetResponseStatusPostAutoMark :one

UPDATE responses r SET

    status = CASE
        -- If there is a question yet to have a mark, set it as unreviewed
        WHEN EXISTS (
            SELECT 1 FROM question_responses qr
            WHERE qr.response_id = r.id
                AND qr.mark IS NULL
        ) THEN 'unreviewed'::response_status
        -- If there is no question yet to have a mark, set as fully marked
        ELSE 'marked'::response_status
    END

WHERE
        r.session_id = @session_id::uuid
    AND r.examinee_id = @examinee_id::uuid
    AND r.status = 'submitted'
RETURNING *;


-- Will set the status only to marked if the count of
-- unmarked questions is 0. Should give error in application
-- layer if otherwise.
--
-- (UNREVIEWED -> MARKED)
--
-- name: SetResponseStatusPostManualMark :one

WITH unmarked_count AS  (
    SELECT count(*) AS value FROM
        responses r JOIN question_responses qr ON r.id = qr.response_id
    WHERE
            r.session_id = @session_id::uuid
        AND r.examinee_id = @examinee_id::uuid
        AND qr.mark IS NULL
)
UPDATE responses r SET
    status = 'marked'
FROM unmarked_count uc
WHERE
        r.session_id = @session_id::uuid
    AND r.examinee_id = @examinee_id::uuid
    AND r.status = 'unreviewed'
    AND uc.value = 0
RETURNING *;


-- name: DebugGetQuestionResponses :many
SELECT qr.mark, qr.feedback FROM
    responses r JOIN question_responses qr ON qr.response_id = r.id
WHERE
        r.session_id = @session_id::uuid
    AND r.examinee_id = @examinee_id::uuid;

-- name: DebugGetCount :one

SELECT count(*) AS value FROM
    responses r JOIN question_responses qr ON r.id = qr.response_id
WHERE
        r.session_id = @session_id::uuid
    AND r.examinee_id = @examinee_id::uuid
    AND qr.mark IS NULL;

-- name: DebugGetStatus :one

SELECT r.status FROM
    responses r JOIN question_responses qr ON r.id = qr.response_id
WHERE
        r.session_id = @session_id::uuid
    AND r.examinee_id = @examinee_id::uuid;

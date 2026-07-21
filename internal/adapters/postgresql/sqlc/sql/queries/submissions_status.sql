
-- name: FindSubmissionStatus :one

SELECT submissions.status FROM submissions
WHERE submissions.session_id = $1
    AND submissions.examinee_id = $2;


-- Returns the submission ID if the examinee may open a websocket connection.
--
-- OPEN:     enrolled or left (waiting room entry / reconnect)
-- STARTED:  disconnected only (grace reconnect; flagged requires appeal first)
--
-- name: FindConnectableSubmission :one

SELECT submissions.id FROM
    submissions INNER JOIN sessions on submissions.session_id = sessions.id
WHERE submissions.session_id = $1
    AND submissions.examinee_id = $2
    AND (
        (sessions.status = 'open' AND submissions.status IN ('enrolled', 'left'))
        OR (sessions.status = 'started' AND submissions.status = 'disconnected')
    );


-- Set the submission status to a connected state based on current
-- submission and session state
--
-- submission: (ENROLLED, LEFT) session: OPEN       -> JOINED
-- submission: (DISCONNECTED)   sesssion: STARTED   -> EDITABLE
--
-- name: SetSubmissionOnConnect :one

UPDATE submissions SET
    status = CASE
        WHEN sessions.status = 'open' THEN 'joined'::submission_status
        WHEN sessions.status = 'started' THEN 'editable'::submission_status
    END
FROM sessions
WHERE submissions.session_id = sessions.id
    AND submissions.session_id = $1
    AND submissions.examinee_id = $2
    AND (
        (sessions.status = 'open' AND submissions.status IN ('enrolled', 'left'))
        OR (sessions.status = 'started' AND submissions.status = 'disconnected')
    )
RETURNING submissions.*;


-- Set the submission status to a disconnected state based
-- on current submission and session state
--
-- submission: (JOINED)     session: OPEN       -> LEFT
-- submission: (EDITABLE)   sesssion: STARTED   -> DISCONNECTED
--
-- name: SetSubmissionOnDisconnect :one

UPDATE submissions SET
    status = CASE
        WHEN sessions.status = 'open' THEN 'left'::submission_status
        WHEN sessions.status = 'started' THEN 'disconnected'::submission_status
    END
FROM sessions
WHERE submissions.session_id = sessions.id
    AND submissions.session_id = $1
    AND submissions.examinee_id = $2
    AND (
        (sessions.status = 'open' AND submissions.status = 'joined')
        OR (sessions.status = 'started' AND submissions.status = 'editable')
    )
RETURNING submissions.*;


-- On session start: promote waiting-room examinees and lock out no-shows.
-- Use after the session is STARTED and submission questions are assigned.
--
-- ENROLLED -> LEFT: enrolled but never connected during OPEN (see SetSubmissionJoined).
-- JOINED -> EDITABLE: was in the waiting room when the exam started.
--
-- Rows already LEFT (disconnected during OPEN) are unchanged and require
-- examiner readmit (SetSubmissionDisconnectedFromLeft) to participate.
--
-- name: SetSubmissionStatusesForStartedSession :execrows

WITH enrolled_to_left AS (
    UPDATE submissions SET
        status = 'left'
    WHERE submissions.session_id = $1
        AND submissions.status = 'enrolled'
    RETURNING id
)
UPDATE submissions SET
    status = 'editable'
WHERE submissions.session_id = $1
    AND submissions.status = 'joined';


-- On session end: auto-submit all in-progress submissions for the session.
-- Updates LEFT, EDITABLE, DISCONNECTED, and FLAGGED to SUBMITTED.
--
-- ENROLLED and JOINED are excluded (should not exist after session start;
-- see SetSubmissionStatusesForStartedSession). SUBMITTED and MARKED are terminal.
--
-- name: SubmitAllSubmissionsForSession :execrows

UPDATE submissions SET
    status          = 'submitted',
    submitted_at    = now()
WHERE submissions.session_id = $1
    AND submissions.status NOT IN ('submitted', 'marked', 'joined', 'enrolled');


-- Voluntary final submit by the examinee (EDITABLE -> SUBMITTED).
--
-- name: SetSubmissionSubmitted :one

UPDATE submissions SET
    status          = 'submitted',
    submitted_at    = now()
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
    AND submissions.status = 'editable'
RETURNING *;


-- Connect during OPEN: first entry (ENROLLED -> JOINED)
-- or reconnect (LEFT -> JOINED).
--
-- name: SetSubmissionJoined :one

UPDATE submissions SET
    status = 'joined'
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
    AND submissions.status IN ('enrolled', 'left')
RETURNING *;


-- Disconnect during OPEN (JOINED -> LEFT). Ensures session start does not
-- promote disconnected examinees to EDITABLE.
--
-- name: SetSubmissionLeft :one

UPDATE submissions SET
    status = 'left'
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
    AND submissions.status = 'joined'
RETURNING *;


-- Examiner readmit after missing the start window (LEFT -> DISCONNECTED).
-- Examinee reconnects with SetSubmissionEditableFromDisconnect.
--
-- name: SetSubmissionDisconnectedFromLeft :one

UPDATE submissions SET
    status = 'disconnected'
WHERE submissions.examinee_id = $1
AND submissions.session_id = $2
AND submissions.status = 'left'
RETURNING *;


-- Reconnect during STARTED after grace disconnect (DISCONNECTED -> EDITABLE).
--
-- name: SetSubmissionEditableFromDisconnect :one

UPDATE submissions SET
    status = 'editable'
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
    AND submissions.status = 'disconnected'
RETURNING *;


-- Examiner grants appeal and client is connected in session
-- hub (FLAGGED -> EDITABLE).
--
-- name: SetSubmissionEditableFromFlagged :one

UPDATE submissions SET
    status = 'editable'
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
    AND submissions.status = 'flagged'
RETURNING *;


-- Examiner grants appeal and client is disconnected in
-- session hub (FLAGGED -> DISCONNECTED).
--
-- name: SetSubmissionDisconnectedFromFlagged :one

UPDATE submissions SET
    status = 'disconnected'
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
    AND submissions.status = 'flagged'
RETURNING *;



-- Disconnect during STARTED while still eligible to answer (EDITABLE -> DISCONNECTED).
--
-- name: SetSubmissionDisconnected :one

UPDATE submissions SET
    status = 'disconnected'
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
    AND submissions.status IN ('editable')
RETURNING *;


-- Flag submission when grace period has expired
-- (DISCONNECTED -> FLAGGED).
--
-- name: SetSubmissionFlaggedFromDisconnect :one

UPDATE submissions SET
    status = 'flagged'
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
    AND submissions.status = 'disconnected'
RETURNING *;


-- Flag submission when if server flags user as cheating
-- (EDITABLE -> FLAGGED).
--
-- name: SetSubmissionFlaggedFromEditable :one

UPDATE submissions SET
    status = 'flagged'
WHERE submissions.examinee_id = $1
    AND submissions.session_id = $2
    AND submissions.status = 'editable'
RETURNING *;






-- Set the status of the submission post automatic marking.
-- If the submission still has questions yet to be marked, THEN
-- set it as UNREVIEWED.
--
-- Should only be used after marking every individual question
-- that could be marked using AutoMarkQuestion().
--
-- (SUBMITTED -> UNREVIEWED)    : len(questions.mark == NULL) >= 1
-- (SUBMITTED -> MARKED)        : len(questions.mark == NULL) == 0
--
-- The examiner should handle manually marking questions for submissions
-- post automatic marking that still have unresolved questions.
--
-- name: SetSubmissionStatusPostAutoMark :one

UPDATE submissions s SET

    status = CASE
        -- If there is a question yet to have a mark, set it as unreviewed
        WHEN EXISTS (
            SELECT 1 FROM submission_questions sq
            WHERE sq.submission_id = s.id
                AND sq.mark IS NULL
        ) THEN 'unreviewed'::submission_status
        -- If there is no question yet to have a mark, set as fully marked
        ELSE 'marked'::submission_status
    END

WHERE
        s.session_id = @session_id::uuid
    AND s.examinee_id = @examinee_id::uuid
    AND s.status = 'submitted'
RETURNING *;


-- Will set the status only to marked if the count of
-- unmarked questions is 0. Should give error in application
-- layer if otherwise.
--
-- (UNREVIEWED -> MARKED)
--
-- name: SetSubmissionStatusPostManualMark :one

WITH unmarked_count AS  (
    SELECT count(*) AS value FROM
        submissions s JOIN submission_questions sq ON s.id = sq.submission_id
    WHERE
            s.session_id = @session_id::uuid
        AND s.examinee_id = @examinee_id::uuid
        AND sq.mark IS NULL
)
UPDATE submissions s SET
    status = 'marked'
FROM unmarked_count uc
WHERE
        s.session_id = @session_id::uuid
    AND s.examinee_id = @examinee_id::uuid
    AND s.status = 'unreviewed'
    AND uc.value = 0
RETURNING *;


-- name: DebugGetQuestionResponses :many
SELECT sq.mark, sq.feedback FROM
    submissions s JOIN submission_questions sq ON sq.submission_id = s.id
WHERE
        s.session_id = @session_id::uuid
    AND s.examinee_id = @examinee_id::uuid;

-- name: DebugGetCount :one

SELECT count(*) AS value FROM
    submissions s JOIN submission_questions sq ON s.id = sq.submission_id
WHERE
        s.session_id = @session_id::uuid
    AND s.examinee_id = @examinee_id::uuid
    AND sq.mark IS NULL;

-- name: DebugGetStatus :one

SELECT s.status FROM
    submissions s JOIN submission_questions sq ON s.id = sq.submission_id
WHERE
        s.session_id = @session_id::uuid
    AND s.examinee_id = @examinee_id::uuid;

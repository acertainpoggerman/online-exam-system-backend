
-- Creates a new script
--
-- name: CreateScript :one

INSERT INTO scripts (
    title,
    heading,
    description,
    creator_id
) VALUES ($1, $2, $3, $4) RETURNING *;


-- Updates the script if possible
--
-- name: UpdateScriptFields :one

UPDATE scripts SET
    title           = $2,
    heading         = $3,
    description     = $4,
    default_mark    = $5
WHERE scripts.id = $1
    AND scripts.locked = false
RETURNING id;


-- Deletes the script if possible
--
-- name: DeleteScript :exec

DELETE FROM scripts
WHERE scripts.id = $1
    AND scripts.locked = false;

-- Gets the scripts belonging to the examiner
-- with cursor pagination and a search query
--
-- name: FindScriptsForExaminer :many

SELECT * FROM scripts
WHERE scripts.creator_id = @examiner_id::UUID
    AND (@search::TEXT = '' OR scripts.title ILIKE '%' || @search::TEXT || '%')
    AND (scripts.last_modified_at, scripts.id) < (@cursor_ts::TIMESTAMPTZ, @cursor_id::UUID)
ORDER BY scripts.last_modified_at DESC, scripts.id DESC
LIMIT @page_size;


-- Finds the number of scripts belonging to the
-- examiner with the search query
--
-- name: FindScriptCountForExaminer :one

SELECT COUNT(*) FROM scripts
WHERE
    scripts.creator_id = @examiner_id::UUID
    AND (@search::TEXT = '' OR scripts.title ILIKE '%' || @search::TEXT || '%')
;


-- Finds a script by its ID
--
-- name: FindScriptByID :one

SELECT * FROM scripts
WHERE scripts.id = $1;


-- name: FindScriptForSubmission :one

SELECT scripts.* FROM
    scripts INNER JOIN sessions ON scripts.id = sessions.script_id
    INNER JOIN submissions ON submissions.session_id = sessions.id
WHERE submissions.id = @submission_id::uuid LIMIT 1;

--
-- name: ScriptTotalMarks :one

SELECT SUM(COALESCE(questions.mark, scripts.default_mark)) FROM
    scripts INNER JOIN questions ON scripts.id = questions.script_id
WHERE scripts.id = $1;

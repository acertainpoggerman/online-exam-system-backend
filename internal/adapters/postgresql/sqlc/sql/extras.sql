--------------------------------------------------------------------------------
--- FUNCTIONS ------------------------------------------------------------------
--------------------------------------------------------------------------------

-- Returns a trigger to set the last_modified_at of the table.

CREATE OR REPLACE FUNCTION set_last_modified_at()
RETURNS TRIGGER AS $$
BEGIN
  new.last_modified_at = now();
  RETURN new;
END;
$$ LANGUAGE plpgsql;

--------------------------------------------------------------------------------

-- Returns a trigger to set the last_modified_at of
-- the top level script.

CREATE OR REPLACE FUNCTION update_scripts_last_modified_at()
RETURNS TRIGGER AS $$
DECLARE
    v_script_id UUID;
BEGIN
    CASE (TG_TABLE_NAME)

    WHEN 'questions' THEN
        v_script_id := COALESCE(new.script_id, old.script_id);

    WHEN 'answer_keys' THEN
        SELECT script_id INTO v_script_id FROM questions
        WHERE id = coalesce(new.question_id, old.question_id);

    WHEN 'choice_questions' THEN
        SELECT script_id INTO v_script_id FROM questions
        WHERE id = coalesce(new.id, old.id);

    WHEN 'text_questions' THEN
        SELECT script_id INTO v_script_id FROM questions
        WHERE id = coalesce(new.id, old.id);

    WHEN 'options' THEN
        SELECT q.script_id INTO v_script_id FROM questions q
        WHERE q.id = coalesce(new.question_id, old.question_id);

    END CASE;

    UPDATE scripts SET
        last_modified_at = NOW()
    WHERE id = v_script_id;

    RETURN new;
END;
$$ LANGUAGE plpgsql;

--------------------------------------------------------------------------------

-- Deletes all subquestion occurences that have the same ID.
-- Intended to be used before a subquestion insert occurs.

CREATE OR REPLACE FUNCTION delete_all_subquestions(target_id UUID)
RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM text_questions WHERE id = target_id;
    DELETE FROM choice_questions WHERE id = target_id;
END;
$$;

--------------------------------------------------------------------------------

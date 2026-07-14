-- Creates new examiner. Intended to be used by administrators
-- to create user profiles for examiners (e.g. lecturers etc.).
-- In the future, ensure email is linked to valid domains e.g.
-- bazeuniversity.edu.ng
--
-- name: CreateExaminer :one

WITH
inserted_user AS (
    INSERT INTO users (
        first_name,
        last_name,
        email,
        password_hash,
        role
    ) VALUES ($1, $2, $3, $4, 'examiner')
    RETURNING *
),
inserted_examiner AS (
    INSERT INTO examiners (id)
    SELECT inserted_user.id FROM inserted_user
    RETURNING *
)
SELECT inserted_user.* FROM inserted_user;

-- Creates a new examinee. Intended to be used by students
-- to partake in examinations. In the future, ensure email
-- is linked to valid domains e.g. bazeuniversity.edu.ng
--
-- name: CreateExaminee :one

WITH
inserted_user AS (
    INSERT INTO users (
        first_name,
        last_name,
        email,
        password_hash,
        role
    ) VALUES ($1, $2, $3, $4, 'examinee')
    RETURNING *
),
inserted_examinee AS (
    INSERT INTO examinees (id)
    SELECT inserted_user.id FROM inserted_user
    RETURNING *
)
SELECT inserted_user.* FROM inserted_user;



-- name: FindUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: FindUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: FindExaminerByEmail :one
SELECT users.* FROM
    users INNER JOIN examiners ON users.id = examiners.id
WHERE email = $1 LIMIT 1;

-- name: FindExamineeByEmail :one
SELECT users.* FROM
    users INNER JOIN examinees ON users.id = examinees.id
WHERE email = $1 LIMIT 1;

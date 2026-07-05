-- name: CreateUser :one
INSERT INTO users (
    first_name, last_name, email, password_hash, role
) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: CreateExaminer :one
INSERT INTO examiners (id) VALUES ($1) RETURNING id;

-- name: CreateExaminee :one
INSERT INTO examinees (id) VALUES ($1) RETURNING id;

-- name: FindUserByID :one
SELECT * FROM
    users WHERE id = $1 LIMIT 1;

-- name: FindUserByEmail :one
SELECT * FROM
    users WHERE email = $1 LIMIT 1;

-- name: FindExaminerByEmail :one
SELECT users.* FROM
    users INNER JOIN examiners ON users.id = examiners.id
    WHERE email = $1 LIMIT 1;

-- name: FindExamineeByEmail :one
SELECT users.* FROM
    users INNER JOIN examinees ON users.id = examinees.id
    WHERE email = $1 LIMIT 1;

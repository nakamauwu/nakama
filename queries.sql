-- name: CreateUser :one
INSERT INTO users (id, email, username)
VALUES (@user_id, @email, @username)
RETURNING created_at;

-- name: UserByEmail :one
SELECT * FROM users WHERE email = @email;

-- name: UserExistsByEmail :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE email = @email
);

-- name: UserExistsByUsername :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE username ILIKE @username
);

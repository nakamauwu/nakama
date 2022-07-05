-- name: CreateUser :one
INSERT INTO users (id, email, username)
VALUES (@user_id, LOWER(@email), @username)
RETURNING created_at;

-- name: UserByEmail :one
SELECT * FROM users WHERE email = LOWER(@email);

-- name: UserExistsByEmail :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE email = LOWER(@email)
);

-- name: UserExistsByUsername :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE LOWER(username) = LOWER(@username)
);

-- name: CreatePost :one
INSERT INTO posts (id, user_id, content)
VALUES (@post_id, @user_id, @content)
RETURNING created_at;

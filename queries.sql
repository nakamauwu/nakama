-- name: CreateUser :one
INSERT INTO users (id, email, username)
VALUES (@user_id, LOWER(@email), @username)
RETURNING created_at;

-- name: UserByEmail :one
SELECT * FROM users WHERE email = LOWER(@email);

-- name: UserByUsername :one
SELECT * FROM users WHERE LOWER(username) = LOWER(@username);

-- name: UserExistsByEmail :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE email = LOWER(@email)
);

-- name: UserExistsByUsername :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE LOWER(username) = LOWER(@username)
);

-- name: UpdateUser :one
UPDATE users SET
    posts_count = posts_count + @increase_posts_count_by,
    updated_at = now()
WHERE id = @user_id
RETURNING updated_at;

-- name: CreatePost :one
INSERT INTO posts (id, user_id, content)
VALUES (@post_id, @user_id, @content)
RETURNING created_at;

-- name: Posts :many
SELECT posts.*, users.username
FROM posts
INNER JOIN users ON posts.user_id = users.id
WHERE
    CASE
        WHEN @username::varchar <> '' THEN LOWER(users.username) = LOWER(@username::varchar)
        ELSE true
    END
ORDER BY posts.id DESC;

-- name: Post :one
SELECT posts.*, users.username
FROM posts
INNER JOIN users ON posts.user_id = users.id
WHERE posts.id = @post_id;

-- name: PostExists :one
SELECT EXISTS (
    SELECT 1 FROM posts WHERE id = @post_id
);

-- name: UpdatePost :one
UPDATE posts
SET comments_count = comments_count + @increase_comments_count_by, updated_at = now()
WHERE id = @post_id
RETURNING updated_at;

-- name: CreateComment :one
INSERT INTO comments (id, user_id, post_id, content)
VALUES (@comment_id, @user_id, @post_id, @content)
RETURNING created_at;

-- name: Comments :many
SELECT comments.*, users.username
FROM comments
INNER JOIN users ON comments.user_id = users.id
WHERE comments.post_id = @post_id
ORDER BY comments.id DESC;

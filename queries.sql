-- name: CreateUser :one
INSERT INTO users (id, email, username)
VALUES (@user_id, LOWER(@email), @username)
RETURNING created_at;

-- name: UserByEmail :one
SELECT * FROM users WHERE email = LOWER(@email);

-- name: UserByUsername :one
SELECT users.*,
(
    CASE
        WHEN @follower_id::varchar <> '' THEN (
            SELECT EXISTS (
                SELECT 1 FROM user_follows
                WHERE follower_id = @follower_id::varchar
                AND followed_id = users.id
            )
        )
        ELSE false
    END
) AS following
FROM users
WHERE LOWER(username) = LOWER(@username);

-- name: UserExists :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE id = @user_id
);

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
    followers_count = followers_count + @increase_followers_count_by,
    following_count = following_count + @increase_following_count_by,
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

-- name: CreateUserFollow :one
INSERT INTO user_follows (follower_id, followed_id)
VALUES (@follower_id, @followed_id)
RETURNING created_at;

-- name: UserFollowExists :one
SELECT EXISTS (
    SELECT 1 FROM user_follows
    WHERE follower_id = @follower_id
    AND followed_id = @followed_id
);

-- name: DeleteUserFollow :one
DELETE FROM user_follows
WHERE follower_id = @follower_id
AND followed_id = @followed_id
RETURNING now()::timestamp AS deleted_at;

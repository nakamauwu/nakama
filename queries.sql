-- name: CreateUser :one
INSERT INTO users (id, email, username)
VALUES (@user_id, LOWER(@email), @username)
RETURNING created_at;

-- name: User :one
SELECT users.*,
(
    CASE
        WHEN @follower_id::varchar != '' THEN (
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
WHERE CASE
    WHEN @user_id::varchar != '' THEN users.id = @user_id::varchar
    WHEN @email::varchar != '' THEN users.email = LOWER(@email::varchar)
    WHEN @username::varchar != '' THEN LOWER(users.username) = LOWER(@username::varchar)
    ELSE false
END;



-- name: UserExists :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE CASE
        WHEN @user_id::varchar != '' THEN id = @user_id::varchar
        WHEN @email::varchar != '' THEN email = LOWER(@email::varchar)
        WHEN @username::varchar != '' THEN LOWER(username) = LOWER(@username::varchar)
        ELSE false
    END
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
        WHEN @username::varchar != '' THEN LOWER(users.username) = LOWER(@username::varchar)
        ELSE true
    END
ORDER BY posts.id DESC;

-- name: Post :one
SELECT posts.*, users.username
FROM posts
INNER JOIN users ON posts.user_id = users.id
WHERE posts.id = @post_id;

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

-- name: CreateHomeTimelineItem :one
INSERT INTO home_timeline (user_id, post_id)
VALUES (@user_id, @post_id)
RETURNING id, created_at;

-- name: FanoutHomeTimeline :many
INSERT INTO home_timeline (user_id, post_id)
SELECT user_follows.follower_id, @posts_id
FROM user_follows
WHERE user_follows.followed_id = @followed_id
ON CONFLICT (user_id, post_id) DO NOTHING
RETURNING id, created_at;

-- name: HomeTimeline :many
SELECT posts.*, users.username
FROM home_timeline
INNER JOIN posts ON home_timeline.post_id = posts.id
INNER JOIN users ON posts.user_id = users.id
WHERE home_timeline.user_id = @user_id
ORDER BY home_timeline.id DESC;

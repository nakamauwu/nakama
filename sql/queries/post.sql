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

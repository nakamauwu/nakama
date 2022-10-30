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

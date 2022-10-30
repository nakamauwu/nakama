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

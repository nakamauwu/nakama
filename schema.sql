-- CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- DROP DATABASE IF EXISTS nakama CASCADE;
CREATE DATABASE IF NOT EXISTS nakama;
SET DATABASE = nakama;

CREATE TABLE IF NOT EXISTS email_verification_codes (
    email VARCHAR NOT NULL,
    code UUID NOT NULL DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (email, code)
);

CREATE TABLE IF NOT EXISTS users (
    id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR NOT NULL UNIQUE,
    username VARCHAR NOT NULL UNIQUE,
    avatar VARCHAR,
    followers_count INT NOT NULL DEFAULT 0 CHECK (followers_count >= 0),
    followees_count INT NOT NULL DEFAULT 0 CHECK (followees_count >= 0)
);

CREATE TABLE IF NOT EXISTS webauthn_authenticators (
    id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    aaguid BYTES NOT NULL,
    sign_count INT NOT NULL,
    clone_warning BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    webauthn_authenticator_id UUID NOT NULL REFERENCES webauthn_authenticators,
    user_id UUID NOT NULL REFERENCES users,
    credential_id VARCHAR NOT NULL,
    public_key BYTES NOT NULL,
    attestation_type VARCHAR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE INDEX unique_webauthn_credentials (user_id, credential_id)
);

CREATE TABLE IF NOT EXISTS follows (
    follower_id UUID NOT NULL REFERENCES users,
    followee_id UUID NOT NULL REFERENCES users,
    PRIMARY KEY (follower_id, followee_id)
);

CREATE TABLE IF NOT EXISTS posts (
    id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users,
    content VARCHAR NOT NULL,
    spoiler_of VARCHAR,
    nsfw BOOLEAN NOT NULL DEFAULT false,
    likes_count INT NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    comments_count INT NOT NULL DEFAULT 0 CHECK (comments_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    INDEX sorted_posts (created_at DESC)
);

CREATE TABLE IF NOT EXISTS post_likes (
    user_id UUID NOT NULL REFERENCES users,
    post_id UUID NOT NULL REFERENCES posts,
    PRIMARY KEY (user_id, post_id)
);

CREATE TABLE IF NOT EXISTS post_subscriptions (
    user_id UUID NOT NULL REFERENCES users,
    post_id UUID NOT NULL REFERENCES posts,
    PRIMARY KEY (user_id, post_id)
);

CREATE TABLE IF NOT EXISTS timeline (
    id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users,
    post_id UUID NOT NULL REFERENCES posts,
    UNIQUE INDEX unique_timeline_items (user_id, post_id)
);

CREATE TABLE IF NOT EXISTS comments (
    id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users,
    post_id UUID NOT NULL REFERENCES posts,
    content VARCHAR NOT NULL,
    likes_count INT NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    INDEX sorted_comments (created_at DESC)
);

CREATE TABLE IF NOT EXISTS comment_likes (
    user_id UUID NOT NULL REFERENCES users,
    comment_id UUID NOT NULL REFERENCES comments,
    PRIMARY KEY (user_id, comment_id)
);

CREATE TABLE IF NOT EXISTS notifications (
    id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users,
    actors VARCHAR[] NOT NULL,
    type VARCHAR NOT NULL,
    post_id UUID REFERENCES posts,
    read_at TIMESTAMPTZ,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    INDEX sorted_notifications (issued_at DESC),
    UNIQUE INDEX unique_notifications (user_id, type, post_id, read_at)
);

ALTER TABLE webauthn_credentials DROP CONSTRAINT fk_webauthn_authenticator_id_ref_webauthn_authenticators;
ALTER TABLE webauthn_credentials ADD CONSTRAINT fk_webauthn_authenticator_id_ref_webauthn_authenticators FOREIGN KEY (webauthn_authenticator_id) REFERENCES webauthn_authenticators ON DELETE CASCADE;

ALTER TABLE webauthn_credentials DROP CONSTRAINT fk_user_id_ref_users;
ALTER TABLE webauthn_credentials ADD CONSTRAINT fk_user_id_ref_users FOREIGN KEY (user_id) REFERENCES users ON DELETE CASCADE;

ALTER TABLE follows DROP CONSTRAINT fk_follower_id_ref_users;
ALTER TABLE follows ADD CONSTRAINT fk_follower_id_ref_users FOREIGN KEY (follower_id) REFERENCES users ON DELETE CASCADE;

ALTER TABLE follows DROP CONSTRAINT fk_followee_id_ref_users;
ALTER TABLE follows ADD CONSTRAINT fk_followee_id_ref_users FOREIGN KEY (followee_id) REFERENCES users ON DELETE CASCADE;

ALTER TABLE posts DROP CONSTRAINT fk_user_id_ref_users;
ALTER TABLE posts ADD CONSTRAINT fk_user_id_ref_users FOREIGN KEY (user_id) REFERENCES users ON DELETE CASCADE;

ALTER TABLE post_likes DROP CONSTRAINT fk_user_id_ref_users;
ALTER TABLE post_likes ADD CONSTRAINT fk_user_id_ref_users FOREIGN KEY (user_id) REFERENCES users ON DELETE CASCADE;

ALTER TABLE post_likes DROP CONSTRAINT fk_post_id_ref_posts;
ALTER TABLE post_likes ADD CONSTRAINT fk_post_id_ref_posts FOREIGN KEY (post_id) REFERENCES posts ON DELETE CASCADE;

ALTER TABLE post_subscriptions DROP CONSTRAINT fk_user_id_ref_users;
ALTER TABLE post_subscriptions ADD CONSTRAINT fk_user_id_ref_users FOREIGN KEY (user_id) REFERENCES users ON DELETE CASCADE;

ALTER TABLE post_subscriptions DROP CONSTRAINT fk_post_id_ref_posts;
ALTER TABLE post_subscriptions ADD CONSTRAINT fk_post_id_ref_posts FOREIGN KEY (post_id) REFERENCES posts ON DELETE CASCADE;

ALTER TABLE timeline DROP CONSTRAINT fk_user_id_ref_users;
ALTER TABLE timeline ADD CONSTRAINT fk_user_id_ref_users FOREIGN KEY (user_id) REFERENCES users ON DELETE CASCADE;

ALTER TABLE timeline DROP CONSTRAINT fk_post_id_ref_posts;
ALTER TABLE timeline ADD CONSTRAINT fk_post_id_ref_posts FOREIGN KEY (post_id) REFERENCES posts ON DELETE CASCADE;

ALTER TABLE comments DROP CONSTRAINT fk_user_id_ref_users;
ALTER TABLE comments ADD CONSTRAINT fk_user_id_ref_users FOREIGN KEY (user_id) REFERENCES users ON DELETE CASCADE;

ALTER TABLE comments DROP CONSTRAINT fk_post_id_ref_posts;
ALTER TABLE comments ADD CONSTRAINT fk_post_id_ref_posts FOREIGN KEY (post_id) REFERENCES posts ON DELETE CASCADE;

ALTER TABLE comment_likes DROP CONSTRAINT fk_user_id_ref_users;
ALTER TABLE comment_likes ADD CONSTRAINT fk_user_id_ref_users FOREIGN KEY (user_id) REFERENCES users ON DELETE CASCADE;

ALTER TABLE comment_likes DROP CONSTRAINT fk_comment_id_ref_comments;
ALTER TABLE comment_likes ADD CONSTRAINT fk_comment_id_ref_comments FOREIGN KEY (comment_id) REFERENCES comments ON DELETE CASCADE;

ALTER TABLE notifications DROP CONSTRAINT fk_user_id_ref_users;
ALTER TABLE notifications ADD CONSTRAINT fk_user_id_ref_users FOREIGN KEY (user_id) REFERENCES users ON DELETE CASCADE;

-- INSERT INTO users (id, email, username) VALUES
--     ('24ca6ce6-b3e9-4276-a99a-45c77115cc9f', 'shinji@example.org', 'shinji'),
--     ('93dfcef9-0b45-46ae-933c-ea52fbf80edb', 'rei@example.org', 'rei');

-- INSERT INTO posts (id, user_id, content, comments_count) VALUES
--     ('c592451b-fdd2-430d-8d49-e75f058c3dce', '24ca6ce6-b3e9-4276-a99a-45c77115cc9f', 'sample post', 1);
-- INSERT INTO post_subscriptions (user_id, post_id) VALUES
--     ('24ca6ce6-b3e9-4276-a99a-45c77115cc9f', 'c592451b-fdd2-430d-8d49-e75f058c3dce');
-- INSERT INTO timeline (id, user_id, post_id) VALUES
--     ('d7490258-1f2f-4a75-8fbb-1846ccde9543', '24ca6ce6-b3e9-4276-a99a-45c77115cc9f', 'c592451b-fdd2-430d-8d49-e75f058c3dce');

-- INSERT INTO comments (id, user_id, post_id, content) VALUES
--     ('648e60bf-b0ab-42e6-8e48-10f797b19c49', '24ca6ce6-b3e9-4276-a99a-45c77115cc9f', 'c592451b-fdd2-430d-8d49-e75f058c3dce', 'sample comment');

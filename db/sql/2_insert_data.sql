INSERT INTO users (name, email) VALUES ('oji', 'sample+1@gmail');
INSERT INTO users (name, email) VALUES ('oba', 'sample+2@gmail');
INSERT INTO users (name, email) VALUES ('onii', 'sample+3@gmail');

INSERT INTO posts (text, user_id) VALUES ('初投稿いぇ〜', '1');

INSERT INTO comments (text, post_id, user_id) VALUES ('こんにちわ', 1, 2);
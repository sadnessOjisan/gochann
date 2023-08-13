CREATE DATABASE IF NOT EXISTS micro_post;

CREATE TABLE IF NOT EXISTS micro_post.users(
  `id` int(11) AUTO_INCREMENT,
  `name` varchar(12) NOT NULL,
  `email` text NOT NULL,
  `created_at` datetime,
  `updated_at` datetime,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS micro_post.posts(
  `id` int(11) AUTO_INCREMENT,
  `text` text NOT NULL,
  `user_id` int(11) NOT NULL,
  `created_at` datetime,
  `updated_at` datetime,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS micro_post.comments(
  `id` int(11) AUTO_INCREMENT,
  `text` text NOT NULL,
  `post_id` int(11) NOT NULL,
  `user_id` int(11) NOT NULL,
  `created_at` datetime,
  `updated_at` datetime,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
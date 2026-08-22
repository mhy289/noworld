-- 003_message.sql
-- 留言板：记录 ID 同时作为楼层号（自增），昵称 + 留言内容 + 时间
CREATE TABLE IF NOT EXISTS messages (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    nickname   VARCHAR(64) NOT NULL DEFAULT '',
    content    TEXT        NOT NULL,
    created_at DATETIME    NOT NULL,
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 001_init.sql
-- 初始化表结构迁移脚本，服务启动时由 store.InitDB 自动按序执行。
-- 使用 IF NOT EXISTS 保证幂等，可安全重复执行。

-- 投票统计表
CREATE TABLE IF NOT EXISTS votes (
    option_name VARCHAR(255) NOT NULL PRIMARY KEY,
    count       BIGINT       NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 后续新表 / 索引迁移请追加 002_xxx.sql、003_xxx.sql ...

-- 002_visitor.sql
-- 访客上报表：记录前端上报的访客 IP、来源域名、访问时间
CREATE TABLE IF NOT EXISTS visitor_reports (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    ip         VARCHAR(64)  NOT NULL DEFAULT '',
    domain     VARCHAR(255) NOT NULL DEFAULT '',
    visited_at DATETIME     NOT NULL,
    KEY idx_domain (domain),
    KEY idx_visited_at (visited_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

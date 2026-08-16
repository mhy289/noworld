package store

import (
	"fmt"
	"time"
)

// AddVisitorReport 记录一次访客访问（ip、来源域名、访问时间，持久化到 MySQL）。
func AddVisitorReport(ip, domain string, visitedAt time.Time) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	if len(ip) > 64 || len(domain) > 255 {
		return fmt.Errorf("invalid visitor data: field too long")
	}
	const ins = `INSERT INTO visitor_reports (ip, domain, visited_at) VALUES (?, ?, ?);`
	_, err := DB.Exec(ins, ip, domain, visitedAt.Format("2006-01-02 15:04:05"))
	return err
}

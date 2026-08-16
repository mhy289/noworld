// Package service 提供业务逻辑层。
package service

import (
	"time"

	"myworld-backend/internal/store"
)

// AddVisitorReport 记录一次访客访问（ip、来源域名、访问时间）。
func AddVisitorReport(ip, domain string, visitedAt time.Time) error {
	return store.AddVisitorReport(ip, domain, visitedAt)
}

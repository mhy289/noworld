// Package service 提供业务逻辑层。
package service

import (
	"myworld-backend/internal/store"
)

// AddVote 增加一次投票，key 为选项（持久化到 MySQL）。
func AddVote(key string) error {
	return store.AddVote(key)
}

// GetVotes 返回当前所有投票数据（来自 MySQL）。
func GetVotes() (map[string]int64, error) {
	return store.GetVotes()
}

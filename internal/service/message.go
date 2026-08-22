// Package service 提供业务逻辑层。
package service

import (
	"myworld-backend/internal/store"
)

// AddMessage 新增一条留言（昵称 + 内容），返回楼层号（记录 ID）。
func AddMessage(nickname, content string) (int64, error) {
	return store.AddMessage(nickname, content)
}

// GetMessages 返回全部留言（按楼层正序，来自 MySQL）。
func GetMessages() ([]store.Message, error) {
	return store.GetMessages()
}

// Package service 提供业务逻辑层。
package service

import "sync"

// 投票数据（内存存储，并发安全）
var (
	votes     = make(map[string]int)
	votesLock sync.Mutex
)

// AddVote 增加一次投票，key 为选项。
func AddVote(key string) {
	votesLock.Lock()
	defer votesLock.Unlock()
	votes[key]++
}

// GetVotes 返回当前所有投票数据的副本。
func GetVotes() map[string]int {
	votesLock.Lock()
	defer votesLock.Unlock()
	// 返回副本，避免外部修改内部状态
	copyVotes := make(map[string]int, len(votes))
	for k, v := range votes {
		copyVotes[k] = v
	}
	return copyVotes
}

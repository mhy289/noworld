package store

import (
	"fmt"
	"time"
)

// Message 留言记录，ID 同时作为楼层号（自增，永不复用）。
type Message struct {
	ID        int64  `json:"id"`
	Nickname  string `json:"nickname"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// AddMessage 新增一条留言，返回楼层号（即记录 ID）。
func AddMessage(nickname, content string) (int64, error) {
	if DB == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	if nickname == "" || len(nickname) > 64 {
		return 0, fmt.Errorf("invalid nickname: empty or too long (>64)")
	}
	if content == "" || len(content) > 2000 {
		return 0, fmt.Errorf("invalid content: empty or too long (>2000)")
	}
	const ins = `INSERT INTO messages (nickname, content, created_at) VALUES (?, ?, ?);`
	res, err := DB.Exec(ins, nickname, content, time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetMessages 按楼层（ID）正序返回全部留言。
func GetMessages() ([]Message, error) {
	messages := []Message{}
	if DB == nil {
		return messages, fmt.Errorf("database not initialized")
	}
	rows, err := DB.Query("SELECT id, nickname, content, created_at FROM messages ORDER BY id ASC")
	if err != nil {
		return messages, err
	}
	defer rows.Close()

	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Nickname, &m.Content, &m.CreatedAt); err != nil {
			return messages, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

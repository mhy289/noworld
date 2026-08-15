package store

import (
	"fmt"
)

// AddVote 为指定选项增加一次投票，若选项不存在则插入。
func AddVote(option string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	if option == "" || len(option) > 255 {
		return fmt.Errorf("invalid option: empty or too long (>255)")
	}
	const upsert = `
INSERT INTO votes (option_name, count) VALUES (?, 1)
ON DUPLICATE KEY UPDATE count = count + 1;`
	_, err := DB.Exec(upsert, option)
	return err
}

// GetVotes 返回所有选项的投票数映射。
func GetVotes() (map[string]int64, error) {
	result := make(map[string]int64)
	if DB == nil {
		return result, fmt.Errorf("database not initialized")
	}
	rows, err := DB.Query("SELECT option_name, count FROM votes")
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var cnt int64
		// count 列定义为 NOT NULL DEFAULT 0，可直接扫描为 int64
		if err := rows.Scan(&name, &cnt); err != nil {
			return result, err
		}
		result[name] = cnt
	}
	return result, rows.Err()
}

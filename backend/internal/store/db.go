// Package store 提供数据库访问层。
package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB 全局数据库连接池。
var DB *sql.DB

// InitDB 初始化 MySQL 连接池并执行迁移。
// 连接参数通过环境变量配置，未设置时使用默认值：
//
//	DB_HOST  默认 127.0.0.1
//	DB_PORT  默认 3306
//	DB_USER  默认 root
//	DB_PASS  默认空
//	DB_NAME  默认 myworld
func InitDB() error {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "root")
	pass := os.Getenv("DB_PASS")
	dbName := getEnv("DB_NAME", "myworld")

	// 1. 先以不带库名的方式连接，确保目标数据库存在（自动建库）
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true&charset=utf8mb4&loc=Local",
		user, pass, host, port)

	adminDB, err := sql.Open("mysql", rootDSN)
	if err != nil {
		return fmt.Errorf("open mysql(admin): %w", err)
	}
	defer adminDB.Close()
	if err := adminDB.Ping(); err != nil {
		return fmt.Errorf("ping mysql(%s:%s) failed: %w", host, port, err)
	}
	if _, err := adminDB.Exec("CREATE DATABASE IF NOT EXISTS `" + dbName + "` DEFAULT CHARACTER SET utf8mb4"); err != nil {
		return fmt.Errorf("create database %q: %w", dbName, err)
	}

	// 2. 切换到目标库建立正式连接池
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		user, pass, host, port, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// 校验连通性
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping mysql(%s:%s)/%s failed: %w", host, port, dbName, err)
	}

	DB = db
	log.Printf("[store] connected to MySQL %s:%s database=%s", host, port, dbName)

	if err := migrate(db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// migrate 建表迁移。
func migrate(db *sql.DB) error {
	stmt := `
CREATE TABLE IF NOT EXISTS votes (
    option_name VARCHAR(255) NOT NULL PRIMARY KEY,
    count       BIGINT       NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	_, err := db.Exec(stmt)
	return err
}

// CloseDB 关闭连接池（进程退出时调用）。
func CloseDB() {
	if DB != nil {
		_ = DB.Close()
	}
}

// PingDB 检测数据库当前是否可用，返回是否正常。
func PingDB() bool {
	if DB == nil {
		return false
	}
	return DB.Ping() == nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

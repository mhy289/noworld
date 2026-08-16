// Package store 提供数据库访问层。
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"myworld-backend/internal/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB 全局数据库连接池。
var DB *sql.DB

// InitDB 初始化 MySQL 连接池并执行迁移。
//
// 配置来源遵循「环境变量 > 配置文件 > 内置默认值」的优先级：
//   - 连接信息：DB_HOST / DB_PORT / DB_USER / DB_PASS / DB_NAME
//   - 连接池参数：DB_MAX_OPEN_CONNS / DB_MAX_IDLE_CONNS / DB_CONN_MAX_LIFETIME
//   - 可选 JSON 配置文件：通过 DB_CONFIG_FILE 指定路径（详见 config.go）
func InitDB() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load db config: %w", err)
	}

	host, port := cfg.Host, cfg.Port
	user, pass := cfg.User, cfg.Password
	dbName := cfg.Database

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
	// multiStatements=true 允许迁移脚本单文件包含多条 SQL 语句。
	// 仅用于内部执行的迁移/查询，所有用户输入均通过参数化占位符传递，不存在注入风险。
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local&multiStatements=true",
		user, pass, host, port, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 校验连通性
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping mysql(%s:%s)/%s failed: %w", host, port, dbName, err)
	}

	DB = db
	logger.Info("DB", "连接成功 %s:%s database=%s (池: %d/%d, 生命周期: %s)",
		host, port, dbName, cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)

	if err := migrate(db); err != nil {
		logger.Warn("DB", "建表迁移未执行: %v", err)
		return fmt.Errorf("migrate: %w", err)
	}
	logger.Info("DB", "数据库迁移全部完成")
	return nil
}

// migrate 执行建表迁移。
// 遍历 embed 嵌入的 migrations/*.sql 文件，按文件名升序逐个执行。
// 每个语句使用 IF NOT EXISTS 保证幂等，可安全重复执行。
func migrate(db *sql.DB) error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if len(e.Name()) >= 4 && e.Name()[len(e.Name())-4:] == ".sql" {
			names = append(names, e.Name())
		}
	}
	// 按文件名升序执行，保证迁移顺序确定（001_init.sql < 002_xxx.sql）
	sort.Strings(names)

	for _, name := range names {
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		// 按分号拆分为多条独立语句逐条执行，
		// 避免依赖 multiStatements，且可精确定位失败语句。
		stmts := splitStatements(string(sqlBytes))
		for _, stmt := range stmts {
			if _, err := db.Exec(stmt); err != nil {
				logger.Warn("DB", "迁移脚本 %s 执行失败: %v", name, err)
				return fmt.Errorf("execute migration %s: %w", name, err)
			}
		}
		logger.Info("DB", "已执行迁移脚本: %s (%d 条语句)", name, len(stmts))
	}
	return nil
}

// splitStatements 将迁移脚本按分号拆分为非空语句切片，并过滤纯注释/空行。
func splitStatements(script string) []string {
	var stmts []string
	var sb strings.Builder
	lines := strings.Split(script, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过空行和以 -- 开头的注释行
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		sb.WriteString(line)
		sb.WriteString("\n")
		// 语句以分号结束则收集
		if strings.HasSuffix(trimmed, ";") {
			stmts = append(stmts, sb.String())
			sb.Reset()
		}
	}
	// 末尾可能有无分号的残余（忽略）
	return stmts
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

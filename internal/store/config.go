// Package store 提供数据库访问层。
//
// config.go 统一管理数据库配置来源，支持「环境变量 + 可选配置文件」双层加载。
//
// 配置优先级（高 → 低）：
//  1. 环境变量        —— 部署时注入，拥有最高优先级，用于覆盖默认值与配置文件
//  2. 配置文件 (JSON) —— 可选，通过 DB_CONFIG_FILE 指定路径；集中管理非敏感参数
//  3. 内置默认值       —— 兜底，保证零配置也能启动
//
// 不引入任何第三方依赖，配置文件使用标准库 encoding/json 解析。
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"myworld-backend/internal/logger"
)

// Config 数据库全部可配置项。
type Config struct {
	Host            string        `json:"host"`
	Port            string        `json:"port"`
	User            string        `json:"user"`
	Password        string        `json:"password"`
	Database        string        `json:"database"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

// defaultConfig 返回内置默认配置（兜底，保证零配置可启动）。
func defaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            "3306",
		User:            "mhy",
		Password:        "",
		Database:        "myworld",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}
}

// loadConfig 按优先级加载配置：默认值 < 配置文件 < 环境变量。
// 返回最终生效的配置。
func loadConfig() (Config, error) {
	cfg := defaultConfig()

	// 1. 可选 JSON 配置文件（通过 DB_CONFIG_FILE 指定路径）
	if path := os.Getenv("DB_CONFIG_FILE"); path != "" {
		fileCfg, err := loadConfigFile(path)
		if err != nil {
			return cfg, fmt.Errorf("load config file: %w", err)
		}
		cfg = mergeConfig(cfg, fileCfg)
		logger.Info("DB", "已加载配置文件: %s", path)
	}

	// 2. 环境变量覆盖（最高优先级）
	cfg.Host = envOr("DB_HOST", cfg.Host)
	cfg.Port = envOr("DB_PORT", cfg.Port)
	cfg.User = envOr("DB_USER", cfg.User)
	cfg.Password = envOr("DB_PASS", cfg.Password)
	cfg.Database = envOr("DB_NAME", cfg.Database)

	if v := os.Getenv("DB_MAX_OPEN_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return cfg, fmt.Errorf("invalid DB_MAX_OPEN_CONNS=%q: must be positive integer", v)
		}
		cfg.MaxOpenConns = n
	}
	if v := os.Getenv("DB_MAX_IDLE_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return cfg, fmt.Errorf("invalid DB_MAX_IDLE_CONNS=%q: must be non-negative integer", v)
		}
		cfg.MaxIdleConns = n
	}
	if v := os.Getenv("DB_CONN_MAX_LIFETIME"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < 0 {
			return cfg, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME=%q: must be a duration like 1h, 30m", v)
		}
		cfg.ConnMaxLifetime = d
	}

	return cfg, nil
}

// loadConfigFile 读取并解析 JSON 配置文件。
// 仅覆盖配置文件中显式出现的字段；未出现的字段保持默认值。
func loadConfigFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	// 先解析为 map，仅提取显式声明的字段，避免「零值」与「未设置」混淆
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	var cfg Config
	if v, ok := raw["host"]; ok {
		_ = json.Unmarshal(v, &cfg.Host)
	}
	if v, ok := raw["port"]; ok {
		_ = json.Unmarshal(v, &cfg.Port)
	}
	if v, ok := raw["user"]; ok {
		_ = json.Unmarshal(v, &cfg.User)
	}
	if v, ok := raw["password"]; ok {
		_ = json.Unmarshal(v, &cfg.Password)
	}
	if v, ok := raw["database"]; ok {
		_ = json.Unmarshal(v, &cfg.Database)
	}
	if v, ok := raw["max_open_conns"]; ok {
		_ = json.Unmarshal(v, &cfg.MaxOpenConns)
	}
	if v, ok := raw["max_idle_conns"]; ok {
		_ = json.Unmarshal(v, &cfg.MaxIdleConns)
	}
	// conn_max_lifetime 在 JSON 中以秒数表示（便于人工书写），转为 Duration
	if v, ok := raw["conn_max_lifetime"]; ok {
		var secs int64
		_ = json.Unmarshal(v, &secs)
		cfg.ConnMaxLifetime = time.Duration(secs) * time.Second
	}
	return cfg, nil
}

// mergeConfig 将 file 中已显式设置的字段合并到 base 上。
// 通过区分「零值但确实被设置」与「未设置」的语义，这里约定：
//
//	数值字段以 >0 视为已设置（连接池参数不允许为负或零的非法场景），
//	因此零值一律视为未设置，交由默认值兜底。
func mergeConfig(base, file Config) Config {
	if file.Host != "" {
		base.Host = file.Host
	}
	if file.Port != "" {
		base.Port = file.Port
	}
	if file.User != "" {
		base.User = file.User
	}
	if file.Password != "" {
		base.Password = file.Password
	}
	if file.Database != "" {
		base.Database = file.Database
	}
	if file.MaxOpenConns > 0 {
		base.MaxOpenConns = file.MaxOpenConns
	}
	if file.MaxIdleConns > 0 {
		base.MaxIdleConns = file.MaxIdleConns
	}
	if file.ConnMaxLifetime > 0 {
		base.ConnMaxLifetime = file.ConnMaxLifetime
	}
	return base
}

// envOr 读取环境变量，为空时返回 fallback。
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

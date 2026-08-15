// Package logger 提供全局统一的控制台日志输出。
// 采用固定格式：时间 [级别] 模块: 消息，支持分级与颜色区分。
package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// Level 日志级别。
type Level int

const (
	// DEBUG 调试信息。
	DEBUG Level = iota
	// INFO 常规信息。
	INFO
	// WARN 警告信息。
	WARN
	// ERROR 错误信息。
	ERROR
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

// 颜色控制码。
var levelColors = map[Level]string{
	DEBUG: "\x1b[90m", // 灰色
	INFO:  "\x1b[36m", // 青色
	WARN:  "\x1b[33m", // 黄色
	ERROR: "\x1b[31m", // 红色
}

const (
	colorReset  = "\x1b[0m"
	colorDim    = "\x1b[90m"
	colorBold   = "\x1b[1m"
	colorModule = "\x1b[35m" // 紫色（模块标签）
	colorGreen  = "\x1b[32m" // 绿色
)

// minLevel 当前全局最低输出级别，可通过环境变量 LOG_LEVEL 覆盖。
var minLevel = INFO

func init() {
	if v := strings.ToUpper(os.Getenv("LOG_LEVEL")); v != "" {
		switch v {
		case "DEBUG":
			minLevel = DEBUG
		case "WARN":
			minLevel = WARN
		case "ERROR":
			minLevel = ERROR
		default:
			minLevel = INFO
		}
	}
}

// log 输出一条带级别与模块标签的日志。
func log(lv Level, module, format string, args ...interface{}) {
	if lv < minLevel {
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	levelTag := fmt.Sprintf("%s%-5s%s", levelColors[lv], levelNames[lv], colorReset)
	moduleTag := fmt.Sprintf("%s[%s]%s", colorModule, module, colorReset)
	// 时间戳部分使用暗色以突出级别与模块
	fmt.Fprintf(os.Stdout, "%s%s%s %s %s %s\n", colorDim, now, colorReset, levelTag, moduleTag, msg)
}

// Debug 输出调试日志。
func Debug(module, format string, args ...interface{}) {
	log(DEBUG, module, format, args...)
}

// Info 输出常规日志。
func Info(module, format string, args ...interface{}) {
	log(INFO, module, format, args...)
}

// Warn 输出警告日志。
func Warn(module, format string, args ...interface{}) {
	log(WARN, module, format, args...)
}

// Error 输出错误日志。
func Error(module, format string, args ...interface{}) {
	log(ERROR, module, format, args...)
}

// Section 输出一个醒目的区块分隔标题（用于流程阶段分隔）。
func Section(title string) {
	width := 56
	line := strings.Repeat("=", width)
	fmt.Fprintf(os.Stdout, "\n%s%s%s\n", colorBold, line, colorReset)
	fmt.Fprintf(os.Stdout, "%s %s %s\n", colorBold, title, colorReset)
	fmt.Fprintf(os.Stdout, "%s%s%s\n", colorBold, line, colorReset)
}

// OK 输出一条成功提示（绿色）。
func OK(module, format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, "%s[%s]%s %s[OK]%s %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		module,
		colorReset,
		colorGreen,
		colorReset,
		fmt.Sprintf(format, args...))
}

// Caller 返回当前调用位置（文件:行号），便于 DEBUG 定位。
func Caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	short := file
	if i := strings.LastIndex(file, "/"); i >= 0 {
		short = file[i+1:]
	}
	return fmt.Sprintf("%s:%d", short, line)
}

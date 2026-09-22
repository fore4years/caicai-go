package logger

import (
	"os"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rs/zerolog"
)

var Mylog zerolog.Logger

func init() {
	// 确保日志目录存在
	if err := os.MkdirAll("log", 0755); err != nil {
		// 目录创建失败则退化为仅控制台输出
		Mylog = zerolog.New(os.Stdout).With().Timestamp().Logger()
		return
	}

	// 按天滚动：每天一个文件 log/2026-09-22.log，零点切换，保留 30 天。
	rl, err := rotatelogs.New(
		"log/%Y-%m-%d.log",
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithClock(rotatelogs.Local),
		rotatelogs.WithMaxAge(30*24*time.Hour),
	)
	if err != nil {
		Mylog = zerolog.New(os.Stdout).With().Timestamp().Logger()
		return
	}
	defer rl.Close()

	// 控制台：彩色可读格式；文件：JSON 结构化（便于检索）。
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	}
	writer := zerolog.MultiLevelWriter(consoleWriter, rl)
	Mylog = zerolog.New(writer).With().Timestamp().Logger()
}

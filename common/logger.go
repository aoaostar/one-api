package common

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxLogCount = 1000000

var logCount int
var setupLogLock sync.Mutex
var setupLogWorking bool

func SetupLogger() {
	if *LogDir != "" {

		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FieldMap: log.FieldMap{
				log.FieldKeyTime:  "time",
				log.FieldKeyLevel: "level",
				log.FieldKeyMsg:   "message",
				log.FieldKeyFunc:  "caller",
			},
		})
		logPath := filepath.Join(*LogDir, "oneapi.log")

		logWriter := &lumberjack.Logger{
			Filename:   logPath, //日志文件位置
			MaxSize:    5,       // 单文件最大容量,单位是MB
			MaxBackups: 3,       // 最大保留过期文件个数
			MaxAge:     7,       // 保留过期文件的最大时间间隔,单位是天
			Compress:   false,   // 是否需要压缩滚动日志, 使用的 gzip 压缩
			LocalTime:  true,
		}
		ok := setupLogLock.TryLock()
		if !ok {
			log.Println("setup log is already working")
			return
		}
		defer func() {
			setupLogLock.Unlock()
			setupLogWorking = false
		}()

		log.SetOutput(io.MultiWriter(logWriter, os.Stdout))
		log.SetReportCaller(true)

		gin.DefaultWriter = os.Stdout
		gin.DefaultErrorWriter = log.StandardLogger().Writer()
	}
}

func SysLog(s interface{}) {
	log.WithFields(log.Fields{
		"service": "SYS",
	}).Info(s)
}

func SysError(s interface{}) {
	log.WithFields(log.Fields{
		"service": "SYS",
	}).Error(s)
}

func LogInfo(ctx context.Context, msg interface{}) {
	logHelper(ctx, log.InfoLevel, msg)
}

func LogWarn(ctx context.Context, msg interface{}) {
	logHelper(ctx, log.WarnLevel, msg)
}

func LogError(ctx context.Context, msg interface{}) {
	logHelper(ctx, log.ErrorLevel, msg)
}

func logHelper(ctx context.Context, level log.Level, msg interface{}) {

	id := ctx.Value(RequestIdKey)

	log.WithFields(log.Fields{
		"service":        fmt.Sprintf("OneAPI"),
		"request-id-key": id,
	}).Log(level, msg)

	logCount++ // we don't need accurate count, so no lock here
	if logCount > maxLogCount && !setupLogWorking {
		logCount = 0
		setupLogWorking = true
		go func() {
			SetupLogger()
		}()
	}
}

func FatalLog(v ...any) {
	t := time.Now()
	_, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[FATAL] %v | %v \n", t.Format("2006/01/02 - 15:04:05"), v)
	os.Exit(1)
}

func LogQuota(quota int) string {
	if DisplayInCurrencyEnabled {
		return fmt.Sprintf("＄%.6f 额度", float64(quota)/QuotaPerUnit)
	} else {
		return fmt.Sprintf("%d 点额度", quota)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/middleware"
	"caicai-go/model"
	"caicai-go/objects"
	"caicai-go/router"
	"caicai-go/service"
)

var serverPort = ":" + strconv.Itoa(int(conf.Cfg.App.Port))

func main() {
	// 初始化数据库连接
	if err := conf.Dbconnection(conf.Cfg.Mysql); err != nil {
		logger.Mylog.Fatal().Err(err).Msg("数据库初始化失败")
	}
	// 初始化 gorm gen 查询对象
	objects.SetDefault(conf.Db)
	if conf.Cfg.App.Migrate {
		if err := model.Migrate(conf.Db); err != nil {
			logger.Mylog.Fatal().Err(err).Msg("数据库迁移失败")
		}
		logger.Mylog.Info().Msg("AutoMigrate 已执行")
	} else {
		logger.Mylog.Info().Msg("跳过 AutoMigrate")
	}

	// 初始化设备通信基础设施（MQTT 下发 + 订阅上行主题 + Redis）
	service.InitDevice(conf.Cfg.Mqtt, conf.Cfg.Redis)

	// 启动 TCP 设备服务器（兼容仍使用自定义二进制协议的老设备）
	if err := service.StartTCPServer(int(conf.Cfg.Tcp.Port)); err != nil {
		logger.Mylog.Fatal().Err(err).Msg("TCP 设备服务器启动失败")
	}

	// 启动延迟任务分发器（替代 RabbitMQ）
	service.InitDispatcher()

	// 启动定时任务（订单超时取消/余额不足断电/退款同步/预约续期）
	schedCtx, stopSched := context.WithCancel(context.Background())
	service.StartScheduledTasks(schedCtx)

	gin.DefaultWriter = io.Discard
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	if gin.Mode() != gin.ReleaseMode {
		r.Use(middleware.MiddleLog(logger.Mylog))
	}
	r.Use(middleware.Auth())

	router.RouterInit(r)

	// 优雅关闭：捕获 SIGINT/SIGTERM，等待在途请求处理完后退出（对齐审计 CR-003）。
	srv := &http.Server{Addr: serverPort, Handler: r}

	go func() {
		logger.Mylog.Info().Msg(fmt.Sprintf("应用启动成功, 监听端口: %d", conf.Cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Mylog.Fatal().Err(err).Msg("HTTP 服务启动失败")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Mylog.Info().Msg("收到退出信号，开始优雅关闭...")

	stopSched()
	service.StopDispatcher()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Mylog.Error().Err(err).Msg("HTTP 服务优雅关闭失败")
	}
	logger.Mylog.Info().Msg("HTTP 服务已优雅关闭")
}
